-- Migration 010: Federation Fix - Ensure all federation tables exist
-- This migration ensures all federation-related tables are present
-- and adds any missing columns to support federation between instances

-- Instance RSA keypairs for HTTP Signature signing (PRIMARY FIX)
CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Ensure users table has instance_domain column
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='instance_domain') THEN
        ALTER TABLE users ADD COLUMN instance_domain TEXT DEFAULT 'localhost';
    END IF;
END $$;

-- Update instance_domain for existing local users if null
UPDATE users SET instance_domain = COALESCE(instance_domain, 'localhost') WHERE instance_domain IS NULL OR instance_domain = '';

-- Ensure posts table has is_remote column for federated posts
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='posts' AND column_name='is_remote') THEN
        ALTER TABLE posts ADD COLUMN is_remote BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Ensure posts table has original_post_uri for federated content
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='posts' AND column_name='original_post_uri') THEN
        ALTER TABLE posts ADD COLUMN original_post_uri TEXT;
    END IF;
END $$;
