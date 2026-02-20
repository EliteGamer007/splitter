-- Migration 013: Message privacy and profile default visibility settings

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS message_privacy TEXT DEFAULT 'everyone';

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS default_visibility TEXT DEFAULT 'public';

UPDATE users
SET message_privacy = 'everyone'
WHERE message_privacy IS NULL OR message_privacy = '';

UPDATE users
SET default_visibility = 'public'
WHERE default_visibility IS NULL OR default_visibility = '';
