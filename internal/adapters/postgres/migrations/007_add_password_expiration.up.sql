-- Add password_changed_at column to auth_users table
ALTER TABLE auth_users 
ADD COLUMN password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Update existing users to have a default value for password_changed_at
-- This ensures they don't expire immediately if they were created recently
UPDATE auth_users SET password_changed_at = created_at WHERE password_changed_at IS NULL;
