-- Add sequence numbers to syncable tables for efficient cursor-based syncing
-- This allows clients to sync data incrementally without missing updates
-- Add seq column to conversations
ALTER TABLE conversations
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_conversations_seq ON conversations (seq);
-- Add seq column to messages
ALTER TABLE messages
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_messages_seq ON messages (seq);
-- Add seq column to contacts
ALTER TABLE contacts
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_contacts_seq ON contacts (seq);
-- Add seq column to message_media
ALTER TABLE message_media
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_message_media_seq ON message_media (seq);
-- Add seq column to conversation_participants
ALTER TABLE conversation_participants
ADD COLUMN seq BIGSERIAL;
CREATE INDEX idx_conversation_participants_seq ON conversation_participants (seq);
-- Comments
COMMENT ON COLUMN conversations.seq IS 'Auto-incrementing sequence number for sync cursoring';
COMMENT ON COLUMN messages.seq IS 'Auto-incrementing sequence number for sync cursoring';
COMMENT ON COLUMN contacts.seq IS 'Auto-incrementing sequence number for sync cursoring';
COMMENT ON COLUMN message_media.seq IS 'Auto-incrementing sequence number for sync cursoring';
COMMENT ON COLUMN conversation_participants.seq IS 'Auto-incrementing sequence number for sync cursoring';