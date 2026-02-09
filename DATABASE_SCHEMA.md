# Database Schema Documentation

## Overview

Splitter uses **PostgreSQL 15** (hosted on Neon Cloud) as its primary database. The schema supports a federated social media platform with dual authentication modes (password-based and DID-based), messaging, content management, and moderation features.

**Key Features:**
- 19 tables organized into 5 functional domains
- UUID-based primary keys for distributed system compatibility
- Support for both local and federated users
- End-to-end encrypted messaging capability
- Comprehensive moderation and governance system
- Activity deduplication for federation

**Database Extensions:**
- `uuid-ossp` - UUID generation support

---

## Entity-Relationship Overview

The database is organized into five main domains:

1. **Users & Identity** - Local users, remote actors, and cryptographic keys
2. **Social Relationships** - Follow relationships between users
3. **Content & Posts** - Posts, media attachments, interactions, and bookmarks
4. **Messaging** - Direct message threads and encrypted messages
5. **Federation & Governance** - ActivityPub federation, moderation, and admin tools

**Core Relationships:**
- Users create Posts (1:N)
- Users follow other Users (M:N via `follows`)
- Posts have Media attachments (1:N)
- Users interact with Posts via likes/reposts (M:N via `interactions`)
- Users exchange Messages in Threads (M:N)
- Posts can be Reported for moderation (1:N via `reports`)

---

## Tables

### Users & Identity

#### `users`
Local user accounts supporting both password-based and DID authentication.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique user identifier |
| `username` | TEXT | NOT NULL | Username (unique per instance) |
| `email` | TEXT | UNIQUE | Email for password authentication |
| `password_hash` | TEXT | - | Bcrypt hashed password |
| `instance_domain` | TEXT | NOT NULL | Home instance domain |
| `did` | TEXT | UNIQUE | Decentralized Identifier (optional) |
| `display_name` | TEXT | - | User's display name |
| `bio` | TEXT | - | User biography |
| `avatar_url` | TEXT | - | Profile picture URL |
| `public_key` | TEXT | - | Ed25519 public key (for DID auth) |
| `role` | TEXT | DEFAULT 'user', CHECK | User role: `user`, `moderator`, `admin` |
| `moderation_requested` | BOOLEAN | DEFAULT FALSE | Whether user requested moderator role |
| `moderation_requested_at` | TIMESTAMPTZ | - | Timestamp of moderation request |
| `is_locked` | BOOLEAN | DEFAULT FALSE | Account locked status |
| `is_suspended` | BOOLEAN | DEFAULT FALSE | Account suspension status |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Account creation timestamp |
| `updated_at` | TIMESTAMPTZ | DEFAULT now() | Last update timestamp |

**Indexes:**
- `idx_users_did` on `did`
- `idx_users_username` on `username`
- `idx_users_email` on `email` (WHERE email IS NOT NULL)
- `idx_users_role` on `role`

---

#### `remote_actors`
Federated users from other instances.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique identifier |
| `actor_uri` | TEXT | UNIQUE, NOT NULL | ActivityPub actor URI |
| `username` | TEXT | NOT NULL | Remote username |
| `instance_domain` | TEXT | NOT NULL | Remote instance domain |
| `did` | TEXT | - | Remote user's DID (if available) |
| `public_key` | TEXT | NOT NULL | Public key for signature verification |
| `inbox_url` | TEXT | NOT NULL | ActivityPub inbox endpoint |
| `outbox_url` | TEXT | NOT NULL | ActivityPub outbox endpoint |
| `last_fetched_at` | TIMESTAMPTZ | - | Last profile fetch timestamp |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | First seen timestamp |

---

#### `user_keys`
Multiple public keys per user for device management.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique key identifier |
| `user_id` | UUID | FOREIGN KEY → users(id) ON DELETE CASCADE | Owner user |
| `public_key` | TEXT | NOT NULL | Public key material |
| `key_type` | TEXT | NOT NULL | Key algorithm (e.g., Ed25519) |
| `is_revoked` | BOOLEAN | DEFAULT FALSE | Key revocation status |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Key creation timestamp |

---

### Social Relationships

#### `follows`
Follow relationships between users (local and federated).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique relationship identifier |
| `follower_did` | TEXT | NOT NULL | DID of the follower |
| `following_did` | TEXT | NOT NULL | DID of the user being followed |
| `status` | TEXT | DEFAULT 'accepted', CHECK | Status: `pending`, `accepted`, `rejected` |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Follow request timestamp |

**Indexes:**
- `idx_follows_follower_status` on `(follower_did, status)`
- `idx_follows_following_status` on `(following_did, status)`

---

### Content & Posts

#### `posts`
User-generated content (local and federated).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique post identifier |
| `author_did` | TEXT | NOT NULL | Author's DID |
| `content` | TEXT | - | Post text content |
| `visibility` | TEXT | DEFAULT 'public', CHECK | Visibility: `public`, `followers`, `circle` |
| `is_remote` | BOOLEAN | DEFAULT FALSE | Whether post is from remote instance |
| `original_post_uri` | TEXT | - | Original ActivityPub URI (for remote posts) |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Post creation timestamp |
| `updated_at` | TIMESTAMPTZ | - | Last edit timestamp |
| `deleted_at` | TIMESTAMPTZ | - | Soft delete timestamp |
| `expires_at` | TIMESTAMPTZ | - | Expiration timestamp (ephemeral posts) |

**Indexes:**
- `idx_posts_author` on `author_did`
- `idx_posts_created` on `created_at DESC`
- `idx_posts_visibility` on `visibility`

---

#### `media`
Media attachments for posts (images, videos, etc.).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique media identifier |
| `post_id` | UUID | FOREIGN KEY → posts(id) ON DELETE CASCADE | Parent post |
| `media_url` | TEXT | NOT NULL | Media file URL |
| `media_type` | TEXT | NOT NULL | MIME type (e.g., image/jpeg) |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Upload timestamp |

---

#### `interactions`
User interactions with posts (likes, reposts).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique interaction identifier |
| `post_id` | UUID | FOREIGN KEY → posts(id) ON DELETE CASCADE | Target post |
| `actor_did` | TEXT | NOT NULL | DID of the user interacting |
| `interaction_type` | TEXT | CHECK | Type: `like`, `repost` |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Interaction timestamp |

**Constraints:**
- UNIQUE(`post_id`, `actor_did`, `interaction_type`) - Prevents duplicate interactions

---

#### `bookmarks`
Private saved posts per user.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique bookmark identifier |
| `user_id` | UUID | FOREIGN KEY → users(id) ON DELETE CASCADE | User who bookmarked |
| `post_id` | UUID | FOREIGN KEY → posts(id) ON DELETE CASCADE | Bookmarked post |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Bookmark timestamp |

**Constraints:**
- UNIQUE(`user_id`, `post_id`) - Prevents duplicate bookmarks

---

### Messaging

#### `message_threads`
Direct message conversation threads between two users.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique thread identifier |
| `participant_a` | TEXT | - | DID of first participant |
| `participant_b` | TEXT | - | DID of second participant |
| `participant_a_id` | UUID | FOREIGN KEY → users(id) | First participant (local users) |
| `participant_b_id` | UUID | FOREIGN KEY → users(id) | Second participant (local users) |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Thread creation timestamp |
| `updated_at` | TIMESTAMPTZ | DEFAULT now() | Last message timestamp |

**Indexes:**
- `idx_message_threads_participants` on `(participant_a_id, participant_b_id)`

---

#### `messages`
Individual messages within threads (supports E2EE).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique message identifier |
| `thread_id` | UUID | FOREIGN KEY → message_threads(id) ON DELETE CASCADE | Parent thread |
| `sender_did` | TEXT | - | Sender's DID |
| `recipient_did` | TEXT | - | Recipient's DID |
| `sender_id` | UUID | FOREIGN KEY → users(id) | Sender (local users) |
| `recipient_id` | UUID | FOREIGN KEY → users(id) | Recipient (local users) |
| `ciphertext` | BYTEA | - | Encrypted message content |
| `content` | TEXT | - | Plaintext content (for unencrypted messages) |
| `is_read` | BOOLEAN | DEFAULT FALSE | Read status |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Message timestamp |
| `delivered_at` | TIMESTAMPTZ | - | Delivery confirmation timestamp |

**Indexes:**
- `idx_messages_thread` on `thread_id`
- `idx_messages_sender` on `sender_id`
- `idx_messages_recipient` on `recipient_id`

---

### Federation Engine

#### `inbox_activities`
Incoming ActivityPub activities from federated instances.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique record identifier |
| `activity_id` | TEXT | UNIQUE, NOT NULL | ActivityPub activity ID |
| `actor_uri` | TEXT | NOT NULL | Actor URI who sent the activity |
| `activity_type` | TEXT | NOT NULL | Activity type (Follow, Like, Create, etc.) |
| `payload` | JSONB | NOT NULL | Full ActivityPub JSON payload |
| `received_at` | TIMESTAMPTZ | DEFAULT now() | Receipt timestamp |

**Indexes:**
- `idx_inbox_actor` on `actor_uri`

---

#### `outbox_activities`
Outgoing ActivityPub activities to federated instances.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique record identifier |
| `activity_type` | TEXT | NOT NULL | Activity type |
| `payload` | JSONB | NOT NULL | ActivityPub JSON payload |
| `target_inbox` | TEXT | NOT NULL | Destination inbox URL |
| `status` | TEXT | CHECK | Delivery status: `pending`, `sent`, `failed` |
| `retry_count` | INT | DEFAULT 0 | Number of delivery attempts |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Creation timestamp |

**Indexes:**
- `idx_outbox_status` on `status`

---

#### `activity_deduplication`
Cache to prevent processing duplicate activities.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `activity_id` | TEXT | PRIMARY KEY | ActivityPub activity ID |
| `processed_at` | TIMESTAMPTZ | DEFAULT now() | Processing timestamp |
| `expires_at` | TIMESTAMPTZ | - | Cache expiration timestamp |

---

### Governance & Moderation

#### `moderation_requests`
User requests to become moderators.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique request identifier |
| `user_id` | UUID | FOREIGN KEY → users(id) ON DELETE CASCADE | Requesting user |
| `status` | TEXT | DEFAULT 'pending', CHECK | Status: `pending`, `approved`, `rejected` |
| `reason` | TEXT | - | User's reason for requesting |
| `reviewed_by` | UUID | FOREIGN KEY → users(id) | Admin who reviewed |
| `reviewed_at` | TIMESTAMPTZ | - | Review timestamp |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Request timestamp |

**Indexes:**
- `idx_moderation_requests_status` on `status`

---

#### `blocked_domains`
Blocked remote instances (defederation).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `domain` | TEXT | PRIMARY KEY | Blocked domain name |
| `reason` | TEXT | - | Reason for blocking |
| `blocked_by` | TEXT | - | Admin who blocked |
| `blocked_at` | TIMESTAMPTZ | DEFAULT now() | Block timestamp |

---

#### `reports`
Content moderation reports.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique report identifier |
| `reporter_did` | TEXT | NOT NULL | DID of reporting user |
| `post_id` | UUID | FOREIGN KEY → posts(id) | Reported post |
| `reason` | TEXT | - | Report reason |
| `status` | TEXT | CHECK | Status: `pending`, `resolved` |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Report timestamp |
| `resolved_at` | TIMESTAMPTZ | - | Resolution timestamp |

---

#### `admin_actions`
Audit log of all administrative actions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique action identifier |
| `admin_id` | TEXT | NOT NULL | DID of admin performing action |
| `action_type` | TEXT | NOT NULL | Action type (suspend, delete, etc.) |
| `target` | TEXT | - | Target of action (user ID, post ID, etc.) |
| `reason` | TEXT | - | Reason for action |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Action timestamp |

**Indexes:**
- `idx_admin_actions_admin_id` on `admin_id`
- `idx_admin_actions_created_at` on `created_at DESC`

---

#### `instance_reputation`
Reputation scoring for federated instances.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `domain` | TEXT | PRIMARY KEY | Instance domain |
| `reputation_score` | INT | DEFAULT 0 | Calculated reputation score |
| `spam_count` | INT | DEFAULT 0 | Number of spam incidents |
| `failure_count` | INT | DEFAULT 0 | Federation failure count |
| `updated_at` | TIMESTAMPTZ | DEFAULT now() | Last update timestamp |

---

#### `federation_failures`
Circuit breaker for failing federated instances.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `domain` | TEXT | PRIMARY KEY | Instance domain |
| `failure_count` | INT | DEFAULT 0 | Consecutive failure count |
| `last_failure_at` | TIMESTAMPTZ | - | Last failure timestamp |
| `circuit_open_until` | TIMESTAMPTZ | - | Circuit breaker timeout |

---

## Relationships (Foreign Keys)

```
users (1) ──< (N) user_keys
users (1) ──< (N) bookmarks
users (1) ──< (N) moderation_requests (as requester)
users (1) ──< (N) moderation_requests (as reviewer)
users (1) ──< (N) message_threads (as participant_a)
users (1) ──< (N) message_threads (as participant_b)
users (1) ──< (N) messages (as sender)
users (1) ──< (N) messages (as recipient)

posts (1) ──< (N) media
posts (1) ──< (N) interactions
posts (1) ──< (N) bookmarks
posts (1) ──< (N) reports

message_threads (1) ──< (N) messages
```

---

## Indexing Strategy

### Performance Indexes
- **User lookups**: `did`, `username`, `email`, `role`
- **Post queries**: `author_did`, `created_at DESC`, `visibility`
- **Social graphs**: Composite indexes on `follows` for follower/following queries
- **Messaging**: Thread and participant indexes for fast message retrieval
- **Federation**: Actor URI and status indexes for activity processing
- **Moderation**: Status and timestamp indexes for admin dashboards

### Partial Indexes
- `idx_users_email` - Only indexes non-NULL emails (password users)

---

## Sample SQL Schema

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Example: Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT UNIQUE,
    password_hash TEXT,
    instance_domain TEXT NOT NULL,
    did TEXT UNIQUE,
    display_name TEXT,
    bio TEXT,
    avatar_url TEXT,
    public_key TEXT,
    role TEXT DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin')),
    moderation_requested BOOLEAN DEFAULT FALSE,
    moderation_requested_at TIMESTAMPTZ,
    is_locked BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create indexes
CREATE INDEX idx_users_did ON users(did);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_role ON users(role);

-- Create trigger for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**Full Schema**: See `migrations/000_master_schema.sql` for the complete schema definition.

---

## Notes / Best Practices

### Migration Management
- **Master Schema**: Use `000_master_schema.sql` for fresh database setup
- **Incremental Migrations**: Individual migration files in `migrations/` directory
- **Verification**: Run `verify_migration.sql` after applying migrations
- **No Auto-Migration**: Migrations are manual-only for production safety

### Connection Settings
- **SSL Required**: Always use `sslmode=require` for Neon connections
- **Connection Pooling**: Recommended for production deployments
- **Timeout Settings**: Configure appropriate statement timeouts

### Data Integrity
- **Soft Deletes**: Posts use `deleted_at` instead of hard deletion
- **Cascading Deletes**: User deletion cascades to related data (bookmarks, keys, etc.)
- **Unique Constraints**: Prevent duplicate interactions, bookmarks, and follows
- **Check Constraints**: Enforce valid enum values for roles, statuses, visibility

### Performance Optimization
- **Use Indexes**: All foreign keys and frequently queried columns are indexed
- **JSONB for Flexibility**: ActivityPub payloads stored as JSONB for efficient querying
- **Timestamp Indexes**: DESC indexes on `created_at` for feed queries
- **Composite Indexes**: Multi-column indexes for complex queries (follows, messages)

### Security Considerations
- **Password Hashing**: Always use bcrypt (handled in application layer)
- **DID Validation**: Validate DID format before insertion
- **Input Sanitization**: Prevent SQL injection (use parameterized queries)
- **Audit Logging**: All admin actions logged in `admin_actions`

### Backup Strategy
- **Regular Backups**: Neon provides automated backups
- **Point-in-Time Recovery**: Available on Neon paid plans
- **Export Critical Data**: Regularly export user and content tables

### Monitoring
- **Query Performance**: Monitor slow queries on posts and follows tables
- **Index Usage**: Verify indexes are being used with EXPLAIN ANALYZE
- **Table Sizes**: Monitor growth of `posts`, `messages`, and `inbox_activities`
- **Connection Count**: Track active database connections
