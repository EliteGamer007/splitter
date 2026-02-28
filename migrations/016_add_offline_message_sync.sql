-- Story 4.9 backend support: idempotent offline message sync metadata
-- Allows client-queued encrypted DMs to sync safely after reconnect.

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS client_message_id TEXT,
    ADD COLUMN IF NOT EXISTS client_created_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_sender_client_message_id
    ON messages(sender_id, client_message_id)
    WHERE client_message_id IS NOT NULL;