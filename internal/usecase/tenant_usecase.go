package usecase

import (
	"context"
	"fmt"

	"github.com/vertercloud/auth-service/internal/config"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
)

type TenantUseCase struct {
	tenantRepo     ports.TenantRepository
	userRepo       ports.UserRepository
	roleRepo       ports.RoleRepository
	passwordHasher ports.PasswordHasher
	config         *config.Config
}

func NewTenantUseCase(
	tenantRepo ports.TenantRepository,
	userRepo ports.UserRepository,
	roleRepo ports.RoleRepository,
	passwordHasher ports.PasswordHasher,
	cfg *config.Config,
) *TenantUseCase {
	return &TenantUseCase{
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		passwordHasher: passwordHasher,
		config:         cfg,
	}
}

type RegisterTenantInput struct {
	TenantSlug    string
	TenantName    string
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

func (uc *TenantUseCase) Register(ctx context.Context, input RegisterTenantInput) (*domain.Tenant, *domain.User, error) {
	// 1. Validar que el slug es único
	existingTenant, err := uc.tenantRepo.GetBySlug(ctx, input.TenantSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check tenant existence: %w", err)
	}
	if existingTenant != nil {
		return nil, nil, fmt.Errorf("tenant with slug %s already exists", input.TenantSlug) // Debería ir un domain error estricto
	}

	// 2. Crear al nuevo Tenant / Inquilino
	tenant := domain.NewTenant(domain.NewTenantInput{
		Slug: input.TenantSlug,
		Name: input.TenantName,
	})

	if err := uc.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// 3. Crear el Rol de Administrador para este Inquilino específico
	// (Verificamos si no existe un pre-lote automático, usualmente hay que sembrarlo)
	adminRole := domain.NewRole("admin", "System Administrator")
	adminRole.TenantID = tenant.ID // Required to map the newly global role into this specific context scope

	if err := uc.roleRepo.CreateRole(ctx, adminRole); err != nil {
		// Log error but continue user creation
		fmt.Printf("warning: failed to create admin role for tenant %s: %v\n", tenant.ID, err)
	}

	// 4. Crear el Usuario Administrador base
	passwordHash, err := uc.passwordHasher.Hash(input.AdminPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := domain.NewUser(domain.NewUserInput{
		TenantID: tenant.ID,
		Username: input.AdminUsername,
		Email:    input.AdminEmail,
		Password: input.AdminPassword,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to structure admin user: %w", err)
	}

	if err := uc.userRepo.Create(ctx, user, passwordHash); err != nil {
		// Lo ideal sería hacer Rollback del tenant en caso de fallo crítico
		return nil, nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// 5. Vincular al usuario con el rol de admin
	if err := uc.roleRepo.AssignRoleToUser(ctx, tenant.ID, user.ID, adminRole.ID); err != nil {
		fmt.Printf("warning: failed to assign admin role to user %s: %v\n", user.ID, err)
	}

	return tenant, user, nil
}

// GetBySlug localiza un Tenant dado su URL alias string (ej: 'google')
func (uc *TenantUseCase) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return uc.tenantRepo.GetBySlug(ctx, slug)
}
