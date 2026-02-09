# Sequence Diagrams

This document shows the complete interaction flows for the Splitter federated social media platform, including authentication, post creation, federation, messaging, and moderation.

```mermaid
sequenceDiagram
  autonumber

  actor Visitor
  actor User
  actor Admin
  participant FE as Frontend (Next.js + Tailwind)
  participant IDB as IndexedDB (Keys + Offline Cache)
  participant BE as Backend API (Go + Echo)
  participant PG as PostgreSQL
  participant R as Redis (Cache + Queues)
  participant OUT as Outbox Worker (Go)
  participant IN as Inbox Endpoint (ActivityPub)
  participant MEDIA as Media Store (MinIO/IPFS)
  participant Remote as Remote Server (ActivityPub)

  %% =========================
  %% 1) Visitor -> Signup (DID generated client-side)
  %% =========================
  Visitor->>FE: Browse instances + select home server
  Visitor->>FE: Click Sign Up

  FE->>FE: Generate DID + Keypair (did:key)
  FE->>IDB: Store Private Key locally (never sent)
  FE->>BE: Send Signup Request (DID + Public Key + profile info)

  BE->>PG: Create User + DIDIdentity record
  PG-->>BE: User created
  BE-->>FE: Signup success

  %% =========================
  %% 2) Login (Challenge-Response)
  %% =========================
  User->>FE: Enter handle + password
  FE->>BE: Request login challenge (nonce)
  BE->>R: Store nonce (TTL)
  R-->>BE: OK
  BE-->>FE: Return nonce

  FE->>IDB: Load private key
  FE->>FE: Sign nonce with private key
  FE->>BE: Send signed nonce + DID

  BE->>R: Fetch nonce
  R-->>BE: nonce
  BE->>PG: Fetch public key for DID
  PG-->>BE: Public key
  BE->>BE: Verify signature

  alt Valid signature
    BE-->>FE: Issue session token
  else Invalid signature
    BE-->>FE: Reject login
  end

  %% =========================
  %% 3) User creates a post (text/media/story)
  %% =========================
  User->>FE: Create Post (text + optional image)
  FE->>MEDIA: Upload media
  MEDIA-->>FE: Return media URL(s)

  FE->>BE: Create Post(content, mediaURLs, visibility)
  BE->>PG: Store Post + Media metadata
  PG-->>BE: Post saved
  BE-->>FE: Post created

  %% =========================
  %% 4) Federation Outbox Delivery (async)
  %% =========================
  BE->>R: Enqueue ActivityPub Create job
  R-->>BE: Queued

  OUT->>R: Dequeue delivery job
  R-->>OUT: Job payload

  OUT->>PG: Fetch follower inbox URLs (local + remote)
  PG-->>OUT: Inbox targets list

  OUT->>OUT: Build JSON-LD ActivityPub Create
  OUT->>OUT: Sign request (HTTP Signatures)

  OUT->>Remote: POST Activity to Remote Inbox
  alt Remote reachable
    Remote-->>OUT: 202 Accepted
  else Remote down
    OUT->>R: Push into Retry Queue (exp backoff)
  end

  %% =========================
  %% 5) Remote server receives activity (Inbox + Dedup)
  %% =========================
  Remote->>IN: Deliver incoming JSON-LD activity
  IN->>IN: Verify HTTP Signature
  IN->>R: Check Activity ID hash (Dedup TTL)
  alt Duplicate
    IN-->>Remote: Drop silently
  else New activity
    IN->>PG: Store activity + post
    PG-->>IN: Stored
    IN-->>Remote: Accepted
  end

  %% =========================
  %% 6) User loads timelines (Home/Local/Federated)
  %% =========================
  User->>FE: Open Home Timeline
  FE->>BE: GET /timeline/home (cursor pagination)
  BE->>PG: Query posts + pagination cursor
  PG-->>BE: Timeline posts
  BE-->>FE: Timeline response

  FE->>IDB: Cache timeline for offline read

  %% =========================
  %% 7) User interacts (Like / Reply / Repost)
  %% =========================
  User->>FE: Like / Reply / Repost a post
  FE->>BE: POST /interactions
  BE->>PG: Store interaction
  PG-->>BE: Stored
  BE->>R: Enqueue federation interaction activity
  BE-->>FE: Interaction confirmed

  %% =========================
  %% 8) Direct Message (E2EE)
  %% =========================
  User->>FE: Send DM to recipient
  FE->>BE: Fetch recipient DID public key
  BE->>PG: Lookup recipient public key
  PG-->>BE: Public key
  BE-->>FE: Public key

  FE->>FE: Encrypt message locally (E2EE)
  FE->>BE: Send cipherBlob only
  BE->>PG: Store cipherBlob (server cannot decrypt)
  PG-->>BE: Stored
  BE-->>FE: DM delivered (stored)

  %% =========================
  %% 9) Admin Defederation + Moderation
  %% =========================
  Admin->>FE: Block remote domain
  FE->>BE: POST /admin/block-domain
  BE->>PG: Store ServerDomainBlock
  PG-->>BE: Updated
  BE->>OUT: Stop sending requests to blocked domain
  BE-->>FE: Domain blocked

  Admin->>FE: View moderation queue
  FE->>BE: GET /admin/moderation
  BE->>PG: Fetch reports
  PG-->>BE: Reports list
  BE-->>FE: Moderation queue
```

## Key Flows Covered

### 1. User Registration (Steps 1-7)
- Visitor browses instances and selects home server
- DID and keypair generated client-side
- Private key stored locally in IndexedDB (never sent to server)
- User record created in PostgreSQL

### 2. Authentication (Steps 8-18)
- Challenge-response authentication using DID signatures
- Nonce stored in Redis with TTL
- Signature verification using public key from database
- Session token issued on successful authentication

### 3. Post Creation (Steps 19-24)
- Media upload to media store (MinIO/IPFS)
- Post and media metadata stored in PostgreSQL
- Post creation confirmed to frontend

### 4. Federation Outbox (Steps 25-35)
- ActivityPub Create activity enqueued in Redis
- Outbox worker processes delivery jobs
- HTTP Signatures used for request signing
- Retry queue for failed deliveries with exponential backoff

### 5. Federation Inbox (Steps 36-43)
- Incoming activities verified via HTTP Signatures
- Activity deduplication using Redis cache
- Activities and posts stored in PostgreSQL

### 6. Timeline Loading (Steps 44-49)
- Cursor-based pagination for efficient loading
- Timeline cached in IndexedDB for offline access

### 7. Post Interactions (Steps 50-55)
- Likes, replies, and reposts stored in database
- Federation activities enqueued for remote delivery

### 8. Direct Messaging (Steps 56-64)
- End-to-end encryption (E2EE)
- Messages encrypted client-side before sending
- Server stores only ciphertext (cannot decrypt)

### 9. Moderation & Defederation (Steps 65-75)
- Admin can block remote domains
- Outbox worker stops deliveries to blocked domains
- Moderation queue for content reports

## Components

- **Frontend**: Next.js + Tailwind CSS
- **IndexedDB**: Client-side key storage and offline cache
- **Backend API**: Go + Echo framework
- **PostgreSQL**: Primary data store
- **Redis**: Caching and job queues
- **Outbox Worker**: Async federation delivery
- **Inbox Endpoint**: ActivityPub activity receiver
- **Media Store**: MinIO or IPFS for media files
- **Remote Server**: Other federated instances
