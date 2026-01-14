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

## Important Notes

- Always backup your database before running migrations
- Test migrations in development environment first
- Migrations are run in numerical order
- Never modify existing migration files after they've been applied
