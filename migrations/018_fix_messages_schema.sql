-- Migration 018: Fix messages table schema
-- Adds missing columns required by internal/repository/message_repo.go
-- that are not present in the original 001_initial_schema.sql definition.

ALTER TABLE messages
ADD COLUMN IF NOT EXISTS sender_id UUID,
ADD COLUMN IF NOT EXISTS recipient_id UUID,
ADD COLUMN IF NOT EXISTS content TEXT,
ADD COLUMN IF NOT EXISTS encrypted_keys JSONB,
ADD COLUMN IF NOT EXISTS client_message_id TEXT,
ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS client_created_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ;

-- Index to support idempotent message sync
CREATE INDEX IF NOT EXISTS idx_messages_sender_client_msg
ON messages(sender_id, client_message_id);
