-- Rollback Migration v9: RBAC & ABAC

DROP TABLE IF EXISTS auth_user_roles;
DROP TABLE IF EXISTS auth_role_permissions;
DROP TABLE IF EXISTS auth_permissions;
DROP TABLE IF EXISTS auth_roles;

ALTER TABLE auth_users DROP COLUMN IF EXISTS attributes;
