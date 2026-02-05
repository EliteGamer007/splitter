# Splitter Cloud Database Setup Guide

> Complete guide for deploying Splitter with Neon PostgreSQL cloud database

## 📋 Overview

This guide covers:
- ✅ Database schema migration to Neon (PostgreSQL cloud)
- ✅ Environment configuration
- ✅ Test user creation
- ✅ Running the application
- ✅ Troubleshooting common issues

---

## 🚀 Quick Start (5 Minutes)

### Prerequisites
- Neon account ([neon.tech](https://neon.tech))
- Docker (for running migrations)
- Go 1.21+ (for backend)
- Node.js 18+ (for frontend)

### Step 1: Get Your Neon Connection String

1. Create a new project on [Neon](https://console.neon.tech)
2. Copy your connection string:
   ```
   postgresql://user:password@host.region.neon.tech/database?sslmode=require
   ```

### Step 2: Run Database Migration

```bash
# From the project root directory
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/000_master_schema.sql
```

**Expected Output:**
```
CREATE EXTENSION
CREATE TABLE
CREATE TABLE
...
(19 tables created)
```

### Step 3: Verify Migration

```bash
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/verify_migration.sql
```

**Expected:** All checks show ✓ PASS

### Step 4: Configure Environment

Edit `.env` file in the `splitter/` directory:

```env
# Neon Database Configuration
DB_HOST=ep-xxx-xxx.region.neon.tech
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=your_database

# Application Configuration
PORT=8000
ENV=production
JWT_SECRET=your-super-secret-jwt-key-change-this
BASE_URL=http://localhost:8000
```

**Extract these from your Neon connection string:**
```
postgresql://[DB_USER]:[DB_PASSWORD]@[DB_HOST]:5432/[DB_NAME]?sslmode=require
```

### Step 5: Start the Backend

```bash
cd splitter
go run ./cmd/server
```

**Expected Output:**
```
2026/02/06 00:00:00 Connecting to database: ep-xxx.neon.tech:5432/neondb...
2026/02/06 00:00:00 Database connection established successfully
2026/02/06 00:00:00 Admin user created (username: admin, password: splitteradmin)
2026/02/06 00:00:00 Starting server on port 8000
⇨ http server started on [::]:8000
```

### Step 6: Start the Frontend

```bash
cd Splitter-frontend
npm install  # First time only
npm run dev
```

**Access:** http://localhost:3000

---

## 👤 Default Admin Account

After first run, an admin account is auto-created:

```
Username: admin
Password: splitteradmin
Email: admin@localhost
Role: Admin
```

**⚠️ IMPORTANT:** Change this password in production!

---

## 🧪 Creating Test Users

### Option A: Via API (Recommended)

Use the registration endpoint:

```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@test.com",
    "password": "password123",
    "display_name": "Alice Johnson"
  }'
```

### Option B: Via SQL Script

Create users with proper password hashes through the app itself (Option A), as manual hash creation can be error-prone.

---

## 📊 Database Schema

### Core Tables (19 Total)

**User Management:**
- `users` - User accounts (email/password + DID auth)
- `user_keys` - Multi-device public keys
- `remote_actors` - Federated remote users

**Social Features:**
- `follows` - Follow relationships
- `posts` - User content/posts
- `interactions` - Likes and reposts
- `bookmarks` - Saved posts
- `media` - Post attachments

**Messaging:**
- `message_threads` - DM conversations
- `messages` - Individual messages (E2EE support)

**Moderation:**
- `moderation_requests` - User requests for moderator role
- `admin_actions` - Audit log
- `reports` - Content reports
- `blocked_domains` - Defederated servers

**Federation (Future):**
- `inbox_activities` - Incoming ActivityPub
- `outbox_activities` - Outgoing ActivityPub
- `activity_deduplication` - Duplicate prevention
- `instance_reputation` - Server reputation
- `federation_failures` - Circuit breaker

---

## 🔧 Configuration Details

### Database Connection

The app uses SSL by default for Neon:
```go
// internal/db/postgres.go
connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require", ...)
```

### Auto-Migration Disabled

Migrations are manual only. The app **will not** run migrations on startup:
```go
// cmd/server/main.go
// Migrations are already applied manually to Neon
// Skip automatic migrations to avoid "relation already exists" errors
```

This prevents conflicts and ensures clean deployments.

---

## ✅ Testing Your Setup

### 1. Health Check
```bash
curl http://localhost:8000/api/v1/health
# Expected: {"status":"ok"}
```

### 2. Admin Login
```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"splitteradmin"}'
# Expected: {"token":"...","user":{...}}
```

### 3. User Registration
```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "testpass123",
    "display_name": "Test User"
  }'
# Expected: {"user":{...},"token":"..."}
```

### 4. Database Query
```bash
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -c "SELECT username, role FROM users;"
# Expected: List of users including admin
```

---

## 🐛 Troubleshooting

### "relation already exists" Error

**Cause:** Trying to run master schema on already-migrated database  
**Fix:** Database is already set up. Just start the app.

### "connection is insecure" Error

**Cause:** Missing `?sslmode=require` in connection string  
**Fix:** Add `?sslmode=require` to your connection string

### "Invalid username or password"

**Cause:** Admin user may have corrupted password hash  
**Fix:**
```bash
# Delete and recreate admin
docker run --rm postgres:15 psql 'YOUR_CONNECTION_STRING' \
  -c "DELETE FROM users WHERE username='admin';"
# Restart the app - it will recreate admin
go run ./cmd/server
```

### Port Already in Use

**Cause:** Previous server process still running  
**Fix (Windows):**
```powershell
# Find process
netstat -ano | findstr :8000
# Kill it (replace PID with actual process ID)
Stop-Process -Id PID -Force
```

**Fix (Linux/Mac):**
```bash
lsof -ti:8000 | xargs kill -9
```

### Frontend Can't Connect to Backend

**Check:**
1. Backend is running: `curl http://localhost:8000/api/v1/health`
2. Frontend API URL is correct: `lib/api.ts` should have `http://localhost:8000/api/v1`
3. CORS is enabled (it is by default in the backend)

---

## 📁 Migration Files Reference

| File | Purpose | When to Use |
|------|---------|-------------|
| `000_master_schema.sql` | ⭐ Complete schema | **Use for fresh Neon database** |
| `004_consolidated_fixes.sql` | Upgrade existing DB | Use if you have data to preserve |
| `verify_migration.sql` | Check migration success | After running 000 or 004 |
| `001-003_*.sql` | ❌ Deprecated | Do not use (conflicts) |

**IMPORTANT:** Files `001`, `002`, and `003` have conflicts and duplicate column definitions. Always use `000_master_schema.sql` for new databases.

---

## 🔐 Security Best Practices

### Production Deployment

1. **Change Admin Password:**
   ```bash
   # Login as admin, then update via frontend profile page
   # Or via API:
   curl -X PUT http://localhost:8000/api/v1/users/me \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"password":"new_secure_password"}'
   ```

2. **Update JWT Secret:**
   ```env
   JWT_SECRET=use-a-long-random-string-here-at-least-32-chars
   ```
   Generate: `openssl rand -base64 32`

3. **Use Environment Variables:**
   Never commit `.env` file. Use secrets manager in production.

4. **Enable HTTPS:**
   Use a reverse proxy (nginx, Caddy) with SSL certificates.

5. **Database Backups:**
   Neon provides automatic backups. Configure retention policy in Neon dashboard.

---

## 📈 Scaling Considerations

### Neon Features

- **Auto-scaling:** Neon automatically scales compute
- **Branching:** Create database branches for testing
- **Connection Pooling:** Built-in connection pooler
- **Read Replicas:** Available on paid plans

### Backend Optimization

- Connection pool is configured in `internal/config/config.go`:
  ```go
  MaxConns: 25,  // Adjust based on load
  MinConns: 5,
  ```

---

## 🎯 Next Steps

1. ✅ Verify setup with health check
2. ✅ Login as admin
3. ✅ Create test users via registration
4. ✅ Test social features (posts, follows, likes)
5. ✅ Setup moderation (request moderator role as test user)
6. ✅ Configure email (optional, for notifications)
7. ✅ Deploy to production (Heroku, Railway, Fly.io, etc.)

---

## 📞 Support

- **Issues:** Check logs in terminal where you ran `go run ./cmd/server`
- **Database:** Query directly via psql or Neon dashboard
- **Migrations:** See `migrations/verify_migration.sql` for validation

---

## 📝 Quick Commands Cheat Sheet

```bash
# Check database tables
docker run --rm postgres:15 psql 'CONNECTION_STRING' -c "\dt"

# Count users
docker run --rm postgres:15 psql 'CONNECTION_STRING' \
  -c "SELECT COUNT(*) FROM users;"

# List all users
docker run --rm postgres:15 psql 'CONNECTION_STRING' \
  -c "SELECT username, email, role FROM users;"

# Health check
curl http://localhost:8000/api/v1/health

# Test admin login
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"splitteradmin"}'

# Start backend
cd splitter && go run ./cmd/server

# Start frontend
cd Splitter-frontend && npm run dev
```

---

**Last Updated:** February 6, 2026  
**Database:** Neon PostgreSQL  
**Backend:** Go + Echo Framework  
**Frontend:** Next.js + React  

---

## ✨ You're All Set!

Your Splitter instance is now running on Neon cloud database. Enjoy building your federated social network! 🚀
