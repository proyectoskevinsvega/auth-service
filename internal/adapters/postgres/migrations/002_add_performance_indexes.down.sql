-- Remove performance indexes

-- Users table indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_users_email;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_username;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_oauth;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at;

-- Refresh tokens indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_refresh_tokens_session_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_refresh_tokens_parent_token;
DROP INDEX CONCURRENTLY IF EXISTS idx_refresh_tokens_expires_at;

-- Sessions indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_sessions_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_sessions_jti;
DROP INDEX CONCURRENTLY IF EXISTS idx_sessions_expires_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_sessions_last_used_at;

-- Password resets indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_password_resets_token_hash;
DROP INDEX CONCURRENTLY IF EXISTS idx_password_resets_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_password_resets_expires_at;

-- Audit logs indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_action;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_user_action;

-- Blocked IPs indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_blocked_ips_ip;
DROP INDEX CONCURRENTLY IF EXISTS idx_blocked_ips_expires_at;
