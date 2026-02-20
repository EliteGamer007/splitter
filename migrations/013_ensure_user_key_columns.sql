-- Migration 013: Ensure user key/auth columns exist across legacy databases

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT;
