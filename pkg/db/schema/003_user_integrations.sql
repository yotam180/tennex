-- Add user_integrations table for multi-platform support
-- This replaces the WhatsApp-specific accounts table with a generic integration system
-- Drop the old WhatsApp-specific accounts table
DROP TABLE IF EXISTS accounts CASCADE;
-- Create the new generic user_integrations table
CREATE TABLE user_integrations (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    external_id TEXT NOT NULL,
    -- JID for WhatsApp, email address for Email, etc.
    status TEXT NOT NULL DEFAULT 'disconnected' CHECK (
        status IN (
            'connected',
            'disconnected',
            'connecting',
            'error'
        )
    ),
    display_name TEXT,
    avatar_url TEXT,
    metadata JSONB DEFAULT '{}',
    -- Platform-specific data
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Ensure one integration per type per user
    CONSTRAINT unique_user_integration UNIQUE (user_id, integration_type),
    -- Ensure external_id is unique per integration type (prevents duplicate accounts)
    CONSTRAINT unique_external_id UNIQUE (integration_type, external_id)
);
-- Create indexes for performance
CREATE INDEX idx_user_integrations_user_id ON user_integrations(user_id);
CREATE INDEX idx_user_integrations_type ON user_integrations(integration_type);
CREATE INDEX idx_user_integrations_status ON user_integrations(status);
CREATE INDEX idx_user_integrations_external_id ON user_integrations(external_id);
-- Create trigger to update updated_at automatically
CREATE OR REPLACE FUNCTION update_user_integrations_updated_at() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trigger_update_user_integrations_updated_at BEFORE
UPDATE ON user_integrations FOR EACH ROW EXECUTE FUNCTION update_user_integrations_updated_at();
-- Add comment for documentation
COMMENT ON TABLE user_integrations IS 'Generic table for storing user integrations across multiple platforms (WhatsApp, Email, Telegram, etc.)';
COMMENT ON COLUMN user_integrations.integration_type IS 'Platform type: whatsapp, email, telegram, discord, slack';
COMMENT ON COLUMN user_integrations.external_id IS 'Platform-specific identifier (JID for WhatsApp, email address for Email, etc.)';
COMMENT ON COLUMN user_integrations.metadata IS 'Platform-specific data stored as JSON';