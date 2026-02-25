-- Create email verifications table
CREATE TABLE IF NOT EXISTS auth_email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(45),
    user_agent TEXT
);

-- Indexes for performance
CREATE INDEX idx_email_verifications_user_id ON auth_email_verifications(user_id) WHERE verified_at IS NULL;
CREATE INDEX idx_email_verifications_token_hash ON auth_email_verifications(token_hash) WHERE verified_at IS NULL;
CREATE INDEX idx_email_verifications_expires_at ON auth_email_verifications(expires_at) WHERE verified_at IS NULL;

COMMENT ON TABLE auth_email_verifications IS 'Stores email verification tokens with expiration';
