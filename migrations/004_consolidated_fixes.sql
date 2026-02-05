-- Migration 004: Consolidated fixes for complete schema
-- This migration ensures all necessary columns and tables exist
-- Safe to run even if some items already exist (uses IF NOT EXISTS / IF EXISTS)

-- Ensure email and password_hash exist in users table
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='email') THEN
        ALTER TABLE users ADD COLUMN email TEXT UNIQUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='password_hash') THEN
        ALTER TABLE users ADD COLUMN password_hash TEXT;
    END IF;
END $$;

-- Ensure role column exists with proper constraints
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='role') THEN
        ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin'));
    END IF;
END $$;

-- Ensure moderation request columns exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='moderation_requested') THEN
        ALTER TABLE users ADD COLUMN moderation_requested BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='moderation_requested_at') THEN
        ALTER TABLE users ADD COLUMN moderation_requested_at TIMESTAMPTZ;
    END IF;
END $$;

-- Make DID and public_key nullable for password-based auth
ALTER TABLE users ALTER COLUMN did DROP NOT NULL;
ALTER TABLE users ALTER COLUMN public_key DROP NOT NULL;

-- Create moderation_requests table if it doesn't exist
CREATE TABLE IF NOT EXISTS moderation_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reason TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Enhance message_threads with user IDs for easier querying
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='message_threads' AND column_name='participant_a_id') THEN
        ALTER TABLE message_threads ADD COLUMN participant_a_id UUID REFERENCES users(id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='message_threads' AND column_name='participant_b_id') THEN
        ALTER TABLE message_threads ADD COLUMN participant_b_id UUID REFERENCES users(id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='message_threads' AND column_name='updated_at') THEN
        ALTER TABLE message_threads ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();
    END IF;
END $$;

-- Enhance messages table
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='messages' AND column_name='sender_id') THEN
        ALTER TABLE messages ADD COLUMN sender_id UUID REFERENCES users(id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='messages' AND column_name='recipient_id') THEN
        ALTER TABLE messages ADD COLUMN recipient_id UUID REFERENCES users(id);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='messages' AND column_name='content') THEN
        ALTER TABLE messages ADD COLUMN content TEXT;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='messages' AND column_name='is_read') THEN
        ALTER TABLE messages ADD COLUMN is_read BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Create essential indexes for performance
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient_id);
CREATE INDEX IF NOT EXISTS idx_message_threads_participants ON message_threads(participant_a_id, participant_b_id);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_moderation_requests_status ON moderation_requests(status);
CREATE INDEX IF NOT EXISTS idx_follows_follower_status ON follows(follower_did, status);
CREATE INDEX IF NOT EXISTS idx_follows_following_status ON follows(following_did, status);
CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility);
CREATE INDEX IF NOT EXISTS idx_admin_actions_admin_id ON admin_actions(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_actions_created_at ON admin_actions(created_at DESC);

-- Update updated_at timestamp function for users
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for users table
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create trigger for message_threads table
DROP TRIGGER IF EXISTS update_message_threads_updated_at ON message_threads;
CREATE TRIGGER update_message_threads_updated_at
    BEFORE UPDATE ON message_threads
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE users IS 'User accounts with support for both DID and email/password authentication';
COMMENT ON TABLE moderation_requests IS 'Requests from users to become moderators';
COMMENT ON TABLE message_threads IS 'Direct message conversation threads between users';
COMMENT ON TABLE messages IS 'Individual messages within threads';
COMMENT ON COLUMN users.email IS 'User email for password-based authentication';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.role IS 'User role: user, moderator, or admin';
COMMENT ON COLUMN users.did IS 'Decentralized Identifier - optional for password auth users';
