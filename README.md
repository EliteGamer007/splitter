# Splitter

> **Federated social media platform** built with Go, ActivityPub, and Next.js. Run your own instance and communicate across the network.

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js)](https://nextjs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql)](https://postgresql.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow)](LICENSE)

**Live Demo:** https://splitter-red-phi.vercel.app  
**Backend API:** https://splitter-m0kv.onrender.com/api/v1/health

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Federation](#federation)
- [Configuration](#configuration)
- [Testing](#testing)
- [Documentation Index](#documentation-index)
- [Contributing](#contributing)

---

## Overview

Splitter is a full-stack federated social network. Each instance is independent but communicates with others via the **ActivityPub** protocol — users on one instance can follow and exchange content with users on any other Splitter (or compatible) instance.

**Core Features:**
- **Federated social graph** — follow, posts, boosts, replies across instances
- **End-to-end encrypted DMs** — Ed25519 key-pairs per device; server never sees plaintext
- **Dual authentication** — password/JWT and DID (Decentralized Identity) cryptographic login
- **Hashtag trending** — regex extraction + real-time trending tab backed by PostgreSQL full-text search
- **AI `@split` bot** — mention `@split` in any post to receive a synchronous AI reply (Gemini 1.5 Flash / GPT-4o-mini)
- **Automated bot population** — GitHub Actions cron job seeds the network with realistic content every 30 minutes
- **Circle visibility** — posts visible only to curated "circle" members
- **Admin & moderation** — role management, content reports, AI-assisted screening

---

## Architecture

```
┌─────────────────────┐     HTTPS      ┌─────────────────────┐
│  Next.js Frontend   │ ◄────────────► │  Go/Echo Backend    │
│  (Vercel)           │                │  (Render)           │
│  React + Tailwind   │                │  REST API + AP      │
└─────────────────────┘                └──────────┬──────────┘
                                                   │ pgx/v5
                                       ┌───────────▼──────────┐
                                       │  PostgreSQL 15        │
                                       │  (Neon — serverless)  │
                                       └───────────────────────┘

Federation:
  Instance 1  ◄──── ActivityPub (HTTP Signatures) ────►  Instance 2
  splitter-m0kv.onrender.com                             splitter-2.onrender.com
```

| Component | Technology |
|-----------|-----------|
| Backend API | Go 1.24 / Echo v4 |
| Database | PostgreSQL 15 (Neon serverless) |
| ORM / Driver | jackc/pgx v5 |
| Frontend | Next.js 16 / React 19 / Tailwind CSS |
| Auth | bcrypt + JWT (RS256) / Ed25519 DID keypairs |
| Federation | ActivityPub (W3C), HTTP Signatures, WebFinger |
| AI | Google Gemini 1.5 Flash / OpenAI GPT-4o-mini |
| Bots | Python 3 + GitHub Actions cron |
| Hosting | Render (backend) + Vercel (frontend) + Neon (DB) |

---

## Project Structure

```
splitter/
├── cmd/
│   ├── server/             # Main server entrypoint
│   └── migrate/            # Standalone migration runner
├── internal/
│   ├── auth/               # DID keypair generation & verification
│   ├── config/             # Environment-based configuration
│   ├── db/                 # Database connection (Neon + SSL)
│   ├── federation/         # ActivityPub delivery, signatures, peer health
│   ├── handlers/           # HTTP request handlers (one file per domain)
│   ├── helpers/            # Shared utility functions
│   ├── middleware/         # JWT auth, optional auth, CORS
│   ├── models/             # Domain model structs
│   ├── repository/         # Data-access layer (pgx queries)
│   └── server/             # Router setup & middleware wiring
├── migrations/             # Ordered SQL migration files
├── scripts/
│   └── bots/               # Python bot population scripts
├── tests/
│   ├── docs/               # Testing documentation (01–08)
│   ├── unit/               # Unit tests (auth, posts, replies, users, security)
│   ├── integration/        # Integration test suite
│   ├── e2e_test/           # End-to-end tests
│   ├── load/               # Load tests (k6 / Go)
│   ├── seeder/             # Test data seeder
│   └── results/            # Captured test output
├── diagrams/               # Architecture, ER, sequence, class diagrams (Mermaid)
├── .env.example            # Environment variables template
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.instances.yml  # Two-instance local federation setup
├── Makefile                # Common dev commands
└── go.mod
```

---

## Quick Start

### Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.21+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 18+ | [nodejs.org](https://nodejs.org) |
| Docker | Any | For running `psql` migrations |
| Neon account | — | Free tier at [neon.tech](https://neon.tech) |

### 1. Clone & Configure

```bash
git clone <repository-url>
cd splitter
cp .env.example .env
```

Edit `.env` with your Neon credentials:

```env
DB_HOST=ep-your-endpoint.region.aws.neon.tech
DB_PORT=5432
DB_USER=neondb_owner
DB_PASSWORD=your-password
DB_NAME=neondb
PORT=8000
ENV=development
BASE_URL=http://localhost:8000
JWT_SECRET=change-this-to-a-secure-random-string
```

### 2. Apply Database Migrations

```bash
docker run --rm postgres:15 psql \
  'postgres://USER:PASS@HOST/DBNAME?sslmode=require' \
  -f migrations/000_master_schema.sql
```

### 3. Start the Backend

```bash
go mod download
go run ./cmd/server
# → Server listening on http://localhost:8000
```

Default admin account created on first startup:

```
Username: admin
Password: splitteradmin   ⚠️ Change immediately in production
```

### 4. Start the Frontend

```bash
cd ../Splitter-frontend
npm install
npm run dev
# → Frontend at http://localhost:3000
```

### 5. Two-Instance Local Federation (Optional)

```bash
docker compose -f docker-compose.instances.yml up
```

This spins up two backend instances that federate with each other locally.

---

## API Reference

Base URL: `http://localhost:8000/api/v1`  
`🔒` = Requires `Authorization: Bearer <jwt_token>`

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/register` | Register with username/email/password |
| `POST` | `/auth/login` | Login → returns JWT |
| `POST` | `/auth/challenge` | Request a DID auth nonce |
| `POST` | `/auth/verify` | Verify DID signature → returns JWT |
| `POST` | `/auth/refresh` 🔒 | Rotate JWT |
| `POST` | `/auth/logout` 🔒 | Invalidate token |

### Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/users/:id` | Get user by ID |
| `GET` | `/users/did` | Get user by DID |
| `GET` 🔒 | `/users/me` | Get own profile |
| `PUT` 🔒 | `/users/me` | Update profile |
| `DELETE` 🔒 | `/users/me` | Delete account |
| `GET` 🔒 | `/users/search` | Search users |
| `POST` 🔒 | `/users/me/circle/:id` | Add user to circle |
| `DELETE` 🔒 | `/users/me/circle/:id` | Remove from circle |
| `GET` 🔒 | `/users/me/circle/:id/check` | Check circle membership |

### Posts

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/posts/public` | Public feed |
| `GET` | `/posts/:id` | Get post |
| `GET` | `/posts/user/:did` | Posts by user DID |
| `GET` 🔒 | `/posts/feed` | Personalized feed (follows + own) |
| `POST` 🔒 | `/posts` | Create post (multipart/form-data) |
| `PUT` 🔒 | `/posts/:id` | Update post |
| `DELETE` 🔒 | `/posts/:id` | Delete post |

### Replies

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/posts/:id/replies` | Threaded replies for a post |
| `POST` 🔒 | `/posts/:id/replies` | Reply to a post |

### Social

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` 🔒 | `/users/:id/follow` | Follow user |
| `DELETE` 🔒 | `/users/:id/follow` | Unfollow |
| `GET` | `/users/:id/followers` | Followers list |
| `GET` | `/users/:id/following` | Following list |
| `POST` 🔒 | `/posts/:id/like` | Like post |
| `DELETE` 🔒 | `/posts/:id/like` | Unlike post |
| `POST` 🔒 | `/posts/:id/repost` | Boost/repost |
| `POST` 🔒 | `/posts/:id/bookmark` | Bookmark post |

### Direct Messages

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` 🔒 | `/messages/threads` | All message threads |
| `GET` 🔒 | `/messages/conversation/:userId` | Messages with a user |
| `POST` 🔒 | `/messages/send` | Send encrypted message |

### Federation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/federation/timeline` | Cross-instance federated timeline |
| `GET` | `/federation/users` | Search users across instances |
| `POST` | `/ap/users/:username/inbox` | ActivityPub inbox |
| `GET` | `/ap/users/:username` | ActivityPub Actor JSON-LD |
| `GET` | `/.well-known/webfinger` | WebFinger discovery |

### Admin `🔒 Admin role required`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/users` | List all users |
| `PUT` | `/admin/users/:id/role` | Change role |
| `POST` | `/admin/users/:id/suspend` | Suspend account |
| `POST` | `/admin/users/:id/unsuspend` | Unsuspend account |
| `GET` | `/admin/moderation-requests` | Content reports queue |
| `POST` | `/admin/moderation-requests/:id/approve` | Approve moderator |

Full annotated reference with request/response examples: [`API_ENDPOINTS.md`](API_ENDPOINTS.md)

---

## Authentication

### Password Login

Standard username + password flow returning a JWT:

```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret"}'
# → {"token":"eyJ..."}
```

Use the token as `Authorization: Bearer <token>` on subsequent requests.

### DID (Decentralized Identity)

Challenge-response using Ed25519 keypairs — the server never receives the private key:

```
1. POST /auth/challenge   { did: "did:web:..." }   → { challenge: "nonce" }
2. Client signs nonce with private key
3. POST /auth/verify      { did, challenge, signature } → { token: "JWT" }
```

Private keys are stored locally (encrypted) in the browser. Recovery requires the encrypted backup file downloaded at registration.

---

## Federation

Splitter implements a subset of [ActivityPub](https://www.w3.org/TR/activitypub/):

### Supported Activities
`Create` · `Follow` · `Accept` · `Like` · `Announce` · `Undo` · `Delete`

### Security
All inbound federation POSTs require:
- `Signature` header — HTTP Signatures (RSA-SHA256)
- `Digest` header — SHA-256 body digest
- `Date` header — within ±30 seconds (keep clocks synced via NTP)

### Environmental Requirements
```env
FEDERATION_ENABLED=true
FEDERATION_DOMAIN=your-instance-id
FEDERATION_URL=https://your-public-url
```

For full well-known endpoints and ActivityPub routes, see [`API_ENDPOINTS.md`](API_ENDPOINTS.md#federation--well-known-endpoints).

---

## Configuration

All configuration is via environment variables. See `.env.example` for the full list.

| Variable | Required | Description |
|----------|----------|-------------|
| `DB_HOST` | ✅ | PostgreSQL host (Neon endpoint) |
| `DB_PORT` | ✅ | PostgreSQL port (default: 5432) |
| `DB_USER` | ✅ | Database username |
| `DB_PASSWORD` | ✅ | Database password |
| `DB_NAME` | ✅ | Database name |
| `PORT` | ✅ | HTTP server port |
| `ENV` | ✅ | `development` or `production` |
| `JWT_SECRET` | ✅ | Secret for JWT signing |
| `BASE_URL` | ✅ | Public base URL of this instance |
| `SPLIT_BOT_API_KEY` | Optional | OpenAI / Gemini key for `@split` bot |
| `FEDERATION_ENABLED` | Optional | Enable ActivityPub federation |
| `FEDERATION_DOMAIN` | Optional | Canonical domain/ID for this instance |
| `FEDERATION_URL` | Optional | Public HTTPS URL for federation |

---

## Testing

```bash
# Run all tests
make test

# Run with coverage report
make test-cover

# Test a specific package
go test ./tests/unit/auth -v

# Integration tests (requires live DB in .env)
go test ./tests/integration/... -v

# Load tests
go test ./tests/load -v
```

Test documentation is in [`tests/docs/`](tests/docs/):

| Doc | Coverage |
|-----|---------|
| [`01_UNIT_TESTING.md`](tests/docs/01_UNIT_TESTING.md) | Unit test strategy & examples |
| [`02_INTEGRATION_TESTING.md`](tests/docs/02_INTEGRATION_TESTING.md) | Integration flows |
| [`03_E2E_TESTING.md`](tests/docs/03_E2E_TESTING.md) | End-to-end scenarios |
| [`04_DATABASE_TESTING.md`](tests/docs/04_DATABASE_TESTING.md) | Schema & migration tests |
| [`05_LOAD_TESTING.md`](tests/docs/05_LOAD_TESTING.md) | k6 / Go load benchmarks |
| [`06_TEST_SEEDING.md`](tests/docs/06_TEST_SEEDING.md) | Data seeder usage |
| [`07_TEST_REPORTS.md`](tests/docs/07_TEST_REPORTS.md) | Report format & CI integration |
| [`08_REGRESSION_TESTING.md`](tests/docs/08_REGRESSION_TESTING.md) | Regression suite |

---

## Documentation Index

| Document | Description |
|----------|-------------|
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | Live services, Render + Neon setup, CI/CD, deploy checklist |
| [`API_ENDPOINTS.md`](API_ENDPOINTS.md) | Full annotated API reference with cURL examples |
| [`DATABASE_SCHEMA.md`](DATABASE_SCHEMA.md) | Schema tables, indexes, relationships |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Dev workflow, code standards, extension recipes |
| [`SECURITY.md`](SECURITY.md) | Security model, DID auth, E2EE, threat model |
| [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md) | Common issues — DB connection, federation, DID |
| [`CODING_STANDARDS.md`](CODING_STANDARDS.md) | Go style guide & naming conventions |
| [`OPS.md`](OPS.md) | Operational runbooks |
| [`DESIGN.md`](DESIGN.md) | Design decisions & ADRs |
| [`ROADMAP.md`](ROADMAP.md) | Planned features & milestones |
| [`BOTS.md`](BOTS.md) | Bot system architecture |
| [`GLOSSARY.md`](GLOSSARY.md) | Domain terminology |
| [`diagrams/`](diagrams/README.md) | Architecture, ER, sequence, state diagrams (Mermaid) |
| [`migrations/README.md`](migrations/README.md) | Migration file inventory |
| [`tests/docs/`](tests/docs/) | Full testing documentation suite |

---

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for:
- Development setup
- Branch & commit conventions
- Code standards (Go formatting, error handling, naming)
- How to add new endpoints, models, and handlers
- Developer recipes (ActivityPub extensions, theming, bots)
- PR checklist

---

## License

MIT — see [`LICENSE`](LICENSE).



