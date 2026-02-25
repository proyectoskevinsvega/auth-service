package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vertercloud/auth-service/internal/domain"
)

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	query := `INSERT INTO auth_roles (id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(ctx, query, role.ID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *RoleRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM auth_roles WHERE name = $1`
	role := &domain.Role{}
	err := r.db.QueryRow(ctx, query, name).Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Load permissions
	perms, err := r.getRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	role.Permissions = perms

	return role, nil
}

func (r *RoleRepository) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM auth_roles`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	for _, role := range roles {
		perms, err := r.getRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions = perms
	}

	return roles, nil
}

func (r *RoleRepository) getRolePermissions(ctx context.Context, roleID string) ([]domain.Permission, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at 
		FROM auth_permissions p
		JOIN auth_role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`
	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		p := domain.Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *RoleRepository) AddPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	query := `INSERT INTO auth_role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	return err
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	query := `INSERT INTO auth_user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, query, userID, roleID)
	return err
}

func (r *RoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	query := `DELETE FROM auth_user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.Exec(ctx, query, userID, roleID)
	return err
}

func (r *RoleRepository) GetUserRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at 
		FROM auth_roles r
		JOIN auth_user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		role := domain.Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	for i := range roles {
		perms, err := r.getRolePermissions(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}

	return roles, nil
}

func (r *RoleRepository) CreatePermission(ctx context.Context, perm *domain.Permission) error {
	query := `INSERT INTO auth_permissions (id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(ctx, query, perm.ID, perm.Name, perm.Description, perm.CreatedAt, perm.UpdatedAt)
	return err
}

func (r *RoleRepository) GetPermissionByName(ctx context.Context, name string) (*domain.Permission, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM auth_permissions WHERE name = $1`
	perm := &domain.Permission{}
	err := r.db.QueryRow(ctx, query, name).Scan(&perm.ID, &perm.Name, &perm.Description, &perm.CreatedAt, &perm.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, err
	}
	return perm, nil
}

func (r *RoleRepository) ListPermissions(ctx context.Context) ([]*domain.Permission, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM auth_permissions`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		p := &domain.Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}
