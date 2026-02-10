# ⚡ Splitter Cloud Database Setup (Neon)

This guide walks you through setting up the remote PostgreSQL database on [Neon.tech](https://neon.tech) for the Splitter backend.

## 📋 Prerequisites
- **Neon Account**: Sign up at [neon.tech](https://neon.tech)
- **Docker**: For running database migrations
- **Go 1.21+**: To run the backend server

---

## 🚀 1. Database Setup

1. **Create Project**: Log in to Neon Console and create a new project (e.g., `splitter-db`).
2. **Get Connection String**:
   - On the Dashboard, look for "Connection Details".
   - Copy the connection string. It looks like:
     ```
     postgres://user:password@ep-xyz.region.neon.tech/neondb?sslmode=require
     ```

## 🛠️ 2. Run Migrations

We use Docker to apply the schema to your new cloud database.

Run this command from the `splitter/` root directory:

```bash
# Replace 'YOUR_NEON_CONNECTION_STRING' with the actual string you copied
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/000_master_schema.sql
```

**Verify Success:**
You should see output ending with multiple `CREATE TABLE` and `CREATE INDEX` messages.

To verify, run the verification script:
```bash
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/verify_migration.sql
```
*Look for "✓ PASS" at the end.*

---

## ⚙️ 3. Environment Config

1. Copy `.env.example` to `.env` in the `splitter/` folder.
2. Fill in the `DB_*` values using parts of your connection string:

   `postgres://[DB_USER]:[DB_PASSWORD]@[DB_HOST]/[DB_NAME]?sslmode=require`

   **Example `.env`:**
   ```env
   # Database (Neon)
   DB_HOST=ep-falling-leaf-123456.us-east-2.aws.neon.tech
   DB_PORT=5432
   DB_USER=neondb_owner
   DB_PASSWORD=npg_YourPasswordReference
   DB_NAME=neondb

   # App Settings
   PORT=8000
   ENV=production
   JWT_SECRET=change-this-to-a-secure-random-string
   BASE_URL=http://localhost:8000
   ```

---

## ▶️ 4. Start Server

```bash
cd splitter
go run ./cmd/server
```

**Success logs:**
```
INFO: Connecting to database...
INFO: Database connection established successfully
INFO: Starting server on port 8000
```
