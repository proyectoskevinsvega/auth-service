-- Auth Users Table
CREATE TABLE IF NOT EXISTS auth_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(30) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT false,
    two_factor_secret TEXT,
    oauth_provider VARCHAR(50),
    oauth_provider_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    last_login_country VARCHAR(2),
    CONSTRAINT unique_oauth UNIQUE (oauth_provider, oauth_provider_id)
);

CREATE INDEX idx_auth_users_username ON auth_users(username);
CREATE INDEX idx_auth_users_email ON auth_users(email);
CREATE INDEX idx_auth_users_oauth ON auth_users(oauth_provider, oauth_provider_id);
CREATE INDEX idx_auth_users_active ON auth_users(active);

-- Auth Sessions Table
CREATE TABLE IF NOT EXISTS auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45) NOT NULL,
    country VARCHAR(2),
    device VARCHAR(255),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMP,
    revoked_by VARCHAR(50),
    revoke_reason TEXT
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);
CREATE INDEX idx_auth_sessions_revoked ON auth_sessions(revoked);

-- Auth Refresh Tokens Table
CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    previous_token UUID,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMP
);

CREATE INDEX idx_auth_refresh_tokens_user_id ON auth_refresh_tokens(user_id);
CREATE INDEX idx_auth_refresh_tokens_session_id ON auth_refresh_tokens(session_id);
CREATE INDEX idx_auth_refresh_tokens_token_hash ON auth_refresh_tokens(token_hash);
CREATE INDEX idx_auth_refresh_tokens_expires_at ON auth_refresh_tokens(expires_at);
CREATE INDEX idx_auth_refresh_tokens_revoked ON auth_refresh_tokens(revoked);

-- Auth Password Resets Table
CREATE TABLE IF NOT EXISTS auth_password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMP
);

CREATE INDEX idx_auth_password_resets_user_id ON auth_password_resets(user_id);
CREATE INDEX idx_auth_password_resets_token ON auth_password_resets(token);
CREATE INDEX idx_auth_password_resets_code ON auth_password_resets(user_id, code);
CREATE INDEX idx_auth_password_resets_expires_at ON auth_password_resets(expires_at);

-- Auth Audit Log Table
CREATE TABLE IF NOT EXISTS auth_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    country VARCHAR(2),
    success BOOLEAN NOT NULL,
    error_msg TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_audit_log_user_id ON auth_audit_log(user_id);
CREATE INDEX idx_auth_audit_log_action ON auth_audit_log(action);
CREATE INDEX idx_auth_audit_log_created_at ON auth_audit_log(created_at);
CREATE INDEX idx_auth_audit_log_success ON auth_audit_log(success);

-- Auth Blocked IPs Table
CREATE TABLE IF NOT EXISTS auth_blocked_ips (
    ip_address VARCHAR(45) PRIMARY KEY,
    reason TEXT NOT NULL,
    blocked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_auth_blocked_ips_expires_at ON auth_blocked_ips(expires_at);

-- Auth 2FA Table (for backup codes and TOTP configuration)
CREATE TABLE IF NOT EXISTS auth_2fa (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES auth_users(id) ON DELETE CASCADE,
    secret TEXT NOT NULL,
    backup_codes TEXT[],
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_2fa_user_id ON auth_2fa(user_id);
