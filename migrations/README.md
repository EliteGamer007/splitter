# Database Migrations

This directory contains SQL migration files for the Splitter application database schema.

## Current Migrations

- **001_initial_schema.sql**: Creates all tables for users, posts, federation, messaging, and moderation

## Running Migrations

### Initial Setup

Run the migration after creating the database:

```bash
psql -U postgres -d splitter -f migrations/001_initial_schema.sql
```

### Using Migration Tools (Optional)

For production environments, consider using migration management tools:

**golang-migrate:**
```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./migrations \
  -database "postgres://user:password@localhost:5432/splitter?sslmode=disable" \
  up

# Rollback
migrate -path ./migrations \
  -database "postgres://user:password@localhost:5432/splitter?sslmode=disable" \
  down 1
```

**Other tools:**
- [goose](https://github.com/pressly/goose)
- [Atlas](https://atlasgo.io/)
- [dbmate](https://github.com/amacneil/dbmate)

## Schema Overview

The database includes tables for:

- **Identity & Users**: Local users, federated remote actors, user keys
- **Content**: Posts, media attachments, interactions (likes/reposts), bookmarks
- **Social**: Follow relationships with approval status
- **Messaging**: End-to-end encrypted direct messages and threads
- **Federation**: ActivityPub inbox/outbox activities and deduplication
- **Moderation**: Blocked domains, content reports, admin actions, reputation tracking

## Migration Naming Convention

Migrations follow the pattern: `{version}_{description}.sql`

Examples:
- `001_initial_schema.sql`
- `002_add_notifications.sql`
- `003_add_search_indexes.sql`

## Verifying Migration Success

After running migrations, verify everything is set up correctly:

```bash
psql "your-connection-string" -f migrations/verify_migration.sql
```

Expected output:
- ✓ 15+ tables created
- ✓ All critical columns exist (email, password_hash, role, etc.)
- ✓ 20+ indexes for performance
- ✓ Triggers for timestamp updates

## Important Notes

- ✅ Always backup your database before running migrations
- ✅ Test migrations in development environment first
- ✅ For new databases, use `000_master_schema.sql` (single file, no conflicts)
- ✅ For existing databases, use `004_consolidated_fixes.sql` (safe to run multiple times)
- ❌ Never modify existing migration files after they've been applied
- ❌ Skip deprecated files (002, 003) - they have conflicts

## 📖 Need Help?

See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for detailed instructions, troubleshooting, and connection examples.
