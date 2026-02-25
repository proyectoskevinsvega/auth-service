-- Migration v13: 2FA Backup Codes

CREATE TABLE IF NOT EXISTS auth_2fa_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_2fa_backup_codes_user_id ON auth_2fa_backup_codes(user_id);
CREATE INDEX idx_auth_2fa_backup_codes_tenant_id ON auth_2fa_backup_codes(tenant_id);
