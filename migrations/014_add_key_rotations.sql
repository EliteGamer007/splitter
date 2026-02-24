-- Migration 014: Add key rotation history and revocation list
-- Supports Story 4.3: Cryptographic Key Rotation & Revocation

CREATE TABLE IF NOT EXISTS key_rotations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_public_key TEXT NOT NULL,   -- Revoked Ed25519 public key (base64)
    new_public_key TEXT NOT NULL,   -- Replacement Ed25519 public key (base64)
    rotated_at TIMESTAMPTZ DEFAULT now(),
    nonce TEXT NOT NULL,            -- Random UUID to prevent replay attacks
    UNIQUE(nonce)
);

CREATE INDEX IF NOT EXISTS idx_key_rotations_user_id ON key_rotations(user_id);
CREATE INDEX IF NOT EXISTS idx_key_rotations_old_key ON key_rotations(old_public_key);
CREATE INDEX IF NOT EXISTS idx_key_rotations_rotated_at ON key_rotations(rotated_at DESC);

COMMENT ON TABLE key_rotations IS 'Tracks public key rotation history and maintains a revocation list per user';
COMMENT ON COLUMN key_rotations.old_public_key IS 'The revoked key - any message signed with this key should be rejected';
COMMENT ON COLUMN key_rotations.nonce IS 'One-time nonce to prevent rotation request replay attacks';
