-- Migration: Add E2E Encryption Fields
-- Description: Adds encryption_public_key to users table and ciphertext to messages table

-- Add encryption_public_key to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';

-- Add ciphertext to messages
-- We keep content for backward compatibility or system messages, but E2EE messages will use ciphertext
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;

-- Add index for performance if we ever query by encryption key (unlikely but good for uniqueness if needed)
-- CREATE INDEX idx_users_encryption_key ON users(encryption_public_key);
