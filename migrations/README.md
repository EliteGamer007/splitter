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

## Important Notes

- ✅ Always backup your database before running migrations
- ✅ For new databases, use `000_master_schema.sql` (single file, complete schema)
- ✅ For existing databases, use `002_upgrade_to_current.sql` (safe with IF NOT EXISTS)
- ✅ Neon requires `sslmode=require` in connection strings
- ❌ Auto-migrations are disabled in the app (manual only)

## Need Help?

See [NEON_SETUP_GUIDE.md](../NEON_SETUP_GUIDE.md) for complete setup instructions.
