-- Drop table and trigger
DROP TABLE IF EXISTS auth_webauthn_credentials;

-- Remove webauthn_id column
ALTER TABLE auth_users DROP COLUMN IF EXISTS webauthn_id;
