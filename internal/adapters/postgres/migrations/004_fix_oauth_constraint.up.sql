-- Fix unique_oauth constraint to allow multiple NULL values
-- The current constraint prevents multiple users from registering without OAuth
-- because empty strings are considered equal

-- Drop the old constraint
ALTER TABLE auth_users DROP CONSTRAINT IF EXISTS unique_oauth;

-- Create a partial unique index that only applies when oauth_provider is NOT NULL
-- This allows unlimited users without OAuth (NULL or empty oauth_provider)
-- but maintains uniqueness for actual OAuth users
CREATE UNIQUE INDEX idx_auth_users_oauth_unique
    ON auth_users(oauth_provider, oauth_provider_id)
    WHERE oauth_provider IS NOT NULL AND oauth_provider != '';
