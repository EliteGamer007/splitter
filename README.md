# Splitter - Federated Social Media Platform

A federated social media application with Decentralized Identity (DID) authentication built with Go, Echo framework, and PostgreSQL.

## Overview

Splitter uses **passwordless authentication** with Ed25519 cryptographic signatures. Users authenticate using DIDs (Decentralized Identifiers) and cryptographic keypairs instead of traditional passwords.

## Project Structure

```
splitter/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── config/         # Configuration management
│   ├── db/             # Database connection
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # Authentication middleware
│   ├── models/         # Data models
│   ├── repository/     # Data access layer
│   └── server/         # Router setup
├── migrations/         # Database migrations
├── .env.example        # Environment variables template
└── FRONTEND_TASKS.md  # Frontend implementation guide
```

## Prerequisites

- **Go**: 1.21 or higher - [Download Go](https://go.dev/dl/)
- **PostgreSQL**: 14 or higher - [Download PostgreSQL](https://www.postgresql.org/download/)
- **Node.js**: 18+ (for frontend)

## Getting Started

### 1. Start PostgreSQL

**Windows:**
```powershell
Get-Service postgresql* | Start-Service
```

**Linux/Mac:**
```bash
sudo systemctl start postgresql
```

### 2. Create Database

```bash
psql -U postgres
CREATE DATABASE splitter;
\q
```

### 3. Run Migrations

```bash
psql -U postgres -d splitter -f migrations/001_initial_schema.sql
```

### 4. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your database credentials:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=splitter

PORT=3000
ENV=development

JWT_SECRET=your-secret-key-change-this
```

### 5. Install Dependencies

```bash
go mod download
```

### 6. Run Server

```bash
go run cmd/server/main.go
```

Server starts on `http://localhost:3000`

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register with DID |
| POST | `/api/v1/auth/challenge` | Get auth challenge |
| POST | `/api/v1/auth/verify` | Verify signed challenge |

### Users (🔒 = Requires JWT token)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET 🔒 | `/api/v1/users/me` | Get current user |
| PUT 🔒 | `/api/v1/users/me` | Update profile |
| DELETE 🔒 | `/api/v1/users/me` | Delete account |
| GET | `/api/v1/users/:id` | Get user by ID |

### Posts (🔒 = Requires JWT token)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST 🔒 | `/api/v1/posts` | Create post |
| GET | `/api/v1/posts/:id` | Get post |
| PUT 🔒 | `/api/v1/posts/:id` | Update post (owner only) |
| DELETE 🔒 | `/api/v1/posts/:id` | Delete post (owner only) |
| GET 🔒 | `/api/v1/posts/feed` | Get personalized feed |

### Follows (🔒 = Requires JWT token)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST 🔒 | `/api/v1/users/:id/follow` | Follow user |
| DELETE 🔒 | `/api/v1/users/:id/follow` | Unfollow user |

## Authentication Flow

### Registration
1. Generate Ed25519 keypair (client-side)
2. Create DID from public key
3. POST to `/auth/register` with DID + public key
4. Receive JWT token

### Login
1. POST to `/auth/challenge` with DID
2. Receive random nonce
3. Sign nonce with private key (client-side)
4. POST to `/auth/verify` with signature
5. Receive JWT token

### Protected Requests
Include JWT in Authorization header:
```
Authorization: Bearer <jwt_token>
```

## API Examples

### Register User
```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "instance_domain": "localhost:3000",
    "did": "did:key:z6Mkf5rGM...",
    "display_name": "John Doe",
    "public_key": "base64EncodedPublicKey==",
    "bio": "Hello world"
  }'
```

### Get Challenge
```bash
curl -X POST http://localhost:3000/api/v1/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"did": "did:key:z6Mkf5rGM..."}'
```

### Verify Challenge
```bash
curl -X POST http://localhost:3000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "did": "did:key:z6Mkf5rGM...",
    "challenge": "randomNonce==",
    "signature": "base64Signature=="
  }'
```

### Get Current User (Protected)
```bash
curl http://localhost:3000/api/v1/users/me \
  -H "Authorization: Bearer <your-jwt-token>"
```

## Security Features

- ✅ **No passwords stored** - Uses cryptographic keypairs
- ✅ **Challenge-response auth** - Prevents replay attacks
- ✅ **Ed25519 signatures** - Fast, secure elliptic curve crypto
- ✅ **JWT tokens** - Stateless authentication
- ✅ **5-minute challenge expiry** - Time-limited nonces
- ✅ **Private keys never transmitted** - Signing happens client-side

## Frontend Implementation

**See [FRONTEND_TASKS.md](FRONTEND_TASKS.md) for complete implementation guide.**

The frontend needs to implement:
- ✅ Ed25519 keypair generation
- ✅ DID creation from public key
- ✅ Secure private key storage (IndexedDB)
- ✅ Challenge signing
- ✅ Registration & login UI
- ✅ Profile management
- ✅ Post creation & feed
- ✅ Follow system
- ✅ Error handling

See detailed code examples, testing checklist, and step-by-step guide in [FRONTEND_TASKS.md](FRONTEND_TASKS.md).

## Development

### Run Tests
```bash
go test ./...
```

### Build for Production
```bash
make build
./bin/server
```

### Format Code
```bash
go fmt ./...
```

## Resources

- [Frontend Implementation Guide](FRONTEND_TASKS.md) - Complete guide with code examples
- [W3C DID Core](https://www.w3.org/TR/did-core/) - DID specification
- [Ed25519 Signatures](https://ed25519.cr.yp.to/) - Cryptography details
- [Echo Framework](https://echo.labstack.com/) - Go web framework docs

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Backend Status:** ✅ Production-ready and tested  
**Frontend Status:** ⏳ See [FRONTEND_TASKS.md](FRONTEND_TASKS.md) for implementation tasks


