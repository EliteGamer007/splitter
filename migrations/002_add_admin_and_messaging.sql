-- Migration 002: Add admin roles, moderation requests, and fix messaging
-- Run this migration after 001_initial_schema.sql

-- Add email and password fields to users (if not already present)
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Add role field to users (user, moderator, admin)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin'));

-- Add moderation request field
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderation_requested BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderation_requested_at TIMESTAMPTZ;

-- Make DID optional (nullable) for password-based auth
ALTER TABLE users ALTER COLUMN did DROP NOT NULL;
ALTER TABLE users ALTER COLUMN public_key DROP NOT NULL;

-- Create moderation requests table
CREATE TABLE IF NOT EXISTS moderation_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reason TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Fix message_threads to use user IDs instead of DIDs for easier querying
-- Drop and recreate if needed (or add new columns)
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS participant_a_id UUID REFERENCES users(id);
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS participant_b_id UUID REFERENCES users(id);
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

-- Fix messages table to use user IDs
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_id UUID REFERENCES users(id);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS recipient_id UUID REFERENCES users(id);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content TEXT; -- For unencrypted messages (encryption optional)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE;

-- Create index for faster message lookups
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient_id);
CREATE INDEX IF NOT EXISTS idx_message_threads_participants ON message_threads(participant_a_id, participant_b_id);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Note: Admin user (username: admin, password: splitteradmin) is created automatically
-- by the backend on first startup if it doesn't exist.
