-- Initialize backend database schema
-- This extends the WhatsApp initialization with backend-specific tables
-- Create the backend schema
CREATE SCHEMA IF NOT EXISTS backend;
-- Grant permissions to tennex user
GRANT ALL PRIVILEGES ON SCHEMA backend TO tennex;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA backend TO tennex;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA backend TO tennex;
-- Set search path to include backend schema
ALTER USER tennex
SET search_path = backend,
    whatsmeow,
    public;
-- Set search path for current session
SET search_path = backend,
    whatsmeow,
    public;
-- ==================== SCHEMA MIGRATIONS ====================
-- Function to update updated_at timestamp (used by multiple triggers)
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';
-- ==================== 001: Initial Schema ====================
-- Events table: append-only, authoritative source of truth
CREATE TABLE IF NOT EXISTS events (
    seq BIGSERIAL PRIMARY KEY,
    id UUID NOT NULL UNIQUE,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type TEXT NOT NULL,
    account_id TEXT NOT NULL,
    device_id TEXT,
    convo_id TEXT NOT NULL,
    wa_message_id TEXT,
    sender_jid TEXT,
    payload JSONB NOT NULL,
    attachment_ref JSONB
);
-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_events_account_seq ON events (account_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_convo_seq ON events (convo_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_type_ts ON events (type, ts);
CREATE INDEX IF NOT EXISTS idx_events_wa_message_id ON events (wa_message_id)
WHERE wa_message_id IS NOT NULL;
-- Outbox table: durable send path for outgoing messages
CREATE TABLE IF NOT EXISTS outbox (
    client_msg_uuid UUID PRIMARY KEY,
    account_id TEXT NOT NULL,
    convo_id TEXT NOT NULL,
    server_msg_id BIGINT REFERENCES events(seq),
    status TEXT NOT NULL DEFAULT 'queued',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Indexes for outbox processing
CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox (status, created_at)
WHERE status IN ('queued', 'retry');
CREATE INDEX IF NOT EXISTS idx_outbox_account_status ON outbox (account_id, status);
-- Media blobs table: content-addressed storage references
CREATE TABLE IF NOT EXISTS media_blobs (
    content_hash TEXT PRIMARY KEY,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add constraints and checks
DO $$ BEGIN
ALTER TABLE events
ADD CONSTRAINT events_type_check CHECK (
        type IN (
            'msg_in',
            'msg_out_pending',
            'msg_out_sent',
            'msg_delivery',
            'presence',
            'contact_update',
            'history_sync'
        )
    );
EXCEPTION
WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
ALTER TABLE outbox
ADD CONSTRAINT outbox_status_check CHECK (
        status IN ('queued', 'sending', 'sent', 'failed', 'retry')
    );
EXCEPTION
WHEN duplicate_object THEN NULL;
END $$;
-- Triggers to automatically update updated_at
DROP TRIGGER IF EXISTS update_outbox_updated_at ON outbox;
CREATE TRIGGER update_outbox_updated_at BEFORE
UPDATE ON outbox FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- ==================== 002: Add Users ====================
-- Users table: store user accounts and authentication data
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username)
WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email)
WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);
-- Add constraints and checks
DO $$ BEGIN
ALTER TABLE users
ADD CONSTRAINT users_username_check CHECK (
        length(username) >= 3
        AND length(username) <= 30
        AND username ~ '^[a-zA-Z0-9_]+$'
    );
EXCEPTION
WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
ALTER TABLE users
ADD CONSTRAINT users_email_check CHECK (
        email ~ '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$'
    );
EXCEPTION
WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
ALTER TABLE users
ADD CONSTRAINT users_password_hash_check CHECK (length(password_hash) >= 1);
EXCEPTION
WHEN duplicate_object THEN NULL;
END $$;
-- Trigger to automatically update updated_at
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE
UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- ==================== 003: User Integrations ====================
-- Create the new generic user_integrations table
CREATE TABLE IF NOT EXISTS user_integrations (
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
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_integration UNIQUE (user_id, integration_type),
    CONSTRAINT unique_external_id UNIQUE (integration_type, external_id)
);
-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_integrations_user_id ON user_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_type ON user_integrations(integration_type);
CREATE INDEX IF NOT EXISTS idx_user_integrations_status ON user_integrations(status);
CREATE INDEX IF NOT EXISTS idx_user_integrations_external_id ON user_integrations(external_id);
-- Create trigger to update updated_at automatically
CREATE OR REPLACE FUNCTION update_user_integrations_updated_at() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trigger_update_user_integrations_updated_at ON user_integrations;
CREATE TRIGGER trigger_update_user_integrations_updated_at BEFORE
UPDATE ON user_integrations FOR EACH ROW EXECUTE FUNCTION update_user_integrations_updated_at();
-- ==================== 004: Conversations and Messages ====================
-- Conversations table: generic conversation/chat/channel storage
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id INTEGER NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    external_conversation_id TEXT NOT NULL,
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    conversation_type TEXT NOT NULL CHECK (
        conversation_type IN ('individual', 'group', 'broadcast', 'channel')
    ),
    name TEXT,
    description TEXT,
    avatar_url TEXT,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    mute_until TIMESTAMPTZ,
    is_read_only BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    unread_count INTEGER NOT NULL DEFAULT 0,
    unread_mention_count INTEGER NOT NULL DEFAULT 0,
    total_message_count INTEGER NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    CONSTRAINT unique_external_conversation UNIQUE(user_integration_id, external_conversation_id),
    CONSTRAINT conversations_unread_count_check CHECK (unread_count >= 0),
    CONSTRAINT conversations_unread_mention_count_check CHECK (unread_mention_count >= 0),
    CONSTRAINT conversations_total_message_count_check CHECK (total_message_count >= 0)
);
-- Indexes for conversations
CREATE INDEX IF NOT EXISTS idx_conversations_integration ON conversations (user_integration_id, integration_type);
CREATE INDEX IF NOT EXISTS idx_conversations_last_activity ON conversations (last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_unread ON conversations (unread_count)
WHERE unread_count > 0;
CREATE INDEX IF NOT EXISTS idx_conversations_pinned ON conversations (is_pinned)
WHERE is_pinned = true;
CREATE INDEX IF NOT EXISTS idx_conversations_archived ON conversations (is_archived)
WHERE is_archived = true;
-- Conversation participants table
CREATE TABLE IF NOT EXISTS conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    display_name TEXT,
    role TEXT NOT NULL DEFAULT 'member' CHECK (
        role IN ('member', 'admin', 'owner', 'moderator')
    ),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    added_by_external_id TEXT,
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_conversation_participant UNIQUE(conversation_id, external_user_id)
);
-- Indexes for conversation participants
CREATE INDEX IF NOT EXISTS idx_conversation_participants_conversation ON conversation_participants (conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_participants_external_user ON conversation_participants (external_user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_participants_active ON conversation_participants (is_active)
WHERE is_active = true;
-- Messages table: generic message storage
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_message_id TEXT NOT NULL,
    external_server_id TEXT,
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    sender_external_id TEXT NOT NULL,
    sender_display_name TEXT,
    message_type TEXT NOT NULL CHECK (
        message_type IN (
            'text',
            'image',
            'video',
            'audio',
            'document',
            'location',
            'contact',
            'sticker',
            'poll',
            'reaction',
            'system'
        )
    ),
    content TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    edit_timestamp TIMESTAMPTZ,
    is_from_me BOOLEAN NOT NULL DEFAULT FALSE,
    is_forwarded BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    reply_to_message_id UUID,
    reply_to_external_id TEXT,
    delivery_status TEXT NOT NULL DEFAULT 'sent' CHECK (
        delivery_status IN ('sent', 'delivered', 'read', 'failed')
    ),
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_external_message UNIQUE(conversation_id, external_message_id)
);
-- Indexes for messages
CREATE INDEX IF NOT EXISTS idx_messages_conversation_timestamp ON messages (conversation_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages (sender_external_id);
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages (message_type);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages (conversation_id, timestamp)
WHERE delivery_status != 'read';
CREATE INDEX IF NOT EXISTS idx_messages_deleted ON messages (is_deleted)
WHERE is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_messages_reply ON messages (reply_to_message_id)
WHERE reply_to_message_id IS NOT NULL;
-- Message media table: for platforms that support rich media
CREATE TABLE IF NOT EXISTS message_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (
        media_type IN ('image', 'video', 'audio', 'document', 'sticker')
    ),
    file_name TEXT,
    file_size BIGINT,
    mime_type TEXT,
    duration_seconds INTEGER,
    width INTEGER,
    height INTEGER,
    original_url TEXT,
    thumbnail_url TEXT,
    local_file_path TEXT,
    download_status TEXT NOT NULL DEFAULT 'pending' CHECK (
        download_status IN ('pending', 'downloading', 'completed', 'failed')
    ),
    downloaded_at TIMESTAMPTZ,
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT message_media_file_size_check CHECK (
        file_size IS NULL
        OR file_size > 0
    ),
    CONSTRAINT message_media_duration_check CHECK (
        duration_seconds IS NULL
        OR duration_seconds > 0
    ),
    CONSTRAINT message_media_dimensions_check CHECK (
        (
            width IS NULL
            AND height IS NULL
        )
        OR (
            width > 0
            AND height > 0
        )
    )
);
-- Indexes for message media
CREATE INDEX IF NOT EXISTS idx_message_media_message ON message_media (message_id);
CREATE INDEX IF NOT EXISTS idx_message_media_type ON message_media (media_type);
CREATE INDEX IF NOT EXISTS idx_message_media_download_status ON message_media (download_status)
WHERE download_status != 'completed';
-- Contacts table: integration-specific contact storage
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id INTEGER NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    external_contact_id TEXT NOT NULL,
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    display_name TEXT,
    first_name TEXT,
    last_name TEXT,
    phone_number TEXT,
    username TEXT,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen TIMESTAMPTZ,
    avatar_url TEXT,
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_external_contact UNIQUE(user_integration_id, external_contact_id)
);
-- Indexes for contacts
CREATE INDEX IF NOT EXISTS idx_contacts_integration ON contacts (user_integration_id, integration_type);
CREATE INDEX IF NOT EXISTS idx_contacts_external_id ON contacts (external_contact_id);
CREATE INDEX IF NOT EXISTS idx_contacts_blocked ON contacts (is_blocked)
WHERE is_blocked = true;
CREATE INDEX IF NOT EXISTS idx_contacts_favorite ON contacts (is_favorite)
WHERE is_favorite = true;
-- Integration settings table: per-integration settings
CREATE TABLE IF NOT EXISTS integration_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id INTEGER NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    setting_key TEXT NOT NULL,
    setting_value JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_integration_setting UNIQUE(user_integration_id, setting_key),
    CONSTRAINT integration_settings_key_check CHECK (length(setting_key) > 0)
);
-- Indexes for integration settings
CREATE INDEX IF NOT EXISTS idx_integration_settings_user_integration ON integration_settings (user_integration_id);
CREATE INDEX IF NOT EXISTS idx_integration_settings_key ON integration_settings (setting_key);
-- Triggers to automatically update updated_at timestamps
DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;
CREATE TRIGGER update_conversations_updated_at BEFORE
UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_conversation_participants_updated_at ON conversation_participants;
CREATE TRIGGER update_conversation_participants_updated_at BEFORE
UPDATE ON conversation_participants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;
CREATE TRIGGER update_messages_updated_at BEFORE
UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_message_media_updated_at ON message_media;
CREATE TRIGGER update_message_media_updated_at BEFORE
UPDATE ON message_media FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_contacts_updated_at ON contacts;
CREATE TRIGGER update_contacts_updated_at BEFORE
UPDATE ON contacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_integration_settings_updated_at ON integration_settings;
CREATE TRIGGER update_integration_settings_updated_at BEFORE
UPDATE ON integration_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- ==================== 006: Add Sync Sequences ====================
-- Add seq columns for efficient cursor-based syncing
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'conversations'
        AND column_name = 'seq'
) THEN
ALTER TABLE conversations
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_conversations_seq ON conversations (seq);
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'messages'
        AND column_name = 'seq'
) THEN
ALTER TABLE messages
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_messages_seq ON messages (seq);
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'contacts'
        AND column_name = 'seq'
) THEN
ALTER TABLE contacts
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_contacts_seq ON contacts (seq);
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'message_media'
        AND column_name = 'seq'
) THEN
ALTER TABLE message_media
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_message_media_seq ON message_media (seq);
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'conversation_participants'
        AND column_name = 'seq'
) THEN
ALTER TABLE conversation_participants
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_conversation_participants_seq ON conversation_participants (seq);
END IF;
END $$;
-- Re-grant permissions after creating all tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA backend TO tennex;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA backend TO tennex;
-- Log the initialization
\ echo 'Backend database schema initialized successfully with all migrations'