-- Add webhook subscriptions support
CREATE TABLE IF NOT EXISTS auth_webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_tenants(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    event_types TEXT[] NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_tenant_active ON auth_webhook_subscriptions(tenant_id, active);

-- Add comment
COMMENT ON COLUMN auth_webhook_subscriptions.event_types IS 'List of event types this webhook is subscribed to (e.g., auth_user_registered)';
