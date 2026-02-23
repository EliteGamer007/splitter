-- Migration 002: Consolidated upgrade to current production schema
-- Safe for legacy databases (uses IF NOT EXISTS / guarded updates)
-- Non-destructive: no data reset/truncate operations

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ------------------------------------------------------------------
-- USERS
-- ------------------------------------------------------------------
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin'));
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderation_requested BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS moderation_requested_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS message_privacy TEXT DEFAULT 'everyone';
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_visibility TEXT DEFAULT 'public';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_data BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_media_type TEXT;

-- Allow password-based auth users without DID/public key
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'did' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE users ALTER COLUMN did DROP NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'public_key' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE users ALTER COLUMN public_key DROP NOT NULL;
    END IF;
END $$;

UPDATE users SET message_privacy = 'everyone' WHERE COALESCE(message_privacy, '') = '';
UPDATE users SET default_visibility = 'public' WHERE COALESCE(default_visibility, '') = '';

-- ------------------------------------------------------------------
-- MODERATION
-- ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS moderation_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reason TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ------------------------------------------------------------------
-- MESSAGING
-- ------------------------------------------------------------------
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS participant_a_id UUID REFERENCES users(id);
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS participant_b_id UUID REFERENCES users(id);
ALTER TABLE message_threads ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_id UUID REFERENCES users(id);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS recipient_id UUID REFERENCES users(id);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ;

-- ------------------------------------------------------------------
-- POSTS / REPLIES / MEDIA
-- ------------------------------------------------------------------
ALTER TABLE posts ADD COLUMN IF NOT EXISTS is_remote BOOLEAN DEFAULT FALSE;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS original_post_uri TEXT;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS direct_reply_count INT DEFAULT 0;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS total_reply_count INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS replies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES replies(id) ON DELETE CASCADE,
    author_did TEXT NOT NULL,
    content TEXT NOT NULL,
    depth INT NOT NULL,
    likes_count INT DEFAULT 0,
    direct_reply_count INT DEFAULT 0,
    total_reply_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

ALTER TABLE media ADD COLUMN IF NOT EXISTS media_data BYTEA;

-- ------------------------------------------------------------------
-- FEDERATION
-- ------------------------------------------------------------------
ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ;
ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS last_error TEXT;

UPDATE outbox_activities
SET next_retry_at = COALESCE(next_retry_at, now())
WHERE status IN ('pending','failed');

CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS federation_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_domain TEXT NOT NULL,
    target_domain TEXT NOT NULL,
    success_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    last_status TEXT CHECK (last_status IN ('sent', 'failed', 'pending')),
    last_seen TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(source_domain, target_domain)
);

-- remote_actors evolved across migrations; keep backward-compatible upgrades
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS domain TEXT;
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS display_name TEXT DEFAULT '';
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS public_key_pem TEXT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'remote_actors' AND column_name = 'instance_domain'
    ) THEN
        EXECUTE 'UPDATE remote_actors SET domain = instance_domain WHERE COALESCE(domain, '''') = ''''';
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'remote_actors' AND column_name = 'public_key'
    ) THEN
        EXECUTE 'UPDATE remote_actors SET public_key_pem = public_key WHERE COALESCE(public_key_pem, '''') = '''' AND COALESCE(public_key, '''') <> ''''';
    END IF;
END $$;

-- ------------------------------------------------------------------
-- INDEXES
-- ------------------------------------------------------------------
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
CREATE INDEX IF NOT EXISTS idx_replies_post_id ON replies(post_id);
CREATE INDEX IF NOT EXISTS idx_replies_parent_id ON replies(parent_id);
CREATE INDEX IF NOT EXISTS idx_replies_author_did ON replies(author_did);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_remote_actors_domain ON remote_actors(domain);
CREATE INDEX IF NOT EXISTS idx_remote_actors_username_domain ON remote_actors(username, domain);
CREATE INDEX IF NOT EXISTS idx_outbox_next_retry ON outbox_activities(next_retry_at) WHERE status IN ('pending','failed');
CREATE INDEX IF NOT EXISTS idx_federation_failures_circuit_until ON federation_failures(circuit_open_until);
CREATE INDEX IF NOT EXISTS idx_federation_connections_source ON federation_connections(source_domain);
CREATE INDEX IF NOT EXISTS idx_federation_connections_target ON federation_connections(target_domain);

-- ------------------------------------------------------------------
-- TRIGGERS / COMMENTS
-- ------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

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
