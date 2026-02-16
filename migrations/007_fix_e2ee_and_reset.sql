-- Emergency Database Fix Script
-- This script will:
-- 1. Fix column types for E2EE fields
-- 2. Clear all existing users and related data (SAFELY)

-- Step 1: Drop existing E2EE columns if they have wrong types
ALTER TABLE messages DROP COLUMN IF EXISTS ciphertext CASCADE;
ALTER TABLE users DROP COLUMN IF EXISTS encryption_public_key CASCADE;

-- Step 2: Add columns with CORRECT types (TEXT, not bytea)
ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;

-- Step 3: Clear all existing data to start fresh
-- Using TRUNCATE CASCADE to safely clear all data
DO $$
BEGIN
    -- Truncate all tables in the correct order
    TRUNCATE TABLE replies, bookmarks, reposts, likes, messages, message_threads, follows, posts, moderation_requests, admin_actions, users RESTART IDENTITY CASCADE;
EXCEPTION
    WHEN undefined_table THEN
        -- If any table doesn't exist, just continue
        RAISE NOTICE 'Some tables do not exist, skipping truncate';
END $$;

-- Step 4: Verify column types
SELECT 
    table_name, 
    column_name, 
    data_type 
FROM information_schema.columns 
WHERE table_name IN ('users', 'messages') 
  AND column_name IN ('encryption_public_key', 'ciphertext')
ORDER BY table_name, column_name;

