-- Add webauthn_id to auth_users for WebAuthn User Handle
ALTER TABLE auth_users ADD COLUMN IF NOT EXISTS webauthn_id BYTEA UNIQUE;

-- Function for automatic updated_at trigger
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- WebAuthn Credentials Table
CREATE TABLE IF NOT EXISTS auth_webauthn_credentials (
    id BYTEA PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,
    attestation_type TEXT NOT NULL,
    aaguid BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    clone_warning BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookup by user
CREATE INDEX IF NOT EXISTS idx_webauthn_user_id ON auth_webauthn_credentials(user_id);

-- Update trigger for updated_at
CREATE TRIGGER update_auth_webauthn_credentials_modtime
    BEFORE UPDATE ON auth_webauthn_credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();
