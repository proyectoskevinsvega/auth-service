-- Migration v11: Multi-tenancy Isolation (User Pools)

-- 1. Create auth_tenants table
CREATE TABLE IF NOT EXISTS auth_tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(50) UNIQUE NOT NULL, -- e.g. 'default', 'acme', 'vertercloud'
    name VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_tenants_slug ON auth_tenants(slug);

-- 2. Create default tenant
INSERT INTO auth_tenants (slug, name) 
VALUES ('default', 'Default Organization')
ON CONFLICT (slug) DO NOTHING;

-- 3. Add tenant_id to all core tables
DO $$ 
BEGIN
    -- auth_users
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_users' AND column_name='tenant_id') THEN
        ALTER TABLE auth_users ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_users SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_users ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_sessions
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_sessions' AND column_name='tenant_id') THEN
        ALTER TABLE auth_sessions ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_sessions SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_sessions ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_refresh_tokens
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_refresh_tokens' AND column_name='tenant_id') THEN
        ALTER TABLE auth_refresh_tokens ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_refresh_tokens SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_refresh_tokens ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_password_resets
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_password_resets' AND column_name='tenant_id') THEN
        ALTER TABLE auth_password_resets ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_password_resets SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_password_resets ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_audit_log
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_audit_log' AND column_name='tenant_id') THEN
        ALTER TABLE auth_audit_log ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_audit_log SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_audit_log ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_roles
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_roles' AND column_name='tenant_id') THEN
        ALTER TABLE auth_roles ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_roles SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_roles ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_permissions
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_permissions' AND column_name='tenant_id') THEN
        ALTER TABLE auth_permissions ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_permissions SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_permissions ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_webauthn_credentials
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_webauthn_credentials' AND column_name='tenant_id') THEN
        ALTER TABLE auth_webauthn_credentials ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_webauthn_credentials SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_webauthn_credentials ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_email_verifications
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_email_verifications' AND column_name='tenant_id') THEN
        ALTER TABLE auth_email_verifications ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_email_verifications SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_email_verifications ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_2fa
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_2fa' AND column_name='tenant_id') THEN
        ALTER TABLE auth_2fa ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_2fa SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_2fa ALTER COLUMN tenant_id SET NOT NULL;
    END IF;
END $$;

-- 4. Update constraints to be tenant-scoped
ALTER TABLE auth_users DROP CONSTRAINT IF EXISTS auth_users_username_key;
ALTER TABLE auth_users ADD CONSTRAINT auth_users_username_tenant_key UNIQUE (username, tenant_id);

ALTER TABLE auth_users DROP CONSTRAINT IF EXISTS auth_users_email_key;
ALTER TABLE auth_users ADD CONSTRAINT auth_users_email_tenant_key UNIQUE (email, tenant_id);

ALTER TABLE auth_roles DROP CONSTRAINT IF EXISTS auth_roles_name_key;
ALTER TABLE auth_roles ADD CONSTRAINT auth_roles_name_tenant_key UNIQUE (name, tenant_id);

ALTER TABLE auth_permissions DROP CONSTRAINT IF EXISTS auth_permissions_name_key;
ALTER TABLE auth_permissions ADD CONSTRAINT auth_permissions_name_tenant_key UNIQUE (name, tenant_id);

-- 5. Create indexes for tenant_id filtering
CREATE INDEX idx_auth_users_tenant_id ON auth_users(tenant_id);
CREATE INDEX idx_auth_sessions_tenant_id ON auth_sessions(tenant_id);
CREATE INDEX idx_auth_refresh_tokens_tenant_id ON auth_refresh_tokens(tenant_id);
CREATE INDEX idx_auth_audit_log_tenant_id ON auth_audit_log(tenant_id);
CREATE INDEX idx_auth_roles_tenant_id ON auth_roles(tenant_id);
