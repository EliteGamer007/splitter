-- Migration 012: Move media storage to database (BYTEA)
-- Adds DB-backed storage for post media and user avatars while keeping URL compatibility.

ALTER TABLE media ADD COLUMN IF NOT EXISTS media_data BYTEA;

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_data BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_media_type TEXT;
