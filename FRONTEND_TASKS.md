# Frontend Implementation Tasks

This document outlines all frontend tasks needed to complete the Splitter application with DID authentication.

## 🎯 Implementation Priority

| Priority | Task | Estimated Effort |
|----------|------|------------------|
| 🔴 HIGH | Task 1: Cryptographic Key Management | 4-6 hours |
| 🔴 HIGH | Task 2: Registration Flow | 3-4 hours |
| 🔴 HIGH | Task 3: Login Flow | 3-4 hours |
| 🟡 MEDIUM | Task 4: Authenticated Requests | 2-3 hours |
| 🟡 MEDIUM | Task 5: User Profile Management | 3-4 hours |
| 🟡 MEDIUM | Task 6: Post Creation & Feed | 4-6 hours |
| 🟢 LOW | Task 7: Follow System | 2-3 hours |
| 🟡 MEDIUM | Task 8: Error Handling & UX | 2-3 hours |
| 🟢 LOW | Task 9: Key Backup & Recovery | 3-4 hours |

**Total Estimated Effort:** 26-37 hours

---

## 🎯 Task 1: Cryptographic Key Management

**Priority:** 🔴 HIGH  
**Dependencies:** None  
**Estimated Effort:** 4-6 hours

### Overview
Implement Ed25519 keypair generation, DID creation, secure key storage, and challenge signing. This is the foundation of the entire authentication system.

### Requirements

#### 1.1 Generate Ed25519 Keypair

**Library Recommendation:** `tweetnacl` or `tweetnacl-js`

```javascript
import nacl from 'tweetnacl';

function generateKeypair() {
  const keypair = nacl.sign.keyPair();
  return {
    publicKey: keypair.publicKey,   // Uint8Array(32)
    secretKey: keypair.secretKey    // Uint8Array(64)
  };
}
```

**Alternative: Web Crypto API**
```javascript
async function generateKeypairWebCrypto() {
  const keypair = await crypto.subtle.generateKey(
    { name: "Ed25519" },
    true,
    ["sign", "verify"]
  );
  return keypair;
}
```

#### 1.2 Create DID from Public Key

```javascript
function createDID(publicKey) {
  // Convert Uint8Array to base64
  const publicKeyBase64 = btoa(String.fromCharCode(...publicKey));
  
  // Create DID in did:key format
  const did = `did:key:${publicKeyBase64}`;
  
  return {
    did,
    publicKeyBase64
  };
}
```

#### 1.3 Store Private Key Securely

**Storage Strategy:** Use IndexedDB (never localStorage!)

```javascript
import { openDB } from 'idb';

async function initKeyStore() {
  return await openDB('splitter-keys', 1, {
    upgrade(db) {
      db.createObjectStore('keys');
    }
  });
}

async function storePrivateKey(secretKey) {
  const db = await initKeyStore();
  
  // Store as Uint8Array
  await db.put('keys', secretKey, 'privateKey');
  
  console.log('Private key stored securely');
}

async function getPrivateKey() {
  const db = await initKeyStore();
  const secretKey = await db.get('keys', 'privateKey');
  
  if (!secretKey) {
    throw new Error('Private key not found. Please register or import a key.');
  }
  
  return secretKey;
}

async function deletePrivateKey() {
  const db = await initKeyStore();
  await db.delete('keys', 'privateKey');
}
```

#### 1.4 Sign Challenges

```javascript
function signChallenge(challenge, secretKey) {
  // Convert challenge string to bytes
  const challengeBytes = new TextEncoder().encode(challenge);
  
  // Sign with Ed25519 (detached signature)
  const signature = nacl.sign.detached(challengeBytes, secretKey);
  
  // Convert to base64 for transmission
  const signatureBase64 = btoa(String.fromCharCode(...signature));
  
  return signatureBase64;
}
```

### Complete Key Management Module

```javascript
// crypto.js - Complete implementation

import nacl from 'tweetnacl';
import { openDB } from 'idb';

const DB_NAME = 'splitter-keys';
const DB_VERSION = 1;
const STORE_NAME = 'keys';

// Initialize IndexedDB
async function initDB() {
  return await openDB(DB_NAME, DB_VERSION, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    }
  });
}

// Generate new keypair
export function generateKeypair() {
  const keypair = nacl.sign.keyPair();
  return {
    publicKey: keypair.publicKey,
    secretKey: keypair.secretKey
  };
}

// Create DID from public key
export function createDID(publicKey) {
  const publicKeyBase64 = btoa(String.fromCharCode(...publicKey));
  return {
    did: `did:key:${publicKeyBase64}`,
    publicKeyBase64
  };
}

// Store private key
export async function storePrivateKey(secretKey) {
  const db = await initDB();
  await db.put(STORE_NAME, secretKey, 'privateKey');
}

// Retrieve private key
export async function getPrivateKey() {
  const db = await initDB();
  const secretKey = await db.get(STORE_NAME, 'privateKey');
  
  if (!secretKey) {
    throw new Error('No private key found');
  }
  
  return secretKey;
}

// Check if key exists
export async function hasPrivateKey() {
  try {
    await getPrivateKey();
    return true;
  } catch {
    return false;
  }
}

// Delete private key
export async function deletePrivateKey() {
  const db = await initDB();
  await db.delete(STORE_NAME, 'privateKey');
}

// Sign challenge
export function signChallenge(challenge, secretKey) {
  const challengeBytes = new TextEncoder().encode(challenge);
  const signature = nacl.sign.detached(challengeBytes, secretKey);
  return btoa(String.fromCharCode(...signature));
}

// Verify signature (client-side validation)
export function verifySignature(challenge, signature, publicKey) {
  const challengeBytes = new TextEncoder().encode(challenge);
  const signatureBytes = Uint8Array.from(atob(signature), c => c.charCodeAt(0));
  return nacl.sign.detached.verify(challengeBytes, signatureBytes, publicKey);
}
```

### Testing

```javascript
// test-crypto.js
import { 
  generateKeypair, 
  createDID, 
  storePrivateKey, 
  getPrivateKey,
  signChallenge,
  verifySignature
} from './crypto.js';

async function testCrypto() {
  console.log('Testing crypto module...');
  
  // 1. Generate keypair
  const keypair = generateKeypair();
  console.log('✅ Keypair generated');
  
  // 2. Create DID
  const { did, publicKeyBase64 } = createDID(keypair.publicKey);
  console.log('✅ DID created:', did);
  
  // 3. Store private key
  await storePrivateKey(keypair.secretKey);
  console.log('✅ Private key stored');
  
  // 4. Retrieve private key
  const retrievedKey = await getPrivateKey();
  console.log('✅ Private key retrieved');
  
  // 5. Sign challenge
  const challenge = 'test-challenge-12345';
  const signature = signChallenge(challenge, keypair.secretKey);
  console.log('✅ Challenge signed:', signature.substring(0, 20) + '...');
  
  // 6. Verify signature
  const isValid = verifySignature(challenge, signature, keypair.publicKey);
  console.log('✅ Signature verified:', isValid);
}

testCrypto();
```

### Dependencies to Install

```bash
npm install tweetnacl tweetnacl-util idb
```

---

## 🎯 Task 2: Registration Flow

**Priority:** 🔴 HIGH  
**Dependencies:** Task 1  
**Estimated Effort:** 3-4 hours

### UI Components

1. **Registration Form**
   - Username input (required, 3-50 chars)
   - Display name input (required)
   - Bio textarea (optional)
   - Avatar URL input (optional)
   - Submit button

2. **Key Generation Modal**
   - Show loading while generating keypair
   - Display generated DID
   - Warning about key backup
   - Download backup button
   - Continue button

### Implementation

```javascript
// RegisterPage.jsx (React example)
import React, { useState } from 'react';
import { generateKeypair, createDID, storePrivateKey } from './crypto';
import { registerUser } from './api';

export function RegisterPage() {
  const [formData, setFormData] = useState({
    username: '',
    displayName: '',
    bio: '',
    avatarUrl: ''
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [generatedDID, setGeneratedDID] = useState(null);

  const handleRegister = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      // 1. Generate keypair
      const keypair = generateKeypair();
      const { did, publicKeyBase64 } = createDID(keypair.publicKey);
      
      setGeneratedDID(did);

      // 2. Store private key
      await storePrivateKey(keypair.secretKey);

      // 3. Register with backend
      const response = await registerUser({
        username: formData.username,
        instance_domain: window.location.host,
        did: did,
        display_name: formData.displayName,
        public_key: publicKeyBase64,
        bio: formData.bio,
        avatar_url: formData.avatarUrl
      });

      // 4. Store JWT token
      localStorage.setItem('jwt_token', response.token);

      // 5. Redirect to home
      window.location.href = '/';

    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="register-page">
      <h1>Register</h1>
      
      {error && <div className="error">{error}</div>}
      
      {generatedDID && (
        <div className="did-display">
          <p><strong>Your DID:</strong></p>
          <code>{generatedDID}</code>
          <p className="warning">⚠️ Save this DID! You'll need it to log in.</p>
        </div>
      )}

      <form onSubmit={handleRegister}>
        <input
          type="text"
          placeholder="Username"
          value={formData.username}
          onChange={(e) => setFormData({...formData, username: e.target.value})}
          required
          minLength={3}
          maxLength={50}
        />
        
        <input
          type="text"
          placeholder="Display Name"
          value={formData.displayName}
          onChange={(e) => setFormData({...formData, displayName: e.target.value})}
          required
        />
        
        <textarea
          placeholder="Bio (optional)"
          value={formData.bio}
          onChange={(e) => setFormData({...formData, bio: e.target.value})}
        />
        
        <input
          type="url"
          placeholder="Avatar URL (optional)"
          value={formData.avatarUrl}
          onChange={(e) => setFormData({...formData, avatarUrl: e.target.value})}
        />
        
        <button type="submit" disabled={loading}>
          {loading ? 'Registering...' : 'Register'}
        </button>
      </form>
    </div>
  );
}
```

### API Helper

```javascript
// api.js
const API_BASE = 'http://localhost:3000/api/v1';

export async function registerUser(data) {
  const response = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Registration failed');
  }

  return await response.json();
}
```

---

## 🎯 Task 3: Login Flow

**Priority:** 🔴 HIGH  
**Dependencies:** Task 1  
**Estimated Effort:** 3-4 hours

### UI Components

1. **Login Form**
   - DID input
   - Submit button
   - Link to registration

2. **Challenge Signing Indicator**
   - Show "Signing challenge..."
   - Countdown timer (5 minutes)

### Implementation

```javascript
// LoginPage.jsx
import React, { useState } from 'react';
import { getPrivateKey, signChallenge } from './crypto';
import { getChallenge, verifyChallenge } from './api';

export function LoginPage() {
  const [did, setDid] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setStatus('');

    try {
      // 1. Get challenge from server
      setStatus('Requesting challenge...');
      const { challenge, expires_at } = await getChallenge(did);
      
      // 2. Load private key
      setStatus('Loading private key...');
      const secretKey = await getPrivateKey();
      
      // 3. Sign challenge
      setStatus('Signing challenge...');
      const signature = signChallenge(challenge, secretKey);
      
      // 4. Verify with server
      setStatus('Verifying signature...');
      const { token } = await verifyChallenge({ did, challenge, signature });
      
      // 5. Store token
      localStorage.setItem('jwt_token', token);
      
      // 6. Redirect
      setStatus('Login successful!');
      window.location.href = '/';

    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <h1>Login</h1>
      
      {error && <div className="error">{error}</div>}
      {status && <div className="status">{status}</div>}
      
      <form onSubmit={handleLogin}>
        <input
          type="text"
          placeholder="Enter your DID"
          value={did}
          onChange={(e) => setDid(e.target.value)}
          required
        />
        
        <button type="submit" disabled={loading}>
          {loading ? 'Logging in...' : 'Login'}
        </button>
      </form>
      
      <p>
        Don't have an account? <a href="/register">Register here</a>
      </p>
    </div>
  );
}
```

### API Helpers

```javascript
// api.js (continued)

export async function getChallenge(did) {
  const response = await fetch(`${API_BASE}/auth/challenge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ did })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to get challenge');
  }

  return await response.json();
}

export async function verifyChallenge(data) {
  const response = await fetch(`${API_BASE}/auth/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Authentication failed');
  }

  return await response.json();
}
```

---

## 🎯 Task 4: Authenticated Requests

**Priority:** 🟡 MEDIUM  
**Dependencies:** Task 2, Task 3  
**Estimated Effort:** 2-3 hours

### HTTP Client with Auth

```javascript
// api.js - Enhanced with auth

function getAuthToken() {
  return localStorage.getItem('jwt_token');
}

export function isAuthenticated() {
  return !!getAuthToken();
}

export function logout() {
  localStorage.removeItem('jwt_token');
  window.location.href = '/login';
}

async function apiRequest(endpoint, options = {}) {
  const token = getAuthToken();
  
  const config = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    }
  };

  // Add auth header if token exists
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, config);

  // Handle 401 Unauthorized
  if (response.status === 401) {
    logout();
    throw new Error('Session expired. Please login again.');
  }

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Request failed');
  }

  return await response.json();
}

// Authenticated API calls
export const api = {
  // User endpoints
  getCurrentUser: () => apiRequest('/users/me'),
  updateProfile: (data) => apiRequest('/users/me', {
    method: 'PUT',
    body: JSON.stringify(data)
  }),
  deleteAccount: () => apiRequest('/users/me', { method: 'DELETE' }),
  
  // Post endpoints
  createPost: (content, visibility) => apiRequest('/posts', {
    method: 'POST',
    body: JSON.stringify({ content, visibility })
  }),
  getPost: (id) => apiRequest(`/posts/${id}`),
  updatePost: (id, content) => apiRequest(`/posts/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ content })
  }),
  deletePost: (id) => apiRequest(`/posts/${id}`, { method: 'DELETE' }),
  getFeed: () => apiRequest('/posts/feed'),
  
  // Follow endpoints
  followUser: (userId) => apiRequest(`/users/${userId}/follow`, {
    method: 'POST'
  }),
  unfollowUser: (userId) => apiRequest(`/users/${userId}/follow`, {
    method: 'DELETE'
  })
};
```

### Auth Context (React)

```javascript
// AuthContext.jsx
import React, { createContext, useState, useEffect, useContext } from 'react';
import { api, isAuthenticated, logout } from './api';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (isAuthenticated()) {
      loadUser();
    } else {
      setLoading(false);
    }
  }, []);

  async function loadUser() {
    try {
      const userData = await api.getCurrentUser();
      setUser(userData);
    } catch (error) {
      console.error('Failed to load user:', error);
      logout();
    } finally {
      setLoading(false);
    }
  }

  const value = {
    user,
    loading,
    isAuthenticated: !!user,
    logout,
    refreshUser: loadUser
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
```

---

## 🎯 Task 5: User Profile Management

**Priority:** 🟡 MEDIUM  
**Dependencies:** Task 4  
**Estimated Effort:** 3-4 hours

### Profile View Component

```javascript
// ProfilePage.jsx
import React, { useState, useEffect } from 'react';
import { useAuth } from './AuthContext';
import { api } from './api';

export function ProfilePage() {
  const { user, refreshUser } = useAuth();
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState({
    display_name: '',
    bio: '',
    avatar_url: ''
  });

  useEffect(() => {
    if (user) {
      setFormData({
        display_name: user.display_name,
        bio: user.bio || '',
        avatar_url: user.avatar_url || ''
      });
    }
  }, [user]);

  const handleUpdate = async (e) => {
    e.preventDefault();
    try {
      await api.updateProfile(formData);
      await refreshUser();
      setEditing(false);
      alert('Profile updated successfully!');
    } catch (error) {
      alert(`Error: ${error.message}`);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure? This cannot be undone!')) return;
    
    try {
      await api.deleteAccount();
      alert('Account deleted');
      window.location.href = '/';
    } catch (error) {
      alert(`Error: ${error.message}`);
    }
  };

  if (!user) return <div>Loading...</div>;

  return (
    <div className="profile-page">
      <h1>Profile</h1>
      
      {!editing ? (
        <div className="profile-view">
          <img src={user.avatar_url || '/default-avatar.png'} alt="Avatar" />
          <h2>{user.display_name}</h2>
          <p className="username">@{user.username}</p>
          <p className="did">{user.did}</p>
          <p className="bio">{user.bio}</p>
          
          <button onClick={() => setEditing(true)}>Edit Profile</button>
          <button onClick={handleDelete} className="danger">Delete Account</button>
        </div>
      ) : (
        <form onSubmit={handleUpdate} className="profile-edit">
          <input
            type="text"
            value={formData.display_name}
            onChange={(e) => setFormData({...formData, display_name: e.target.value})}
            placeholder="Display Name"
            required
          />
          
          <textarea
            value={formData.bio}
            onChange={(e) => setFormData({...formData, bio: e.target.value})}
            placeholder="Bio"
          />
          
          <input
            type="url"
            value={formData.avatar_url}
            onChange={(e) => setFormData({...formData, avatar_url: e.target.value})}
            placeholder="Avatar URL"
          />
          
          <button type="submit">Save</button>
          <button type="button" onClick={() => setEditing(false)}>Cancel</button>
        </form>
      )}
    </div>
  );
}
```

---

## 🎯 Task 6: Post Creation & Feed

**Priority:** 🟡 MEDIUM  
**Dependencies:** Task 4  
**Estimated Effort:** 4-6 hours

### Post Composer

```javascript
// PostComposer.jsx
import React, { useState } from 'react';
import { api } from './api';

export function PostComposer({ onPostCreated }) {
  const [content, setContent] = useState('');
  const [visibility, setVisibility] = useState('public');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    try {
      await api.createPost(content, visibility);
      setContent('');
      onPostCreated?.();
    } catch (error) {
      alert(`Error: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="post-composer">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="What's on your mind?"
        maxLength={500}
        required
      />
      
      <div className="composer-footer">
        <select value={visibility} onChange={(e) => setVisibility(e.target.value)}>
          <option value="public">Public</option>
          <option value="followers">Followers</option>
          <option value="private">Private</option>
        </select>
        
        <span className="char-count">{content.length}/500</span>
        
        <button type="submit" disabled={loading || !content.trim()}>
          {loading ? 'Posting...' : 'Post'}
        </button>
      </div>
    </form>
  );
}
```

### Feed Component

```javascript
// Feed.jsx
import React, { useState, useEffect } from 'react';
import { api } from './api';
import { PostComposer } from './PostComposer';
import { PostCard } from './PostCard';

export function Feed() {
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadFeed();
  }, []);

  async function loadFeed() {
    try {
      const data = await api.getFeed();
      setPosts(data);
    } catch (error) {
      console.error('Failed to load feed:', error);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="feed">
      <PostComposer onPostCreated={loadFeed} />
      
      {loading ? (
        <div>Loading feed...</div>
      ) : (
        <div className="posts">
          {posts.map(post => (
            <PostCard key={post.id} post={post} onUpdate={loadFeed} />
          ))}
        </div>
      )}
    </div>
  );
}
```

### Post Card

```javascript
// PostCard.jsx
import React from 'react';
import { api } from './api';
import { useAuth } from './AuthContext';

export function PostCard({ post, onUpdate }) {
  const { user } = useAuth();
  const isOwner = user?.id === post.author_id;

  const handleDelete = async () => {
    if (!confirm('Delete this post?')) return;
    
    try {
      await api.deletePost(post.id);
      onUpdate?.();
    } catch (error) {
      alert(`Error: ${error.message}`);
    }
  };

  return (
    <div className="post-card">
      <div className="post-header">
        <img src={post.author_avatar || '/default-avatar.png'} alt="" />
        <div>
          <strong>{post.author_name}</strong>
          <span className="username">@{post.author_username}</span>
        </div>
        <span className="timestamp">{new Date(post.created_at).toLocaleString()}</span>
      </div>
      
      <div className="post-content">
        {post.content}
      </div>
      
      {isOwner && (
        <div className="post-actions">
          <button onClick={handleDelete}>Delete</button>
        </div>
      )}
    </div>
  );
}
```

---

## 🎯 Task 7: Follow System

**Priority:** 🟢 LOW  
**Dependencies:** Task 4  
**Estimated Effort:** 2-3 hours

### Follow Button Component

```javascript
// FollowButton.jsx
import React, { useState } from 'react';
import { api } from './api';

export function FollowButton({ userId, initialFollowing = false }) {
  const [following, setFollowing] = useState(initialFollowing);
  const [loading, setLoading] = useState(false);

  const handleToggle = async () => {
    setLoading(true);
    try {
      if (following) {
        await api.unfollowUser(userId);
        setFollowing(false);
      } else {
        await api.followUser(userId);
        setFollowing(true);
      }
    } catch (error) {
      alert(`Error: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <button 
      onClick={handleToggle} 
      disabled={loading}
      className={following ? 'following' : 'follow'}
    >
      {loading ? '...' : following ? 'Unfollow' : 'Follow'}
    </button>
  );
}
```

---

## 🎯 Task 8: Error Handling & UX

**Priority:** 🟡 MEDIUM  
**Dependencies:** All previous tasks  
**Estimated Effort:** 2-3 hours

### Global Error Handler

```javascript
// ErrorBoundary.jsx
import React from 'react';

export class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="error-page">
          <h1>Something went wrong</h1>
          <p>{this.state.error?.message}</p>
          <button onClick={() => window.location.href = '/'}>
            Go Home
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
```

### Toast Notifications

```javascript
// toast.js
export function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.textContent = message;
  
  document.body.appendChild(toast);
  
  setTimeout(() => {
    toast.classList.add('show');
  }, 100);
  
  setTimeout(() => {
    toast.classList.remove('show');
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}
```

---

## 🎯 Task 9: Key Backup & Recovery

**Priority:** 🟢 LOW  
**Dependencies:** Task 1  
**Estimated Effort:** 3-4 hours

### Export Private Key

```javascript
// keyBackup.js
import { getPrivateKey } from './crypto';

export async function exportPrivateKey(password) {
  const secretKey = await getPrivateKey();
  
  // Encrypt with password (simplified - use proper encryption in production)
  const encrypted = await encryptKey(secretKey, password);
  
  // Create download
  const blob = new Blob([encrypted], { type: 'application/octet-stream' });
  const url = URL.createObjectURL(blob);
  
  const a = document.createElement('a');
  a.href = url;
  a.download = 'splitter-key-backup.bin';
  a.click();
  
  URL.revokeObjectURL(url);
}

async function encryptKey(key, password) {
  // Use Web Crypto API for proper encryption
  const encoder = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );
  
  // Derive encryption key
  const encKey = await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: encoder.encode('splitter-salt'),
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt']
  );
  
  // Encrypt
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    encKey,
    key
  );
  
  // Combine IV and encrypted data
  const result = new Uint8Array(iv.length + encrypted.byteLength);
  result.set(iv);
  result.set(new Uint8Array(encrypted), iv.length);
  
  return result;
}
```

---

## 📦 Required Dependencies

Add these to your `package.json`:

```json
{
  "dependencies": {
    "tweetnacl": "^1.0.3",
    "tweetnacl-util": "^0.15.1",
    "idb": "^7.1.1"
  }
}
```

---

## ✅ Testing Checklist

### Task 1: Crypto
- [ ] Generate keypair successfully
- [ ] Create valid DID format
- [ ] Store key in IndexedDB
- [ ] Retrieve stored key
- [ ] Sign challenge correctly
- [ ] Verify signature works

### Task 2: Registration
- [ ] Form validation works
- [ ] Keypair generated on submit
- [ ] DID displayed to user
- [ ] Backend registration succeeds
- [ ] JWT token stored
- [ ] Redirect to home page

### Task 3: Login
- [ ] Challenge requested successfully
- [ ] Private key loaded from storage
- [ ] Challenge signed correctly
- [ ] Verification succeeds
- [ ] JWT token stored
- [ ] Redirect to home page

### Task 4: Auth Requests
- [ ] Token included in requests
- [ ] 401 redirects to login
- [ ] Logout clears token

### Task 5: Profile
- [ ] View profile loads
- [ ] Edit profile works
- [ ] Delete account works (with confirmation)

### Task 6: Posts
- [ ] Create post works
- [ ] Feed loads posts
- [ ] Delete post works (owner only)

### Task 7: Follow
- [ ] Follow user works
- [ ] Unfollow user works
- [ ] Button state updates

### Task 8: Errors
- [ ] Network errors handled
- [ ] Validation errors displayed
- [ ] Auth errors redirect

### Task 9: Backup
- [ ] Export key works
- [ ] Import key works
- [ ] Encrypted properly

---

## 🎓 Learning Resources

- [Ed25519 Signatures](https://ed25519.cr.yp.to/)
- [TweetNaCl.js Documentation](https://github.com/dchest/tweetnacl-js)
- [IndexedDB API](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API)
- [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
- [W3C DID Core](https://www.w3.org/TR/did-core/)

---

**Questions?** Check the main README.md or open an issue!
