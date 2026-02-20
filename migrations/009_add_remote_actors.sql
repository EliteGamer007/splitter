-- Migration 009: Add remote actors and instance keys for federation
-- These tables are required for ActivityPub federation between instances

-- Remote actors cache (federated users from other instances)
CREATE TABLE IF NOT EXISTS remote_actors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_uri TEXT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    domain TEXT NOT NULL,
    inbox_url TEXT NOT NULL,
    outbox_url TEXT,
    public_key_pem TEXT,
    display_name TEXT DEFAULT '',
    avatar_url TEXT DEFAULT '',
    last_fetched_at TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Instance RSA keypairs for HTTP Signature signing
CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_remote_actors_domain ON remote_actors(domain);
CREATE INDEX IF NOT EXISTS idx_remote_actors_username_domain ON remote_actors(username, domain);
