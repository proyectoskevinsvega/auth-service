-- Migration v11 Down: Multi-tenancy Isolation

-- 1. Restore unique constraints
ALTER TABLE auth_users DROP CONSTRAINT IF EXISTS auth_users_username_tenant_key;
ALTER TABLE auth_users ADD CONSTRAINT auth_users_username_key UNIQUE (username);

ALTER TABLE auth_users DROP CONSTRAINT IF EXISTS auth_users_email_tenant_key;
ALTER TABLE auth_users ADD CONSTRAINT auth_users_email_key UNIQUE (email);

ALTER TABLE auth_roles DROP CONSTRAINT IF EXISTS auth_roles_name_tenant_key;
ALTER TABLE auth_roles ADD CONSTRAINT auth_roles_name_key UNIQUE (name);

ALTER TABLE auth_permissions DROP CONSTRAINT IF EXISTS auth_permissions_name_tenant_key;
ALTER TABLE auth_permissions ADD CONSTRAINT auth_permissions_name_key UNIQUE (name);

-- 2. Remove tenant_id columns
ALTER TABLE auth_2fa DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_email_verifications DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_webauthn_credentials DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_permissions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_roles DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_audit_log DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_password_resets DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_refresh_tokens DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_sessions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE auth_users DROP COLUMN IF EXISTS tenant_id;

-- 3. Drop auth_tenants table
DROP TABLE IF EXISTS auth_tenants;
