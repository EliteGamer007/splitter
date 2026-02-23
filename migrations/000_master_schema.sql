-- MASTER MIGRATION - Complete Splitter Database Schema
-- Use this file for fresh database setup (like Neon cloud database)
-- This consolidates all migrations into a single, clean schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- USERS & IDENTITY
-- ============================================================

-- Local user accounts (supports both DID and password-based auth)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT UNIQUE,  -- For password-based auth
    password_hash TEXT,  -- Bcrypt hash for password auth
    instance_domain TEXT NOT NULL,
    did TEXT UNIQUE,  -- Optional for password users
    display_name TEXT,
    bio TEXT,
    avatar_url TEXT,
    avatar_data BYTEA,
    avatar_media_type TEXT,
    public_key TEXT,  -- Optional for password users
    encryption_public_key TEXT DEFAULT '',  -- E2EE encryption public key (ECDH)
    message_privacy TEXT DEFAULT 'everyone',
    default_visibility TEXT DEFAULT 'public',
    role TEXT DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin')),
    moderation_requested BOOLEAN DEFAULT FALSE,
    moderation_requested_at TIMESTAMPTZ,
    is_locked BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Federated remote users
CREATE TABLE remote_actors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_uri TEXT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    domain TEXT NOT NULL,
    inbox_url TEXT NOT NULL,
    outbox_url TEXT,
    public_key_pem TEXT,
    display_name TEXT DEFAULT '',
    avatar_url TEXT DEFAULT '',
    last_fetched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Instance RSA keypairs for HTTP signatures
CREATE TABLE instance_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Public keys and devices
CREATE TABLE user_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    key_type TEXT NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- SOCIAL RELATIONSHIPS
-- ============================================================

-- Follow relationships
CREATE TABLE follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_did TEXT NOT NULL,
    following_did TEXT NOT NULL,
    status TEXT DEFAULT 'accepted' CHECK (status IN ('pending','accepted','rejected')),
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- CONTENT & POSTS
-- ============================================================

-- Posts and replies
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_did TEXT NOT NULL,
    content TEXT,
    visibility TEXT DEFAULT 'public' CHECK (visibility IN ('public','followers','circle')),
    is_remote BOOLEAN DEFAULT FALSE,
    original_post_uri TEXT,
    direct_reply_count INT DEFAULT 0,
    total_reply_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

-- Post media attachments
CREATE TABLE media (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    media_data BYTEA,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Likes and reposts
CREATE TABLE interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    actor_did TEXT NOT NULL,
    interaction_type TEXT CHECK (interaction_type IN ('like','repost')),
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(post_id, actor_did, interaction_type)
);

-- Reply threads
CREATE TABLE replies (
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

-- Private saved posts
CREATE TABLE bookmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, post_id)
);

-- ============================================================
-- MESSAGING (E2EE)
-- ============================================================

-- Message conversation threads
CREATE TABLE message_threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    participant_a TEXT,
    participant_b TEXT,
    participant_a_id UUID REFERENCES users(id),
    participant_b_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Encrypted direct messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID REFERENCES message_threads(id) ON DELETE CASCADE,
    sender_did TEXT,
    recipient_did TEXT,
    sender_id UUID REFERENCES users(id),
    recipient_id UUID REFERENCES users(id),
    ciphertext TEXT,  -- E2EE encrypted content
    content TEXT,  -- For unencrypted messages
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    delivered_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,  -- WhatsApp-style soft delete
    edited_at TIMESTAMPTZ    -- Message edit timestamp
);

-- ============================================================
-- FEDERATION ENGINE
-- ============================================================

-- Incoming federation activities
CREATE TABLE inbox_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    activity_id TEXT UNIQUE NOT NULL,
    actor_uri TEXT NOT NULL,
    activity_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ DEFAULT now()
);

-- Outgoing federation activities
CREATE TABLE outbox_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    activity_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    target_inbox TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending','sent','failed')),
    retry_count INT DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Activity deduplication cache
CREATE TABLE activity_deduplication (
    activity_id TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ
);

-- ============================================================
-- GOVERNANCE & MODERATION
-- ============================================================

-- Moderation requests
CREATE TABLE moderation_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reason TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Blocked remote servers
CREATE TABLE blocked_domains (
    domain TEXT PRIMARY KEY,
    reason TEXT,
    blocked_by TEXT,
    blocked_at TIMESTAMPTZ DEFAULT now()
);

-- Reported content queue
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reporter_did TEXT NOT NULL,
    post_id UUID REFERENCES posts(id),
    reason TEXT,
    status TEXT CHECK (status IN ('pending','resolved')),
    created_at TIMESTAMPTZ DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

-- Admin action audit log
CREATE TABLE admin_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    target TEXT,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Remote server reputation
CREATE TABLE instance_reputation (
    domain TEXT PRIMARY KEY,
    reputation_score INT DEFAULT 0,
    spam_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Federation failure tracking
CREATE TABLE federation_failures (
    domain TEXT PRIMARY KEY,
    failure_count INT DEFAULT 0,
    last_failure_at TIMESTAMPTZ,
    circuit_open_until TIMESTAMPTZ
);

-- Federation server-to-server connection graph data
CREATE TABLE federation_connections (
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

-- ============================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================

-- User indexes
CREATE INDEX idx_users_did ON users(did);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_role ON users(role);

-- Post indexes
CREATE INDEX idx_posts_author ON posts(author_did);
CREATE INDEX idx_posts_created ON posts(created_at DESC);
CREATE INDEX idx_posts_visibility ON posts(visibility);

-- Follow indexes
CREATE INDEX idx_follows_follower_status ON follows(follower_did, status);
CREATE INDEX idx_follows_following_status ON follows(following_did, status);

-- Message indexes
CREATE INDEX idx_messages_thread ON messages(thread_id);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_recipient ON messages(recipient_id);
CREATE INDEX idx_message_threads_participants ON message_threads(participant_a_id, participant_b_id);

-- Federation indexes
CREATE INDEX idx_inbox_actor ON inbox_activities(actor_uri);
CREATE INDEX idx_outbox_status ON outbox_activities(status);
CREATE INDEX idx_outbox_next_retry ON outbox_activities(next_retry_at) WHERE status IN ('pending','failed');
CREATE INDEX idx_remote_actors_domain ON remote_actors(domain);
CREATE INDEX idx_remote_actors_username_domain ON remote_actors(username, domain);
CREATE INDEX idx_federation_failures_circuit_until ON federation_failures(circuit_open_until);
CREATE INDEX idx_federation_connections_source ON federation_connections(source_domain);
CREATE INDEX idx_federation_connections_target ON federation_connections(target_domain);

-- Reply indexes
CREATE INDEX idx_replies_post_id ON replies(post_id);
CREATE INDEX idx_replies_parent_id ON replies(parent_id);
CREATE INDEX idx_replies_author_did ON replies(author_did);

-- Message deletion index
CREATE INDEX idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NOT NULL;

-- Moderation indexes
CREATE INDEX idx_moderation_requests_status ON moderation_requests(status);
CREATE INDEX idx_admin_actions_admin_id ON admin_actions(admin_id);
CREATE INDEX idx_admin_actions_created_at ON admin_actions(created_at DESC);

-- ============================================================
-- TRIGGERS
-- ============================================================

-- Function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for message_threads table
CREATE TRIGGER update_message_threads_updated_at
    BEFORE UPDATE ON message_threads
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- COMMENTS/DOCUMENTATION
-- ============================================================

COMMENT ON TABLE users IS 'User accounts with support for both DID and email/password authentication';
COMMENT ON TABLE moderation_requests IS 'Requests from users to become moderators';
COMMENT ON TABLE message_threads IS 'Direct message conversation threads between users';
COMMENT ON TABLE messages IS 'Individual messages within threads';
COMMENT ON TABLE follows IS 'Follow relationships between users';
COMMENT ON TABLE posts IS 'User-generated content/posts';
COMMENT ON TABLE interactions IS 'User interactions with posts (likes, reposts)';
COMMENT ON TABLE bookmarks IS 'User-saved posts';
COMMENT ON TABLE admin_actions IS 'Audit log of all admin/moderator actions';

COMMENT ON COLUMN users.email IS 'User email for password-based authentication';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.role IS 'User role: user, moderator, or admin';
COMMENT ON COLUMN users.did IS 'Decentralized Identifier - optional for password auth users';
COMMENT ON COLUMN users.public_key IS 'Ed25519 public key - optional for password auth users';
