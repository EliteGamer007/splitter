-- AI Moderation Migration
-- Adds AI screening results, appeal tracking, and hidden-by-AI post markers

-- Extend reports table with AI screening fields
ALTER TABLE reports ADD COLUMN IF NOT EXISTS ai_verdict TEXT;
ALTER TABLE reports ADD COLUMN IF NOT EXISTS ai_reason TEXT;
ALTER TABLE reports ADD COLUMN IF NOT EXISTS ai_screened_at TIMESTAMPTZ;

-- Widen status constraint to include AI-actioned states
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_status_check;
ALTER TABLE reports ADD CONSTRAINT reports_status_check
  CHECK (status IN ('pending', 'resolved', 'ai_actioned', 'dismissed'));

-- Mark posts that were hidden by the AI moderator
ALTER TABLE posts ADD COLUMN IF NOT EXISTS hidden_by_ai BOOLEAN DEFAULT FALSE;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS hidden_reason TEXT;

-- Appeals table: users contesting AI-removed content
CREATE TABLE IF NOT EXISTS appeals (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  report_id UUID REFERENCES reports(id) ON DELETE CASCADE,
  post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
  appellant_did TEXT NOT NULL,
  appellant_id UUID REFERENCES users(id) ON DELETE SET NULL,
  appeal_reason TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
  reviewer_id TEXT,
  reviewer_note TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_appeals_status ON appeals(status);
CREATE INDEX IF NOT EXISTS idx_appeals_post_id ON appeals(post_id);
CREATE INDEX IF NOT EXISTS idx_reports_ai_verdict ON reports(ai_verdict);
