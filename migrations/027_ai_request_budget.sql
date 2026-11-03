-- Budget actual user-triggered AI attempts across restarts; never seed activity.
CREATE TABLE IF NOT EXISTS ai_bot_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_did TEXT NOT NULL,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    mention_id TEXT NOT NULL UNIQUE,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ai_bot_requests_requested_at ON ai_bot_requests(requested_at);
