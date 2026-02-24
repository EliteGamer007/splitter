-- Migration 015: Add reason column to key_rotations revocation list
-- Allows recording WHY a key was revoked (rotated, compromised, etc.)

ALTER TABLE key_rotations ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT 'rotated';

COMMENT ON COLUMN key_rotations.reason IS 'Why the key was revoked: rotated, compromised, lost, etc.';
