-- v6: Add account lockout fields for security P0
ALTER TABLE auth_users ADD COLUMN failed_login_attempts INT DEFAULT 0;
ALTER TABLE auth_users ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE;

-- Add index to speed up lookup for locked accounts if needed (though usually we fetch by ID/email)
CREATE INDEX idx_auth_users_locked_until ON auth_users(locked_until) WHERE locked_until IS NOT NULL;
