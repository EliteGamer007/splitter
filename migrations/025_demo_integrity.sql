-- Additive repairs only: preserve existing accounts, posts, keys and ciphertext.
CREATE TABLE IF NOT EXISTS reply_interactions (
    reply_id UUID NOT NULL REFERENCES replies(id) ON DELETE CASCADE,
    actor_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (reply_id, actor_did)
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_locked BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS message_privacy TEXT NOT NULL DEFAULT 'everyone';
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_visibility TEXT NOT NULL DEFAULT 'public';
ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';
ALTER TABLE key_rotations ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT 'rotated';
CREATE INDEX IF NOT EXISTS idx_stories_expires ON stories(expires_at);
CREATE INDEX IF NOT EXISTS idx_posts_active_author_created ON posts(author_did,created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_follows_pending_target ON follows(following_did,created_at DESC) WHERE status='pending';
