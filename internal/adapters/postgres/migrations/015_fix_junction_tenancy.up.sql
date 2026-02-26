-- Migration v15: Fix Multi-tenancy for Junction Tables
-- This migration ensures junction tables also have tenant_id for full isolation

DO $$ 
BEGIN
    -- auth_user_roles (junction table)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_user_roles' AND column_name='tenant_id') THEN
        ALTER TABLE auth_user_roles ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_user_roles SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_user_roles ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- auth_role_permissions (junction table)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='auth_role_permissions' AND column_name='tenant_id') THEN
        ALTER TABLE auth_role_permissions ADD COLUMN tenant_id UUID REFERENCES auth_tenants(id) ON DELETE CASCADE;
        UPDATE auth_role_permissions SET tenant_id = (SELECT id FROM auth_tenants WHERE slug = 'default') WHERE tenant_id IS NULL;
        ALTER TABLE auth_role_permissions ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- Add composite indexes for junction tables if they don't exist
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_auth_user_roles_tenant_user') THEN
        CREATE INDEX idx_auth_user_roles_tenant_user ON auth_user_roles(tenant_id, user_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_auth_role_perms_tenant_role') THEN
        CREATE INDEX idx_auth_role_perms_tenant_role ON auth_role_permissions(tenant_id, role_id);
    END IF;

END $$;
