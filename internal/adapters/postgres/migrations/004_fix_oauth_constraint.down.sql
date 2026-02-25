-- Revert the oauth constraint fix
DROP INDEX IF EXISTS idx_auth_users_oauth_unique;

-- Restore the original constraint
ALTER TABLE auth_users
    ADD CONSTRAINT unique_oauth UNIQUE (oauth_provider, oauth_provider_id);
