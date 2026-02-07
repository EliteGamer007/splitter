-- Create replies table
CREATE TABLE replies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES replies(id) ON DELETE CASCADE, -- NULL for top-level reply
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

-- Add indexes
CREATE INDEX idx_replies_post_id ON replies(post_id);
CREATE INDEX idx_replies_parent_id ON replies(parent_id);
CREATE INDEX idx_replies_author_did ON replies(author_did);

-- Add reply counters to posts table
ALTER TABLE posts ADD COLUMN direct_reply_count INT DEFAULT 0;
ALTER TABLE posts ADD COLUMN total_reply_count INT DEFAULT 0;
