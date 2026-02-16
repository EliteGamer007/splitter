-- Add deleted_at and edited_at columns to messages table for WhatsApp-style delete/edit
-- Deleted messages will show "You deleted this message" instead of content
-- Edited messages will show "✏️ Edited" indicator
-- Both features only work within 3 hours of sending

ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ DEFAULT NULL;

-- Create index for performance on deleted messages queries
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NOT NULL;
