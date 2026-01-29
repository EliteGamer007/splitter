-- Migration: Add password-based authentication
-- This adds email and password_hash fields while keeping DID as secondary auth

-- Add email and password_hash columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Make DID and public_key optional (will be auto-generated if not provided)
ALTER TABLE users ALTER COLUMN did DROP NOT NULL;
ALTER TABLE users ALTER COLUMN public_key DROP NOT NULL;

-- Add unique constraint on email
CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users(email) WHERE email IS NOT NULL;

-- Add index for username lookups
CREATE INDEX IF NOT EXISTS users_username_idx ON users(username);
