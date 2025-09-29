-- Fix foreign key constraint on messages.reply_to_message_id
-- This constraint is too strict for async message syncing where reply messages
-- might arrive before the original message they're replying to.
-- 
-- Since we have reply_to_external_id to track the original message's external ID,
-- we can safely remove the strict foreign key and use it as a soft reference.
-- Drop the existing foreign key constraint
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_reply_to_message_id_fkey;
-- Optionally, add an index to help with lookups (but no strict enforcement)
CREATE INDEX IF NOT EXISTS idx_messages_reply_to ON messages(reply_to_message_id)
WHERE reply_to_message_id IS NOT NULL;
-- Add comment explaining the soft reference
COMMENT ON COLUMN messages.reply_to_message_id IS 'Soft reference to the message being replied to. May be NULL if the original message is not in the database. Use reply_to_external_id for external message references.';
COMMENT ON COLUMN messages.reply_to_external_id IS 'External platform ID of the message being replied to. This is the canonical reference for threading.';