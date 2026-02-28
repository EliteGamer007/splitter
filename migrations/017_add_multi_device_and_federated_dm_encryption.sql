-- Migration 017: Multi-device key authorization + encrypted DM envelopes

CREATE TABLE IF NOT EXISTS user_device_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    device_label TEXT DEFAULT '',
    encryption_public_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'revoked')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    approved_by_device_id TEXT,
    last_seen_at TIMESTAMPTZ,
    UNIQUE(user_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_user_device_keys_user_id ON user_device_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_user_device_keys_status ON user_device_keys(status);

ALTER TABLE messages ADD COLUMN IF NOT EXISTS encrypted_keys JSONB;
CREATE INDEX IF NOT EXISTS idx_messages_encrypted_keys_gin ON messages USING GIN (encrypted_keys);
