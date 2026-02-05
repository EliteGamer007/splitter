# API Quick Reference

Base URL: `http://localhost:8000/api/v1`

## Authentication

### Register New User
```http
POST /auth/register
Content-Type: application/json

{
  "username": "alice",
  "instance_domain": "federate.tech",
  "did": "did:key:z6Mk...",
  "display_name": "Alice Chen",
  "public_key": "base64_encoded_public_key",
  "bio": "Decentralization enthusiast",
  "avatar_url": "https://example.com/avatar.jpg"
}
```

### Get Login Challenge
```http
POST /auth/challenge
Content-Type: application/json

{
  "did": "did:key:z6Mk..."
}

Response:
{
  "challenge": "random_nonce_string",
  "expires_at": 1234567890
}
```

### Verify Challenge & Get JWT
```http
POST /auth/verify
Content-Type: application/json

{
  "did": "did:key:z6Mk...",
  "challenge": "random_nonce_string",
  "signature": "base64_encoded_signature"
}

Response:
{
  "token": "jwt_token_here",
  "user": { ... }
}
```

---

## Users

### Get Current User (Protected)
```http
GET /users/me
Authorization: Bearer <jwt_token>
```

### Get User by ID
```http
GET /users/{id}
# id can be UUID or use /users/did?did=<did> for DID lookup
```

### Update Profile (Protected)
```http
PUT /users/me
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "display_name": "New Name",
  "bio": "New bio",
  "avatar_url": "new_url"
}
```

### Delete Account (Protected)
```http
DELETE /users/me
Authorization: Bearer <jwt_token>
```

---

## Posts

### Create Post (Protected)
```http
POST /posts
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "content": "Hello federated world!",
  "image_url": "optional_image_url"
}
```

### Get Post
```http
GET /posts/{id}
```

### Get User's Posts
```http
GET /posts/user/{user_id}?limit=20&offset=0
```

### Get Feed (Protected)
```http
GET /posts/feed?limit=20&offset=0
Authorization: Bearer <jwt_token>
```

### Update Post (Protected)
```http
PUT /posts/{id}
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "content": "Updated content"
}
```

### Delete Post (Protected)
```http
DELETE /posts/{id}
Authorization: Bearer <jwt_token>
```

---

## Follow System ✨ NEW

### Follow User (Protected)
```http
POST /users/{id}/follow
Authorization: Bearer <jwt_token>

Response:
{
  "follower_did": "did:key:...",
  "following_did": "did:key:...",
  "status": "accepted",
  "created_at": "2026-01-28T..."
}
```

### Unfollow User (Protected)
```http
DELETE /users/{id}/follow
Authorization: Bearer <jwt_token>
```

### Get Followers
```http
GET /users/{id}/followers?limit=50&offset=0

Response: Array of User objects
```

### Get Following
```http
GET /users/{id}/following?limit=50&offset=0

Response: Array of User objects
```

### Get Follow Stats
```http
GET /users/{id}/stats

Response:
{
  "followers": 1250,
  "following": 340
}
```

---

## Post Interactions ✨ NEW

### Like Post (Protected)
```http
POST /posts/{id}/like
Authorization: Bearer <jwt_token>
```

### Unlike Post (Protected)
```http
DELETE /posts/{id}/like
Authorization: Bearer <jwt_token>
```

### Repost/Boost Post (Protected)
```http
POST /posts/{id}/repost
Authorization: Bearer <jwt_token>
```

### Remove Repost (Protected)
```http
DELETE /posts/{id}/repost
Authorization: Bearer <jwt_token>
```

### Bookmark Post (Protected)
```http
POST /posts/{id}/bookmark
Authorization: Bearer <jwt_token>
```

### Remove Bookmark (Protected)
```http
DELETE /posts/{id}/bookmark
Authorization: Bearer <jwt_token>
```

### Get Bookmarks (Protected)
```http
GET /users/me/bookmarks
Authorization: Bearer <jwt_token>

Response: Array of Post objects with author info
```

---

## Health Check

```http
GET /health

Response:
{
  "status": "ok"
}
```

---

## Error Responses

All endpoints return standard error format:

```json
{
  "error": "Error message here"
}
```

### Common Status Codes
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing/invalid JWT
- `404 Not Found` - Resource not found
- `409 Conflict` - Duplicate resource (e.g., already following)
- `500 Internal Server Error` - Server error

---

## Frontend Integration Example

### TypeScript API Service

```typescript
// lib/api.ts
const API_BASE = 'http://localhost:8000/api/v1';

const getAuthHeaders = () => {
  const token = localStorage.getItem('jwt_token');
  return {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` })
  };
};

export const api = {
  // Auth
  async register(data: UserCreate) {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    return res.json();
  },

  async getChallenge(did: string) {
    const res = await fetch(`${API_BASE}/auth/challenge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ did })
    });
    return res.json();
  },

  async verifyChallenge(data: VerifyRequest) {
    const res = await fetch(`${API_BASE}/auth/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    const json = await res.json();
    if (json.token) {
      localStorage.setItem('jwt_token', json.token);
    }
    return json;
  },

  // Users
  async getCurrentUser() {
    const res = await fetch(`${API_BASE}/users/me`, {
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async getUserProfile(id: string) {
    const res = await fetch(`${API_BASE}/users/${id}`);
    return res.json();
  },

  async updateProfile(data: UserUpdate) {
    const res = await fetch(`${API_BASE}/users/me`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(data)
    });
    return res.json();
  },

  // Posts
  async createPost(content: string, imageUrl?: string) {
    const res = await fetch(`${API_BASE}/posts`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ content, image_url: imageUrl })
    });
    return res.json();
  },

  async getFeed(limit = 20, offset = 0) {
    const res = await fetch(
      `${API_BASE}/posts/feed?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return res.json();
  },

  async getPost(id: string) {
    const res = await fetch(`${API_BASE}/posts/${id}`);
    return res.json();
  },

  // Follow
  async followUser(userId: string) {
    const res = await fetch(`${API_BASE}/users/${userId}/follow`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async unfollowUser(userId: string) {
    const res = await fetch(`${API_BASE}/users/${userId}/follow`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async getFollowers(userId: string, limit = 50, offset = 0) {
    const res = await fetch(
      `${API_BASE}/users/${userId}/followers?limit=${limit}&offset=${offset}`
    );
    return res.json();
  },

  async getFollowStats(userId: string) {
    const res = await fetch(`${API_BASE}/users/${userId}/stats`);
    return res.json();
  },

  // Interactions
  async likePost(postId: string) {
    const res = await fetch(`${API_BASE}/posts/${postId}/like`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async unlikePost(postId: string) {
    const res = await fetch(`${API_BASE}/posts/${postId}/like`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async repostPost(postId: string) {
    const res = await fetch(`${API_BASE}/posts/${postId}/repost`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async bookmarkPost(postId: string) {
    const res = await fetch(`${API_BASE}/posts/${postId}/bookmark`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return res.json();
  },

  async getBookmarks() {
    const res = await fetch(`${API_BASE}/users/me/bookmarks`, {
      headers: getAuthHeaders()
    });
    return res.json();
  }
};
```

### React Hook Example

```typescript
// hooks/useAuth.ts
import { useState, useEffect } from 'react';
import { api } from '@/lib/api';

export function useAuth() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem('jwt_token');
    if (token) {
      api.getCurrentUser()
        .then(setUser)
        .catch(() => localStorage.removeItem('jwt_token'))
        .finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, []);

  const login = async (did: string, signature: string, challenge: string) => {
    const data = await api.verifyChallenge({ did, signature, challenge });
    setUser(data.user);
    return data;
  };

  const logout = () => {
    localStorage.removeItem('jwt_token');
    setUser(null);
  };

  return { user, loading, login, logout };
}
```

### Usage in Component

```tsx
// components/pages/HomePage.jsx
import { useAuth } from '@/hooks/useAuth';
import { api } from '@/lib/api';

export default function HomePage() {
  const { user } = useAuth();
  const [posts, setPosts] = useState([]);
  const [newPostText, setNewPostText] = useState('');

  useEffect(() => {
    loadFeed();
  }, []);

  const loadFeed = async () => {
    const data = await api.getFeed();
    setPosts(data);
  };

  const handleCreatePost = async () => {
    if (newPostText.trim()) {
      await api.createPost(newPostText);
      setNewPostText('');
      loadFeed(); // Reload feed
    }
  };

  const handleLike = async (postId) => {
    await api.likePost(postId);
    loadFeed(); // Refresh to see updated counts
  };

  // ... rest of component
}
```

---

## Environment Setup

### Backend (.env)
```
DB_HOST=ep-your-endpoint.region.aws.neon.tech
DB_PORT=5432
DB_USER=your_neon_username
DB_PASSWORD=your_neon_password
DB_NAME=neondb
PORT=8000
ENV=development
BASE_URL=http://localhost:8000
JWT_SECRET=your_secret_key_here
```

### Frontend (.env.local)
```
NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
```

---

## Testing with cURL

### Register
```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "display_name": "Test User"
  }'
```

### Login
```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "splitteradmin"}'
```

### Create Post (Protected)
```bash
curl -X POST http://localhost:8000/api/v1/posts \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "content=Hello world!"
```

### Get Feed
```bash
curl http://localhost:8000/api/v1/posts/feed \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Follow User
```bash
curl -X POST http://localhost:8000/api/v1/users/USER_ID/follow \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Like Post
```bash
curl -X POST http://localhost:8000/api/v1/posts/POST_ID/like \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## Database Quick Reference

### Tables in Use
- `users` - User accounts with DID
- `posts` - User posts/content
- `follows` - Follow relationships
- `interactions` - Likes and reposts
- `bookmarks` - Private saved posts

### Tables Ready But Not Used Yet
- `remote_actors` - Federated users
- `user_keys` - Multi-device support
- `media` - Post attachments
- `message_threads` - DM threads
- `messages` - E2EE messages
- `inbox_activities` - Federation inbox
- `outbox_activities` - Federation outbox
- `reports` - Content reports
- `blocked_domains` - Defederation
- `admin_actions` - Audit log
- `instance_reputation` - Server reputation
- `federation_failures` - Circuit breaker

---

## Migration Commands

```bash
# Run master schema (for fresh database)
docker run --rm postgres:15 psql 'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/000_master_schema.sql

# Verify migration
docker run --rm postgres:15 psql 'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/verify_migration.sql

# Check table structure
docker run --rm postgres:15 psql 'YOUR_NEON_CONNECTION_STRING' \
  -c "\d users"
```

---

## Start Servers

### Backend
```bash
cd splitter
go run ./cmd/server
# Server starts on http://localhost:8000
```

### Frontend
```bash
cd Splitter-frontend
npm install
npm run dev
# Frontend starts on http://localhost:3000
```

---

## Need Help?

- **API Issues**: Check `internal/server/router.go` for route definitions
- **Database Issues**: See `migrations/000_master_schema.sql`
- **Authentication**: Review `internal/handlers/auth_handler.go`
- **Setup Guide**: See `NEON_SETUP_GUIDE.md`
