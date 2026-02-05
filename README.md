# Splitter - Federated Social Media Platform

A federated social media application with **password-based** and **DID (Decentralized Identity)** authentication, built with Go, Echo framework, and PostgreSQL (Neon Cloud).

## Overview

Splitter supports two authentication methods:
- **Password Login** — Standard username/email + password (primary method)
- **DID Authentication** — Ed25519 cryptographic keypairs for advanced/federated users

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.21+ / Echo v4 |
| Database | PostgreSQL 15 (Neon Cloud) |
| Frontend | Next.js / React |
| Auth | bcrypt + JWT / Ed25519 DID |
| ORM | pgx/v5 |

## Project Structure

```
splitter/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── config/         # Configuration management
│   ├── db/             # Database connection (Neon + SSL)
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/      # Authentication middleware
│   ├── models/         # Data models
│   ├── repository/     # Data access layer
│   └── server/         # Router setup
├── migrations/         # Database migration scripts
├── .env.example        # Environment variables template
└── NEON_SETUP_GUIDE.md # Cloud database setup guide
```

Frontend lives in a separate directory: `Splitter-frontend/`

## Prerequisites

- **Go**: 1.21 or higher — [Download Go](https://go.dev/dl/)
- **Node.js**: 18+ — [Download Node.js](https://nodejs.org/)
- **Docker**: For running migrations via psql — [Download Docker](https://www.docker.com/)
- **Neon Account**: Free cloud PostgreSQL — [Sign up](https://neon.tech)

## Quick Start

### 1. Set Up Neon Database

1. Create a project at [console.neon.tech](https://console.neon.tech)
2. Copy your connection string

### 2. Run Database Migration

```bash
docker run --rm postgres:15 psql \
  'postgresql://user:password@host.neon.tech/dbname?sslmode=require' \
  -f migrations/000_master_schema.sql
```

### 3. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your Neon credentials:
```env
# Database (Neon Cloud)
DB_HOST=ep-your-endpoint.region.aws.neon.tech
DB_PORT=5432
DB_USER=your_neon_username
DB_PASSWORD=your_neon_password
DB_NAME=neondb

# Application
PORT=8000
ENV=development
BASE_URL=http://localhost:8000
JWT_SECRET=your-secret-key-change-this
```

### 4. Install Dependencies & Run

**Backend (Terminal 1):**
```bash
cd splitter
go mod download
go run ./cmd/server
```
Server starts on `http://localhost:8000`

**Frontend (Terminal 2):**
```bash
cd Splitter-frontend
npm install
npm run dev
```
Frontend starts on `http://localhost:3000`

### 5. Default Admin Account

On first startup, an admin account is automatically created:
```
Username: admin
Password: splitteradmin
```

> **Change this password in production!**

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register with username/email/password |
| POST | `/api/v1/auth/login` | Login with username + password |
| POST | `/api/v1/auth/challenge` | Get DID auth challenge |
| POST | `/api/v1/auth/verify` | Verify DID signed challenge |

### Users (🔒 = Requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET 🔒 | `/api/v1/users/me` | Get current user profile |
| PUT 🔒 | `/api/v1/users/me` | Update profile |
| DELETE 🔒 | `/api/v1/users/me` | Delete account |
| GET | `/api/v1/users/:id` | Get user by ID |
| GET | `/api/v1/users/did` | Get user by DID |
| GET 🔒 | `/api/v1/users/search` | Search users |

### Posts (🔒 = Requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST 🔒 | `/api/v1/posts` | Create post (multipart/form-data) |
| GET | `/api/v1/posts/:id` | Get post |
| PUT 🔒 | `/api/v1/posts/:id` | Update post |
| DELETE 🔒 | `/api/v1/posts/:id` | Delete post |
| GET 🔒 | `/api/v1/posts/feed` | Get personalized feed |
| GET | `/api/v1/posts/public` | Get public feed |

### Social (🔒 = Requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST 🔒 | `/api/v1/users/:id/follow` | Follow user |
| DELETE 🔒 | `/api/v1/users/:id/follow` | Unfollow user |
| GET | `/api/v1/users/:id/followers` | Get followers |
| GET | `/api/v1/users/:id/following` | Get following |
| POST 🔒 | `/api/v1/posts/:id/like` | Like post |
| POST 🔒 | `/api/v1/posts/:id/repost` | Repost |
| POST 🔒 | `/api/v1/posts/:id/bookmark` | Bookmark post |

### Messaging (🔒 = Requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET 🔒 | `/api/v1/messages/threads` | Get message threads |
| POST 🔒 | `/api/v1/messages/send` | Send message |
| POST 🔒 | `/api/v1/messages/conversation/:userId` | Start conversation |

### Admin (🔒 = Admin role required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET 🔒 | `/api/v1/admin/users` | List all users |
| PUT 🔒 | `/api/v1/admin/users/:id/role` | Change user role |
| POST 🔒 | `/api/v1/admin/users/:id/suspend` | Suspend user |
| POST 🔒 | `/api/v1/admin/users/:id/unsuspend` | Unsuspend user |
| GET 🔒 | `/api/v1/admin/moderation-requests` | List moderation requests |
| POST 🔒 | `/api/v1/admin/moderation-requests/:id/approve` | Approve moderator request |

## API Examples

### Register User
```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "password123",
    "display_name": "Alice"
  }'
```

### Login
```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "splitteradmin"}'
```

### Get Current User (Protected)
```bash
curl http://localhost:8000/api/v1/users/me \
  -H "Authorization: Bearer <your-jwt-token>"
```

### Health Check
```bash
curl http://localhost:8000/api/v1/health
# Returns: {"status":"ok"}
```

## Database

### Cloud Database (Neon)

Splitter uses **Neon PostgreSQL** cloud database with SSL.

- **Schema:** 19 tables (users, posts, follows, messages, moderation, federation)
- **Migration:** `migrations/000_master_schema.sql` (complete schema)
- **SSL:** Required (`sslmode=require`)
- **Auto-migrations:** Disabled (manual only)

See [NEON_SETUP_GUIDE.md](NEON_SETUP_GUIDE.md) for detailed setup.

### Test Accounts

| Username | Password | Role |
|----------|----------|------|
| admin | splitteradmin | Admin |
| alice | password123 | User |
| bob | password123 | User |
| carol | password123 | User |
| dave | password123 | User |
| eve | password123 | User |

## Security Features

- ✅ **bcrypt password hashing** — Secure password storage
- ✅ **JWT authentication** — Stateless token-based auth
- ✅ **Role-based access control** — Admin, Moderator, User roles
- ✅ **Ed25519 DID auth** — Optional cryptographic authentication
- ✅ **SSL database connections** — Encrypted data in transit
- ✅ **Challenge-response auth** — Prevents replay attacks (DID mode)

## Development

### Run Tests
```bash
go test ./...
```

### Build for Production
```bash
go build -o bin/server ./cmd/server
./bin/server
```

### Verify Database
```bash
docker run --rm postgres:15 psql 'YOUR_CONNECTION_STRING' \
  -f migrations/verify_migration.sql
```

## Documentation

| Document | Description |
|----------|-------------|
| [NEON_SETUP_GUIDE.md](NEON_SETUP_GUIDE.md) | Complete cloud database setup |
| [API_QUICK_REFERENCE.md](API_QUICK_REFERENCE.md) | Full API reference with examples |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines |
| [SPRINT_1_STATUS.md](SPRINT_1_STATUS.md) | Sprint 1 completion details |
| [SPRINT_2_STATUS.md](SPRINT_2_STATUS.md) | Sprint 2 progress tracking |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

MIT License — see [LICENSE](LICENSE) file for details.

---

**Project Status:** Active Development (Sprint 2)  
**Backend:** http://localhost:8000  
**Frontend:** http://localhost:3000  
**Database:** Neon Cloud PostgreSQL (SSL)



