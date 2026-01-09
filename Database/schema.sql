-- Splitter Database Schema
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Identity & Users(Epic 1)

-- Local user accounts
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    instance_domain TEXT NOT NULL,
    did TEXT UNIQUE NOT NULL,
    display_name TEXT,
    bio TEXT,
    avatar_url TEXT,
    public_key TEXT NOT NULL,
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
    instance_domain TEXT NOT NULL,
    did TEXT,
    public_key TEXT NOT NULL,
    inbox_url TEXT NOT NULL,
    outbox_url TEXT NOT NULL,
    last_fetched_at TIMESTAMPTZ,
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


-- Follow relationships
CREATE TABLE follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_did TEXT NOT NULL,
    following_did TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending','accepted','rejected')),
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Content & Streams(Epic 3)

-- Posts and replies
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_did TEXT NOT NULL,
    content TEXT,
    visibility TEXT CHECK (visibility IN ('public','followers','circle')),
    is_remote BOOLEAN DEFAULT FALSE,
    original_post_uri TEXT,
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

-- Private saved posts
CREATE TABLE bookmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, post_id)
);

-- Messaging (E2EE)(Epic 4)

-- Message conversation threads
CREATE TABLE message_threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    participant_a TEXT NOT NULL,
    participant_b TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Encrypted direct messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID REFERENCES message_threads(id) ON DELETE CASCADE,
    sender_did TEXT NOT NULL,
    recipient_did TEXT NOT NULL,
    ciphertext BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    delivered_at TIMESTAMPTZ
);

-- Federation Engine(Epic 2)

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
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Activity deduplication cache
CREATE TABLE activity_deduplication (
    activity_id TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ
);

-- Governance & Resilience(Epic 5)

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

--Indexes
CREATE INDEX idx_users_did ON users(did);
CREATE INDEX idx_posts_author ON posts(author_did);
CREATE INDEX idx_posts_created ON posts(created_at DESC);
CREATE INDEX idx_inbox_actor ON inbox_activities(actor_uri);
CREATE INDEX idx_outbox_status ON outbox_activities(status);
CREATE INDEX idx_messages_thread ON messages(thread_id);
