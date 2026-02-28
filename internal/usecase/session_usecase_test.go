package usecase

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/tests/mocks"
)

type SessionUseCaseMocks struct {
	uc          *SessionUseCase
	sessionRepo *mocks.MockSessionRepository
	userRepo    *mocks.MockUserRepository
}

func setupSessionUseCase(_ *testing.T) SessionUseCaseMocks {
	sessionRepo := new(mocks.MockSessionRepository)
	userRepo := new(mocks.MockUserRepository)
	logger := zerolog.New(os.Stdout).Level(zerolog.Disabled)

	uc := NewSessionUseCase(sessionRepo, userRepo, logger)

	return SessionUseCaseMocks{
		uc:          uc,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// TestListUserSessions_Success prueba el listado exitoso de sesiones
func TestListUserSessions_Success(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	currentJTI := uuid.Must(uuid.NewV7()).String()
	otherJTI := uuid.Must(uuid.NewV7()).String()

	sessions := []*domain.Session{
		{
			ID:        uuid.Must(uuid.NewV7()).String(),
			TenantID:  tenantID,
			UserID:    userID,
			JTI:       currentJTI,
			Device:    "Desktop",
			IPAddress: "192.168.1.1",
			Country:   "US",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		},
		{
			ID:        uuid.Must(uuid.NewV7()).String(),
			TenantID:  tenantID,
			UserID:    userID,
			JTI:       otherJTI,
			Device:    "Mobile",
			IPAddress: "192.168.1.2",
			Country:   "US",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			ExpiresAt: time.Now().Add(23 * time.Hour),
			Revoked:   false,
		},
	}

	// Mock expectations
	m.sessionRepo.On("GetByUserID", ctx, tenantID, userID).Return(sessions, nil)

	// Execute
	result, err := m.uc.ListUserSessions(ctx, tenantID, userID, currentJTI)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Verificar que la sesión actual está marcada
	var currentSessionMarked bool
	for _, session := range result {
		if session.JTI == currentJTI {
			assert.True(t, session.IsCurrent, "La sesión actual debe estar marcada")
			currentSessionMarked = true
		} else {
			assert.False(t, session.IsCurrent, "Las otras sesiones no deben estar marcadas")
		}
	}
	assert.True(t, currentSessionMarked, "Debe haber una sesión marcada como actual")

	m.sessionRepo.AssertExpectations(t)
}

// TestListUserSessions_RepositoryError prueba error al listar sesiones
func TestListUserSessions_RepositoryError(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	currentJTI := uuid.Must(uuid.NewV7()).String()

	// Mock expectations
	m.sessionRepo.On("GetByUserID", ctx, tenantID, userID).Return(nil, domain.ErrSessionExpired)

	// Execute
	result, err := m.uc.ListUserSessions(ctx, tenantID, userID, currentJTI)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionExpired, err)
	assert.Nil(t, result)

	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeSession_Success prueba la revocación exitosa de una sesión
func TestRevokeSession_Success(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	sessionID := uuid.Must(uuid.NewV7()).String()

	session := &domain.Session{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    userID,
		JTI:       uuid.Must(uuid.NewV7()).String(),
		Device:    "Desktop",
		IPAddress: "192.168.1.1",
		Revoked:   false,
	}

	// Mock expectations
	m.sessionRepo.On("GetByID", ctx, tenantID, sessionID).Return(session, nil)
	m.sessionRepo.On("Revoke", ctx, tenantID, sessionID, "user", "revoked by user").Return(nil)

	// Execute
	err := m.uc.RevokeSession(ctx, tenantID, userID, sessionID)

	// Assert
	assert.NoError(t, err)
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeSession_SessionNotFound prueba error cuando la sesión no existe
func TestRevokeSession_SessionNotFound(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	sessionID := uuid.Must(uuid.NewV7()).String()

	// Mock expectations
	m.sessionRepo.On("GetByID", ctx, tenantID, sessionID).Return(nil, domain.ErrSessionNotFound)

	// Execute
	err := m.uc.RevokeSession(ctx, tenantID, userID, sessionID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionNotFound, err)
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeSession_WrongUser prueba que un usuario no puede revocar la sesión de otro
func TestRevokeSession_WrongUser(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	otherUserID := uuid.Must(uuid.NewV7()).String()
	sessionID := uuid.Must(uuid.NewV7()).String()

	session := &domain.Session{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    otherUserID, // Sesión pertenece a otro usuario
		JTI:       uuid.Must(uuid.NewV7()).String(),
		Device:    "Desktop",
		IPAddress: "192.168.1.1",
		Revoked:   false,
	}

	// Mock expectations
	m.sessionRepo.On("GetByID", ctx, tenantID, sessionID).Return(session, nil)

	// Execute
	err := m.uc.RevokeSession(ctx, tenantID, userID, sessionID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionNotFound, err)

	// No debe llamar a Revoke
	m.sessionRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeSession_DatabaseError prueba error genérico al obtener la sesión
func TestRevokeSession_DatabaseError(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	sessionID := uuid.Must(uuid.NewV7()).String()

	// Mock expectations - error genérico de base de datos
	m.sessionRepo.On("GetByID", ctx, tenantID, sessionID).Return(nil, fmt.Errorf("database connection error"))

	// Execute
	err := m.uc.RevokeSession(ctx, tenantID, userID, sessionID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")

	// No debe llamar a Revoke
	m.sessionRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeSession_RevokeError prueba error al revocar la sesión
func TestRevokeSession_RevokeError(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	sessionID := uuid.Must(uuid.NewV7()).String()

	session := &domain.Session{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    userID,
		JTI:       uuid.Must(uuid.NewV7()).String(),
		Device:    "Desktop",
		IPAddress: "192.168.1.1",
		Revoked:   false,
	}

	// Mock expectations
	m.sessionRepo.On("GetByID", ctx, tenantID, sessionID).Return(session, nil)
	m.sessionRepo.On("Revoke", ctx, tenantID, sessionID, "user", "revoked by user").Return(fmt.Errorf("revoke failed"))

	// Execute
	err := m.uc.RevokeSession(ctx, tenantID, userID, sessionID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoke failed")
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeAllSessions_Success prueba la revocación de todas las sesiones excepto la actual
func TestRevokeAllSessions_Success(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	currentJTI := uuid.Must(uuid.NewV7()).String()

	sessions := []*domain.Session{
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      currentJTI, // Sesión actual - no debe ser revocada
			Device:   "Desktop",
			Revoked:  false,
		},
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      uuid.Must(uuid.NewV7()).String(),
			Device:   "Mobile",
			Revoked:  false,
		},
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      uuid.Must(uuid.NewV7()).String(),
			Device:   "Tablet",
			Revoked:  false,
		},
	}

	// Mock expectations
	m.sessionRepo.On("GetByUserID", ctx, tenantID, userID).Return(sessions, nil)

	// Solo debe revocar las 2 sesiones que no son la actual
	m.sessionRepo.On("Revoke", ctx, tenantID, sessions[1].ID, "user", "revoked all sessions").Return(nil).Once()
	m.sessionRepo.On("Revoke", ctx, tenantID, sessions[2].ID, "user", "revoked all sessions").Return(nil).Once()

	// Execute
	count, err := m.uc.RevokeAllSessions(ctx, tenantID, userID, currentJTI)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "Debe revocar 2 sesiones (todas excepto la actual)")
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeAllSessions_OnlyCurrentSession prueba cuando solo hay una sesión (la actual)
func TestRevokeAllSessions_OnlyCurrentSession(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	currentJTI := uuid.Must(uuid.NewV7()).String()

	sessions := []*domain.Session{
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      currentJTI,
			Device:   "Desktop",
			Revoked:  false,
		},
	}

	// Mock expectations
	m.sessionRepo.On("GetByUserID", ctx, tenantID, userID).Return(sessions, nil)

	// Execute
	count, err := m.uc.RevokeAllSessions(ctx, tenantID, userID, currentJTI)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "No debe revocar ninguna sesión (solo hay la actual)")

	// No debe llamar a Revoke
	m.sessionRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	m.sessionRepo.AssertExpectations(t)
}

// TestRevokeAllSessions_PartialFailure prueba cuando algunas revocaciones fallan
func TestRevokeAllSessions_PartialFailure(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV7()).String()
	tenantID := "test-tenant"
	currentJTI := uuid.Must(uuid.NewV7()).String()

	sessions := []*domain.Session{
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      currentJTI,
			Device:   "Desktop",
			Revoked:  false,
		},
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      uuid.Must(uuid.NewV7()).String(),
			Device:   "Mobile",
			Revoked:  false,
		},
		{
			ID:       uuid.Must(uuid.NewV7()).String(),
			TenantID: tenantID,
			UserID:   userID,
			JTI:      uuid.Must(uuid.NewV7()).String(),
			Device:   "Tablet",
			Revoked:  false,
		},
	}

	// Mock expectations
	m.sessionRepo.On("GetByUserID", ctx, tenantID, userID).Return(sessions, nil)

	// Primera revocación exitosa
	m.sessionRepo.On("Revoke", ctx, tenantID, sessions[1].ID, "user", "revoked all sessions").Return(nil).Once()
	// Segunda revocación falla
	m.sessionRepo.On("Revoke", ctx, tenantID, sessions[2].ID, "user", "revoked all sessions").Return(domain.ErrSessionExpired).Once()

	// Execute
	count, err := m.uc.RevokeAllSessions(ctx, tenantID, userID, currentJTI)

	// Assert
	assert.NoError(t, err, "No debe retornar error incluso si algunas revocaciones fallan")
	assert.Equal(t, 1, count, "Debe contar solo las revocaciones exitosas")
	m.sessionRepo.AssertExpectations(t)
}

// TestCleanupExpiredSessions_Success prueba la limpieza exitosa de sesiones expiradas
func TestCleanupExpiredSessions_Success(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	// Mock expectations
	m.sessionRepo.On("DeleteExpired", ctx).Return(nil)

	// Execute
	err := m.uc.CleanupExpiredSessions(ctx)

	// Assert
	assert.NoError(t, err)
	m.sessionRepo.AssertExpectations(t)
}

// TestCleanupExpiredSessions_Error prueba error al limpiar sesiones expiradas
func TestCleanupExpiredSessions_Error(t *testing.T) {
	m := setupSessionUseCase(t)
	ctx := context.Background()

	// Mock expectations
	m.sessionRepo.On("DeleteExpired", ctx).Return(domain.ErrSessionExpired)

	// Execute
	err := m.uc.CleanupExpiredSessions(ctx)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionExpired, err)
	m.sessionRepo.AssertExpectations(t)
}
