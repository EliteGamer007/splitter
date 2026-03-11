-- Migration 024: Add circle_members table for close friends / restricted visibility
-- Circle members = users that a given owner adds to their "circle"
-- Posts with visibility='circle' are only visible to the author + their circle members

CREATE TABLE IF NOT EXISTS circle_members (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (owner_id, member_id)
);

CREATE INDEX IF NOT EXISTS idx_circle_members_owner  ON circle_members (owner_id);
CREATE INDEX IF NOT EXISTS idx_circle_members_member ON circle_members (member_id);
