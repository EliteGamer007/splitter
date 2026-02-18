-- Migration 011: Fix remote_actors schema to match code expectations
-- The code references columns: domain, display_name, avatar_url, public_key_pem
-- But the actual table has: instance_domain, did, public_key (no display_name, avatar_url)

-- Add domain column as alias for instance_domain (or rename)
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS domain TEXT DEFAULT '';
UPDATE remote_actors SET domain = instance_domain WHERE domain = '' OR domain IS NULL;

-- Add missing columns
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS display_name TEXT DEFAULT '';
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS public_key_pem TEXT DEFAULT '';

-- Copy public_key to public_key_pem if needed
UPDATE remote_actors SET public_key_pem = public_key WHERE public_key_pem = '' AND public_key IS NOT NULL AND public_key != '';

-- Index on domain
CREATE INDEX IF NOT EXISTS idx_remote_actors_domain ON remote_actors(domain);
CREATE INDEX IF NOT EXISTS idx_remote_actors_username_domain ON remote_actors(username, domain);
