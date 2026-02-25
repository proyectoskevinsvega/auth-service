-- Add performance indexes for frequently queried columns
-- Note: CONCURRENTLY cannot be used with golang-migrate transactions
-- For production: run scripts/create_indexes_production.sql with CONCURRENTLY
-- For development: these run without CONCURRENTLY inside transactions

-- Users table indexes (filtered by active users)
CREATE INDEX IF NOT EXISTS idx_users_email_active ON auth_users(email) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_username_active ON auth_users(username) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_oauth_active ON auth_users(oauth_provider, oauth_provider_id) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_created_at_active ON auth_users(created_at DESC) WHERE active = TRUE;

-- Refresh tokens indexes (filtered by non-revoked tokens)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_active ON auth_refresh_tokens(user_id) WHERE revoked = FALSE;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_active ON auth_refresh_tokens(session_id) WHERE revoked = FALSE;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_active ON auth_refresh_tokens(expires_at) WHERE revoked = FALSE;

-- Sessions indexes (filtered by non-revoked sessions)
CREATE INDEX IF NOT EXISTS idx_sessions_user_active ON auth_sessions(user_id) WHERE revoked = FALSE;
CREATE INDEX IF NOT EXISTS idx_sessions_expires_active ON auth_sessions(expires_at) WHERE revoked = FALSE;
CREATE INDEX IF NOT EXISTS idx_sessions_last_used_active ON auth_sessions(last_used_at DESC) WHERE revoked = FALSE;

-- Password resets indexes (filtered by unused tokens)
CREATE INDEX IF NOT EXISTS idx_password_resets_user_unused ON auth_password_resets(user_id) WHERE used = FALSE;
CREATE INDEX IF NOT EXISTS idx_password_resets_expires_unused ON auth_password_resets(expires_at) WHERE used = FALSE;

-- Audit log indexes (for admin queries - correct table name)
CREATE INDEX IF NOT EXISTS idx_audit_log_user_action ON auth_audit_log(user_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action_time ON auth_audit_log(action, created_at DESC);
