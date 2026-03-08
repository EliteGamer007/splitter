-- Migration 019: Add stories table for Instagram-like story feature
-- Stories are images visible to followers for 24 hours

CREATE TABLE IF NOT EXISTS stories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_did TEXT NOT NULL,
    media_data BYTEA NOT NULL,
    media_type TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ DEFAULT (now() + interval '24 hours')
);

-- Indexes for efficient feed queries
CREATE INDEX IF NOT EXISTS idx_stories_author_did ON stories(author_did);
CREATE INDEX IF NOT EXISTS idx_stories_expires_at ON stories(expires_at);
CREATE INDEX IF NOT EXISTS idx_stories_author_expires ON stories(author_did, expires_at DESC);

COMMENT ON TABLE stories IS 'Instagram-like stories: images visible to followers for 24 hours';
