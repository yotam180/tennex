-- Add platform-agnostic conversations, messages, and contacts tables
-- This extends the user_integrations system with messaging functionality
-- Conversations table: generic conversation/chat/channel storage
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id INTEGER NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    external_conversation_id TEXT NOT NULL,
    -- WhatsApp conversation ID, Telegram chat ID, etc.
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
    -- Generic conversation metadata
    name TEXT,
    description TEXT,
    avatar_url TEXT,
    -- Generic conversation states (applicable to most platforms)
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    mute_until TIMESTAMPTZ,
    is_read_only BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    -- Message counts
    unread_count INTEGER NOT NULL DEFAULT 0,
    unread_mention_count INTEGER NOT NULL DEFAULT 0,
    total_message_count INTEGER NOT NULL DEFAULT 0,
    -- Timestamps
    last_message_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Platform-specific data
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    -- Constraints
    CONSTRAINT unique_external_conversation UNIQUE(user_integration_id, external_conversation_id),
    CONSTRAINT conversations_unread_count_check CHECK (unread_count >= 0),
    CONSTRAINT conversations_unread_mention_count_check CHECK (unread_mention_count >= 0),
    CONSTRAINT conversations_total_message_count_check CHECK (total_message_count >= 0)
);
-- Indexes for conversations
CREATE INDEX idx_conversations_integration ON conversations (user_integration_id, integration_type);
CREATE INDEX idx_conversations_last_activity ON conversations (last_activity_at DESC);
CREATE INDEX idx_conversations_unread ON conversations (unread_count)
WHERE unread_count > 0;
CREATE INDEX idx_conversations_pinned ON conversations (is_pinned)
WHERE is_pinned = true;
CREATE INDEX idx_conversations_archived ON conversations (is_archived)
WHERE is_archived = true;
-- Conversation participants table
CREATE TABLE conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    -- JID, Telegram user ID, etc.
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    -- Participant info
    display_name TEXT,
    role TEXT NOT NULL DEFAULT 'member' CHECK (
        role IN ('member', 'admin', 'owner', 'moderator')
    ),
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    added_by_external_id TEXT,
    -- Who added this participant
    -- Platform-specific data
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_conversation_participant UNIQUE(conversation_id, external_user_id)
);
-- Indexes for conversation participants
CREATE INDEX idx_conversation_participants_conversation ON conversation_participants (conversation_id);
CREATE INDEX idx_conversation_participants_external_user ON conversation_participants (external_user_id);
CREATE INDEX idx_conversation_participants_active ON conversation_participants (is_active)
WHERE is_active = true;
-- Messages table: generic message storage
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_message_id TEXT NOT NULL,
    -- WhatsApp message ID, Telegram message ID, etc.
    external_server_id TEXT,
    -- Optional server-side ID (some platforms have this)
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    -- Message source
    sender_external_id TEXT NOT NULL,
    -- Sender's platform ID
    sender_display_name TEXT,
    -- Sender's name at time of message
    -- Message content
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
    -- Text content or caption
    -- Message metadata
    timestamp TIMESTAMPTZ NOT NULL,
    edit_timestamp TIMESTAMPTZ,
    -- When message was last edited
    is_from_me BOOLEAN NOT NULL DEFAULT FALSE,
    is_forwarded BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    -- Threading (replies)
    reply_to_message_id UUID REFERENCES messages(id),
    reply_to_external_id TEXT,
    -- Original message's external ID
    -- Status
    delivery_status TEXT NOT NULL DEFAULT 'sent' CHECK (
        delivery_status IN ('sent', 'delivered', 'read', 'failed')
    ),
    -- Platform-specific data (reactions, media metadata, etc.)
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Constraints
    CONSTRAINT unique_external_message UNIQUE(conversation_id, external_message_id)
);
-- Indexes for messages
CREATE INDEX idx_messages_conversation_timestamp ON messages (conversation_id, timestamp DESC);
CREATE INDEX idx_messages_sender ON messages (sender_external_id);
CREATE INDEX idx_messages_type ON messages (message_type);
CREATE INDEX idx_messages_unread ON messages (conversation_id, timestamp)
WHERE delivery_status != 'read';
CREATE INDEX idx_messages_deleted ON messages (is_deleted)
WHERE is_deleted = false;
CREATE INDEX idx_messages_reply ON messages (reply_to_message_id)
WHERE reply_to_message_id IS NOT NULL;
-- Message media table: for platforms that support rich media
CREATE TABLE message_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    -- Generic media info
    media_type TEXT NOT NULL CHECK (
        media_type IN ('image', 'video', 'audio', 'document', 'sticker')
    ),
    file_name TEXT,
    file_size BIGINT,
    mime_type TEXT,
    duration_seconds INTEGER,
    -- For audio/video
    -- Dimensions
    width INTEGER,
    height INTEGER,
    -- URLs and paths
    original_url TEXT,
    thumbnail_url TEXT,
    local_file_path TEXT,
    -- If downloaded locally
    -- Download status
    download_status TEXT NOT NULL DEFAULT 'pending' CHECK (
        download_status IN ('pending', 'downloading', 'completed', 'failed')
    ),
    downloaded_at TIMESTAMPTZ,
    -- Platform-specific data (encryption keys, etc.)
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Constraints
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
CREATE INDEX idx_message_media_message ON message_media (message_id);
CREATE INDEX idx_message_media_type ON message_media (media_type);
CREATE INDEX idx_message_media_download_status ON message_media (download_status)
WHERE download_status != 'completed';
-- Contacts table: integration-specific contact storage
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_integration_id INTEGER NOT NULL REFERENCES user_integrations(id) ON DELETE CASCADE,
    external_contact_id TEXT NOT NULL,
    -- Contact's platform ID
    integration_type TEXT NOT NULL CHECK (
        integration_type IN (
            'whatsapp',
            'email',
            'telegram',
            'discord',
            'slack'
        )
    ),
    -- Contact info
    display_name TEXT,
    first_name TEXT,
    last_name TEXT,
    phone_number TEXT,
    username TEXT,
    -- Platform username if exists
    -- Status
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen TIMESTAMPTZ,
    -- Avatar
    avatar_url TEXT,
    -- Platform-specific data
    platform_metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_external_contact UNIQUE(user_integration_id, external_contact_id)
);
-- Indexes for contacts
CREATE INDEX idx_contacts_integration ON contacts (user_integration_id, integration_type);
CREATE INDEX idx_contacts_external_id ON contacts (external_contact_id);
CREATE INDEX idx_contacts_blocked ON contacts (is_blocked)
WHERE is_blocked = true;
CREATE INDEX idx_contacts_favorite ON contacts (is_favorite)
WHERE is_favorite = true;
-- Integration settings table: per-integration settings
CREATE TABLE integration_settings (
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
CREATE INDEX idx_integration_settings_user_integration ON integration_settings (user_integration_id);
CREATE INDEX idx_integration_settings_key ON integration_settings (setting_key);
-- Triggers to automatically update updated_at timestamps
CREATE TRIGGER update_conversations_updated_at BEFORE
UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_conversation_participants_updated_at BEFORE
UPDATE ON conversation_participants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_messages_updated_at BEFORE
UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_message_media_updated_at BEFORE
UPDATE ON message_media FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_contacts_updated_at BEFORE
UPDATE ON contacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_integration_settings_updated_at BEFORE
UPDATE ON integration_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- Comments for documentation
COMMENT ON TABLE conversations IS 'Platform-agnostic conversation/chat/channel storage supporting multiple messaging platforms';
COMMENT ON TABLE conversation_participants IS 'Participants in conversations, supporting groups and channels';
COMMENT ON TABLE messages IS 'Platform-agnostic message storage with support for various message types and threading';
COMMENT ON TABLE message_media IS 'Rich media attachments for messages (images, videos, audio, documents)';
COMMENT ON TABLE contacts IS 'Platform-specific contact information for each user integration';
COMMENT ON TABLE integration_settings IS 'Per-integration configuration settings';
COMMENT ON COLUMN conversations.external_conversation_id IS 'Platform-specific conversation identifier (WhatsApp conversation ID, Telegram chat ID, etc.)';
COMMENT ON COLUMN conversations.platform_metadata IS 'Platform-specific data stored as JSON (group invite links, etc.)';
COMMENT ON COLUMN messages.external_message_id IS 'Platform-specific message identifier for deduplication';
COMMENT ON COLUMN messages.platform_metadata IS 'Platform-specific data stored as JSON (reactions, mentions, etc.)';
COMMENT ON COLUMN contacts.external_contact_id IS 'Platform-specific contact identifier (JID, user ID, etc.)';