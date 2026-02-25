-- Add password_reset_required column to auth_users table
ALTER TABLE auth_users ADD COLUMN password_reset_required BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN auth_users.password_reset_required IS 'Flag set by admin to force a password change on next login';
