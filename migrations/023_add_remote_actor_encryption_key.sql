ALTER TABLE remote_actors
    ADD COLUMN IF NOT EXISTS encryption_public_key TEXT;
