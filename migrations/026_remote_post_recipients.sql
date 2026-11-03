-- Preserve explicit audiences for incoming restricted ActivityPub posts.
CREATE TABLE IF NOT EXISTS remote_post_recipients (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    recipient_did TEXT NOT NULL,
    PRIMARY KEY(post_id, recipient_did)
);
