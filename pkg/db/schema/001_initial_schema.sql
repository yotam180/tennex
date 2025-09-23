-- Initial schema for Tennex backend
-- This creates the core tables for event sourcing and message handling

-- Events table: append-only, authoritative source of truth
CREATE TABLE events (
    seq           BIGSERIAL PRIMARY KEY,
    id            UUID NOT NULL UNIQUE,
    ts            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type          TEXT NOT NULL,
    account_id    TEXT NOT NULL,
    device_id     TEXT,
    convo_id      TEXT NOT NULL,
    wa_message_id TEXT,
    sender_jid    TEXT,
    payload       JSONB NOT NULL,
    attachment_ref JSONB
);

-- Indexes for efficient querying
CREATE INDEX idx_events_account_seq ON events (account_id, seq);
CREATE INDEX idx_events_convo_seq ON events (convo_id, seq);
CREATE INDEX idx_events_type_ts ON events (type, ts);
CREATE INDEX idx_events_wa_message_id ON events (wa_message_id) WHERE wa_message_id IS NOT NULL;

-- Outbox table: durable send path for outgoing messages
CREATE TABLE outbox (
    client_msg_uuid UUID PRIMARY KEY,
    account_id      TEXT NOT NULL,
    convo_id        TEXT NOT NULL,
    server_msg_id   BIGINT REFERENCES events(seq),
    status          TEXT NOT NULL DEFAULT 'queued',
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for outbox processing
CREATE INDEX idx_outbox_status_created ON outbox (status, created_at) WHERE status IN ('queued', 'retry');
CREATE INDEX idx_outbox_account_status ON outbox (account_id, status);

-- Accounts table: bind WhatsApp JID to account
CREATE TABLE accounts (
    id          TEXT PRIMARY KEY,
    wa_jid      TEXT UNIQUE,
    display_name TEXT,
    avatar_url   TEXT,
    status       TEXT NOT NULL DEFAULT 'disconnected',
    last_seen    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Media blobs table: content-addressed storage references
CREATE TABLE media_blobs (
    content_hash   TEXT PRIMARY KEY,
    mime_type      TEXT NOT NULL,
    size_bytes     BIGINT NOT NULL,
    storage_url    TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add constraints and checks
ALTER TABLE events ADD CONSTRAINT events_type_check 
    CHECK (type IN ('msg_in', 'msg_out_pending', 'msg_out_sent', 'msg_delivery', 'presence', 'contact_update', 'history_sync'));

ALTER TABLE outbox ADD CONSTRAINT outbox_status_check 
    CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'retry'));

ALTER TABLE accounts ADD CONSTRAINT accounts_status_check 
    CHECK (status IN ('connected', 'disconnected', 'connecting', 'error'));

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_outbox_updated_at BEFORE UPDATE ON outbox 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE events IS 'Append-only event log, source of truth for all messages and state changes';
COMMENT ON TABLE outbox IS 'Reliable message sending queue with transactional guarantees';
COMMENT ON TABLE accounts IS 'WhatsApp account bindings and status';
COMMENT ON TABLE media_blobs IS 'Content-addressed media storage references';

COMMENT ON COLUMN events.seq IS 'Auto-incrementing sequence number for cursoring';
COMMENT ON COLUMN events.id IS 'Globally unique event identifier for idempotency';
COMMENT ON COLUMN events.wa_message_id IS 'WhatsApp message ID for deduplication';
COMMENT ON COLUMN outbox.client_msg_uuid IS 'Client-provided UUID for request idempotency';
COMMENT ON COLUMN outbox.server_msg_id IS 'Reference to the events.seq for the outbound message';
