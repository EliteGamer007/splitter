-- Migration: Safely add email and password_hash columns if they are missing
-- This fixes the "column email does not exist" error without touching existing ambiguous migrations.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
