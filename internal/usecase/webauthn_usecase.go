package usecase

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type WebAuthnUseCase struct {
	userRepo     ports.UserRepository
	webauthnRepo ports.WebAuthnRepository
	sessionStore ports.WebAuthnSessionStore
	authUseCase  *AuthUseCase // Para generar tokens tras login exitoso
	webAuthn     *webauthn.WebAuthn
}

func NewWebAuthnUseCase(
	userRepo ports.UserRepository,
	webauthnRepo ports.WebAuthnRepository,
	sessionStore ports.WebAuthnSessionStore,
	authUseCase *AuthUseCase,
	cfg *config.Config,
) (*WebAuthnUseCase, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.WebAuthn.RPID,
		RPDisplayName: cfg.WebAuthn.RPDisplayName,
		RPOrigins:     cfg.WebAuthn.RPOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize webauthn: %w", err)
	}

	return &WebAuthnUseCase{
		userRepo:     userRepo,
		webauthnRepo: webauthnRepo,
		sessionStore: sessionStore,
		authUseCase:  authUseCase,
		webAuthn:     w,
	}, nil
}

// User Wrapper para cumplir con webauthn.User
type webauthnUser struct {
	user        *domain.User
	credentials []*domain.WebAuthnCredential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return u.user.WebAuthnID
}

func (u *webauthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.user.Username
}

func (u *webauthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	res := make([]webauthn.Credential, len(u.credentials))
	for i, cred := range u.credentials {
		res[i] = webauthn.Credential{
			ID:              cred.ID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    cred.AAGUID,
				SignCount: cred.SignCount,
			},
		}
	}
	return res
}

func (uc *WebAuthnUseCase) BeginRegistration(ctx context.Context, tenantID, userID string) (*protocol.CredentialCreation, error) {
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Si no tiene WebAuthnID, generarlo
	if len(user.WebAuthnID) == 0 {
		user.WebAuthnID = []byte(user.ID) // Usar el ID de usuario como base o generar uno random
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	creds, err := uc.webauthnRepo.GetCredentialsByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	wUser := &webauthnUser{user: user, credentials: creds}

	options, session, err := uc.webAuthn.BeginRegistration(wUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	// Guardar sesión en Redis
	sessionData := &domain.WebAuthnSessionData{
		Challenge:        session.Challenge,
		UserID:           userID,
		ExpiresAt:        time.Now().Add(5 * time.Minute),
		UserVerification: string(session.UserVerification),
	}

	if err := uc.sessionStore.SaveWebAuthnSession(ctx, session.Challenge, sessionData, 5*time.Minute); err != nil {
		return nil, err
	}

	return options, nil
}

func (uc *WebAuthnUseCase) FinishRegistration(ctx context.Context, tenantID, userID string, challenge string, r *http.Request) error {
	user, err := uc.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	sessionData, err := uc.sessionStore.GetWebAuthnSession(ctx, challenge)
	if err != nil {
		return fmt.Errorf("session not found or expired: %w", err)
	}

	if sessionData.UserID != userID {
		return fmt.Errorf("invalid user for session")
	}

	wUser := &webauthnUser{user: user}
	session := webauthn.SessionData{
		Challenge:        sessionData.Challenge,
		UserID:           []byte(sessionData.UserID),
		UserVerification: protocol.UserVerificationRequirement(sessionData.UserVerification),
	}

	credential, err := uc.webAuthn.FinishRegistration(wUser, session, r)
	if err != nil {
		return fmt.Errorf("failed to finish registration: %w", err)
	}

	// Persistir credencial
	dbCred := &domain.WebAuthnCredential{
		ID:              credential.ID,
		UserID:          userID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.webauthnRepo.CreateCredential(ctx, tenantID, dbCred); err != nil {
		return err
	}

	// Limpiar sesión
	_ = uc.sessionStore.DeleteWebAuthnSession(ctx, sessionData.Challenge)

	return nil
}

func (uc *WebAuthnUseCase) BeginLogin(ctx context.Context, tenantID, identifier string) (*protocol.CredentialAssertion, error) {
	user, err := uc.userRepo.GetByEmailOrUsername(ctx, tenantID, identifier)
	if err != nil {
		return nil, err
	}

	creds, err := uc.webauthnRepo.GetCredentialsByUserID(ctx, tenantID, user.ID)
	if err != nil {
		return nil, err
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("no security keys registered for this user")
	}

	wUser := &webauthnUser{user: user, credentials: creds}

	options, session, err := uc.webAuthn.BeginLogin(wUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	// Guardar sesión en Redis
	sessionData := &domain.WebAuthnSessionData{
		Challenge:        session.Challenge,
		UserID:           user.ID,
		ExpiresAt:        time.Now().Add(5 * time.Minute),
		UserVerification: string(session.UserVerification),
	}

	if err := uc.sessionStore.SaveWebAuthnSession(ctx, session.Challenge, sessionData, 5*time.Minute); err != nil {
		return nil, err
	}

	return options, nil
}

func (uc *WebAuthnUseCase) FinishLogin(ctx context.Context, tenantID, identifier string, challenge string, r *http.Request, loginInput PasswordlessLoginInput) (*LoginResponse, error) {
	user, err := uc.userRepo.GetByEmailOrUsername(ctx, tenantID, identifier)
	if err != nil {
		return nil, err
	}

	sessionData, err := uc.sessionStore.GetWebAuthnSession(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired: %w", err)
	}

	if sessionData.UserID != user.ID {
		return nil, fmt.Errorf("invalid user for session")
	}

	creds, err := uc.webauthnRepo.GetCredentialsByUserID(ctx, tenantID, user.ID)
	if err != nil {
		return nil, err
	}

	wUser := &webauthnUser{user: user, credentials: creds}
	session := webauthn.SessionData{
		Challenge:        sessionData.Challenge,
		UserID:           []byte(sessionData.UserID),
		UserVerification: protocol.UserVerificationRequirement(sessionData.UserVerification),
	}

	credential, err := uc.webAuthn.FinishLogin(wUser, session, r)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %w", err)
	}

	// Actualizar sign count y detectar posibles clones
	for _, c := range creds {
		if string(c.ID) == string(credential.ID) {
			if credential.Authenticator.SignCount <= c.SignCount && c.SignCount > 0 {
				c.CloneWarning = true
				// Aquí podrías bloquear la cuenta o alertar
			}
			c.SignCount = credential.Authenticator.SignCount
			c.UpdatedAt = time.Now()
			if err := uc.webauthnRepo.UpdateCredential(ctx, tenantID, c); err != nil {
				return nil, err
			}
			break
		}
	}

	// Login exitoso -> Generar token usando AuthUseCase
	// Necesitamos un método en AuthUseCase que genere el token para un usuario ya autenticado
	// o usar directamente el JWTService si tenemos acceso.
	// Por ahora asumiremos que podemos llamar a un método interno o similar.

	// Limpiar sesión de desafío
	_ = uc.sessionStore.DeleteWebAuthnSession(ctx, sessionData.Challenge)

	return uc.authUseCase.PasswordlessLogin(ctx, user, loginInput)
}
