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
├── Frontend/           # Next.js Frontend application
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

PORT=8000
ENV=development

JWT_SECRET=your-secret-key-change-this
```

### 5. Install Dependencies

**Backend:**
```bash
go mod download
```

**Frontend:**
```bash
cd Frontend
npm install
cd ..
```

### 6. Run Application

**Terminal 1 (Backend):**
```bash
go run cmd/server/main.go
```
Server starts on `http://localhost:8000`

**Terminal 2 (Frontend):**
```bash
cd Frontend
npm run dev
```
Frontend starts on `http://localhost:3000`

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

## Development Status

**Current Sprint:** Sprint 2 (Target: 65% completion)  
**Overall Progress:** 52% complete (26/50 user stories)

### ✅ Completed Features (Sprint 1 & 2)

**Identity & Onboarding (100% complete):**
- ✅ Landing page with federation explanation
- ✅ Instance discovery and selection
- ✅ DID-based decentralized registration
- ✅ Ed25519 cryptographic authentication
- ✅ Security and recovery options
- ✅ Multi-step onboarding flow

**Content & Social Features (53% complete):**
- ✅ Post creation with text and media support
- ✅ Visibility controls (public, followers, circle)
- ✅ Home timeline with follow filtering
- ✅ Post interactions (likes, reposts, bookmarks)
- ✅ User search functionality
- ✅ Follow/unfollow system
- ✅ Profile pages with real-time stats
- ✅ Post deletion

**Messaging (64% complete):**
- ✅ Direct messaging UI
- ✅ Conversation threads
- ✅ Unread message indicators
- ✅ Real-time message updates

**Admin & Moderation (45% complete):**
- ✅ Comprehensive admin dashboard
- ✅ User suspension/ban system
- ✅ Moderation request approval system
- ✅ Admin action audit logging
- ✅ Role-based access control

### 🟡 In Progress Features (Sprint 2)

**Federation Engine (11% complete):**
- 🟡 WebFinger discovery endpoint
- 🟡 ActivityPub inbox for receiving federated content
- ⏳ ActivityPub outbox for sending posts
- ⏳ HTTP signatures for secure federation

**Enhanced Moderation:**
- 🟡 Content reporting system
- 🟡 Instance blocking (defederation) UI
- 🟡 Enhanced audit logging

**Content Improvements:**
- 🟡 Reply threading and conversation trees
- 🟡 Media upload UI with file picker
- 🟡 Hashtag extraction and linking

**Messaging:**
- 🟡 End-to-end encryption integration

### 🎯 Planned Features (Sprint 2+)

**Federation & Distribution:**
- Remote user discovery and following
- Cross-instance post delivery
- Federated interactions (likes, reposts, replies)
- Activity deduplication
- Profile update propagation
- Federated content deletion

**Content & Media:**
- Image and video upload processing
- Media proxy for privacy
- Post editing with version history
- Advanced search with filters
- Trending topics and hashtags

**Moderation & Safety:**
- Content reporting queue and review
- Automated spam detection
- Circuit breaker for failing instances
- Appeal system for moderation actions
- Automated content filtering
- User blocking and muting

**Messaging & Privacy:**
- End-to-end encrypted DMs
- Message key exchange
- Encryption indicators
- Message deletion and editing
- Group messaging

**User Experience:**
- Timeline switching (home/local/federated)
- Notification grouping and filtering
- Dark/light theme customization
- Accessibility improvements
- Mobile-responsive design
- Progressive Web App (PWA)

**Advanced Features:**
- Content warnings and sensitive media
- Polls and surveys
- Custom emojis
- Multi-account support
- Import/export data
- Advanced privacy settings
- Circle/list management
- Scheduled posts

For detailed progress tracking, see:
- [Sprint 1 Status](SPRINT_1_STATUS.md) - Completed features
- [Sprint 2 Status](SPRINT_2_STATUS.md) - Current sprint progress

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Project Status:** 🟡 Active Development (Sprint 2)  
**Backend Status:** ✅ Core features production-ready  
**Frontend Status:** 🟡 52% complete with admin dashboard  
**Federation Status:** ⏳ In Progress (WebFinger + ActivityPub)

**See [SPRINT_2_STATUS.md](SPRINT_2_STATUS.md) for detailed progress tracking.**


