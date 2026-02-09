# Activity Diagram

This flowchart shows the complete user journey and system workflows in the Splitter federated social media platform, from initial signup through all major features including posting, federation, messaging, and moderation.

```mermaid
flowchart TD

  %% ENTRY / ONBOARDING
  A([Start Visitor opens Splitter]) --> B[Browse instances and read intro]
  B --> C{Choose action}

  C -->|Sign Up| D[Generate DID and Keypair in browser]
  D --> D1[Store Private Key in IndexedDB]
  D1 --> E[Send Public DID and Public Key to Server]
  E --> F[Create User and DIDIdentity in PostgreSQL]
  F --> G[Onboarding set privacy defaults and recovery]
  G --> H([Account Ready])

  C -->|Login| I[Request login nonce challenge]
  I --> I1[Backend stores nonce in Redis TTL]
  I1 --> J[Frontend signs nonce using Private Key]
  J --> K[Backend verifies signature using Public Key]
  K --> L{Valid login}
  L -->|No| L1([Reject Login])
  L -->|Yes| M([Session Token Issued])

  %% MAIN NAVIGATION
  H --> N[Open App Home]
  M --> N

  N --> O{Choose module}
  O -->|Timelines| P
  O -->|Create Post| Q
  O -->|Search and Discovery| S
  O -->|Direct Messages| T
  O -->|Settings and Identity| U
  O -->|Reporting| V
  O -->|Admin Dashboard| W

  %% TIMELINES
  P[Select timeline Home Local Federated] --> P1[Fetch timeline with cursor pagination]
  P1 --> P2[Backend queries PostgreSQL]
  P2 --> P3[Return posts and next cursor]
  P3 --> P4[Cache timeline in IndexedDB for offline read]
  P4 --> O

  %% CREATE POST
  Q[Compose post text media visibility] --> Q1{Media attached}
  Q1 -->|Yes| Q2[Upload media to MinIO or IPFS]
  Q1 -->|No| Q3[Skip upload]
  Q2 --> Q4[Create Post record in PostgreSQL]
  Q3 --> Q4

  Q4 --> Q5{Post type}
  Q5 -->|Normal| Q6[Generate ActivityPub Create JSON]
  Q5 -->|Story| Q7[Store Story and schedule TTL cleanup]
  Q7 --> Q6

  Q6 --> Q8[Sign outgoing federation request using HTTP signatures]
  Q8 --> Q9[Enqueue delivery job in Redis queue]
  Q9 --> Q10([Outbox Worker picks job])

  Q10 --> Q11{Remote server reachable}
  Q11 -->|Yes| Q12[POST signed activity to Remote Inbox]
  Q11 -->|No| Q13[Schedule retry exponential backoff]

  Q12 --> Q14{Remote success}
  Q14 -->|Yes| Q15([Delivery complete])
  Q14 -->|No| Q13

  Q13 --> Q16[Push retry item back to Redis]
  Q16 --> Q10
  Q15 --> O

  %% INTERACTIONS
  P3 --> R[User interacts Like Reply Repost Bookmark]
  R --> R1{Interaction type}

  R1 -->|Bookmark| R2[Store bookmark locally in PostgreSQL]
  R1 -->|Like Reply Repost| R3[Store interaction in PostgreSQL]
  R3 --> R4[Generate ActivityPub interaction activity]
  R4 --> R5[Sign and enqueue to Redis for federation]
  R5 --> Q10
  R2 --> O

  %% SEARCH + WEBFINGER
  S[Search keyword hashtag or handle] --> S1{Remote handle}
  S1 -->|Yes| S2[WebFinger lookup and resolve DID]
  S2 --> S3[Store remote actor reference in PostgreSQL]
  S3 --> O
  S1 -->|No| S4[Search local index and cached federated posts]
  S4 --> O

  %% DIRECT MESSAGES
  T[Open DM inbox] --> T1[Fetch DM thread list]
  T1 --> T2[User composes message]
  T2 --> T3[Fetch recipient public key]
  T3 --> T4[Encrypt message on client E2EE]
  T4 --> T5[Send cipherBlob to server]
  T5 --> T6[Store cipherBlob in PostgreSQL]
  T6 --> T7{Recipient remote}
  T7 -->|Yes| T8[Send federated encrypted activity]
  T7 -->|No| T9([DM delivered locally])
  T8 --> Q10
  T9 --> O

  %% SETTINGS / IDENTITY
  U[Identity and Security settings] --> U1{Action}
  U1 -->|Key Rotation| U2[Client generates new keypair]
  U2 --> U3[Sign rotation request with old private key]
  U3 --> U4[Server updates public key and revocation list]
  U4 --> U5[Propagate Update activity to followers]
  U5 --> Q10

  U1 -->|Export Suitcase| U6[Export JSON data and keys as encrypted bundle]
  U6 --> O
  U1 -->|Import Suitcase| U7[Import bundle into new instance]
  U7 --> O

  %% REPORTING
  V[User reports post spam harassment] --> V1[Store report in PostgreSQL]
  V1 --> V2[Notify admin queue]
  V2 --> O

  %% ADMIN DASHBOARD
  W[Admin dashboard] --> W1{Admin action}
  W1 -->|Moderation queue| W2[Fetch reports from PostgreSQL]
  W2 --> W3[Resolve warn suspend delete]
  W3 --> O

  W1 -->|Block domain| W4[Update domain blocklist in PostgreSQL]
  W4 --> W5[Stop sending requests to blocked domains]
  W5 --> O

  W1 -->|Inspect federation stats| W6[Read retry queue and inbox outbox logs]
  W6 --> O

  W1 -->|View server graph| W7[Generate connected server graph]
  W7 --> O

  %% REMOTE INBOX SIDE
  Q12 --> X[Remote Inbox receives activity JSON]
  X --> X1[Verify HTTP signature]
  X1 --> X2{Signature valid}
  X2 -->|No| X3([Reject])
  X2 -->|Yes| X4[Deduplicate using activity id hash TTL]
  X4 --> X5{Duplicate}
  X5 -->|Yes| X6([Drop silently])
  X5 -->|No| X7[Store activity and update timelines]
  X7 --> X8([Remote users see updates])
```

## Key Workflows Covered

### 1. Onboarding & Authentication
- **Signup**: DID and keypair generation in browser, private key stored locally
- **Login**: Challenge-response authentication with signature verification

### 2. Main Navigation
Central hub with access to all major modules:
- Timelines
- Post creation
- Search & discovery
- Direct messages
- Settings & identity management
- Reporting
- Admin dashboard

### 3. Timeline Viewing
- Home, Local, and Federated timelines
- Cursor-based pagination
- Offline caching in IndexedDB

### 4. Post Creation & Federation
- Text and media posts
- Story posts with TTL
- ActivityPub Create activity generation
- HTTP signature signing
- Async delivery via Redis queue
- Retry mechanism with exponential backoff

### 5. User Interactions
- Bookmarks (local only)
- Likes, replies, reposts (federated)
- ActivityPub activity generation and delivery

### 6. Search & Discovery
- Local search for posts and users
- WebFinger lookup for remote handles
- Remote actor caching

### 7. Direct Messaging (E2EE)
- End-to-end encryption on client
- Server stores only ciphertext
- Federated encrypted message delivery

### 8. Identity Management
- Key rotation with signature verification
- Export/import identity "suitcase"
- Update activity propagation

### 9. Content Moderation
- User reporting system
- Admin moderation queue
- Domain blocking (defederation)
- Federation statistics monitoring

### 10. Remote Inbox Processing
- HTTP signature verification
- Activity deduplication
- Timeline updates for remote users

## System Components

- **Frontend**: Browser-based DID generation, encryption, signing
- **IndexedDB**: Client-side key storage and offline cache
- **Backend API**: Go + Echo framework
- **PostgreSQL**: Primary data store
- **Redis**: Job queues and caching
- **Outbox Worker**: Async federation delivery
- **Media Store**: MinIO or IPFS
- **Remote Servers**: Federated ActivityPub instances
