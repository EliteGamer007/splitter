# Database Migrations

This directory contains SQL migration files for the Splitter application database schema.

## Current Migrations

| File | Purpose | When to Use |
|------|---------|-------------|
| `000_master_schema.sql` | Complete current schema | **Fresh database setup** |
| `001_initial_schema.sql` | Original base schema | Legacy baseline only |
| `002_upgrade_to_current.sql` | Safe consolidated upgrade | Upgrade existing database |
| `verify_migration.sql` | Verification checks | After any migration |

## Running Migrations

### Fresh Database (Neon Cloud)

```bash
docker run --rm postgres:15 psql \
  'postgresql://user:password@host.neon.tech/dbname?sslmode=require' \
  -f migrations/000_master_schema.sql
```

### Verify Migration

```bash
docker run --rm postgres:15 psql \
  'postgresql://user:password@host.neon.tech/dbname?sslmode=require' \
  -f migrations/verify_migration.sql
```

### Upgrade Existing Database

```bash
docker run --rm postgres:15 psql \
  'YOUR_CONNECTION_STRING' \
  -f migrations/002_upgrade_to_current.sql
```

## Schema Overview

The database includes tables for:

- **Identity & Users**: Local users, federated remote actors, user keys
- **Content**: Posts, media attachments, interactions (likes/reposts), bookmarks
- **Social**: Follow relationships with approval status
- **Messaging**: End-to-end encrypted direct messages and threads
- **Federation**: ActivityPub inbox/outbox activities and deduplication
- **Moderation**: Blocked domains, content reports, admin actions, reputation tracking

## ⚠️ Required: Migration 018 (Messaging Schema)

Migration `018_fix_messages_schema.sql` adds columns required by the messaging system (`sender_id`, `recipient_id`, `content`, `encrypted_keys`, `client_message_id`, `is_read`, `client_created_at`, `deleted_at`, `edited_at`). **All instances must apply this migration before running the server** to avoid `"column does not exist"` database errors during Direct Messaging.

```bash
docker run --rm postgres:15 psql \
  'YOUR_CONNECTION_STRING' \
  -f migrations/018_fix_messages_schema.sql
```

The migration is idempotent and safe to run multiple times.

## Important Notes

- ✅ Always backup your database before running migrations
- ✅ For new databases, use `000_master_schema.sql` (single file, complete schema)
- ✅ For existing databases, use `002_upgrade_to_current.sql` (safe with IF NOT EXISTS)
- ✅ Apply `018_fix_messages_schema.sql` on **every instance** for DM support
- ✅ Neon requires `sslmode=require` in connection strings
- ❌ Auto-migrations are disabled in the app (manual only)

## Need Help?

See [NEON_SETUP_GUIDE.md](../NEON_SETUP_GUIDE.md) for complete setup instructions.
