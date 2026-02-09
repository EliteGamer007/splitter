# Architecture Diagram

This diagram shows the complete system architecture of the Splitter federated social media platform, including all layers from the browser client to external federated instances.

```mermaid
graph TB
    subgraph Browser ["Web Browser"]
        subgraph Client_Layer ["Client Side (Next.js)"]
            UI["Frontend UI (Tailwind/Shadcn)"]
            IDB[("IndexedDB (Private Keys/Local Cache)")]
            UI <--> IDB
        end
    end

    subgraph API_Layer ["Backend API (Go + Echo)"]
        Auth["Identity & Auth (Challenge-Response)"]
        Social["Social Module (Feed/Posts)"]
        E2EE["E2EE Extension Module"]
    end

    subgraph Data_Layer ["Persistence & Messaging"]
        DB[(PostgreSQL)]
        Cache[(Redis - Queues/Cache)]
    end

    subgraph Federation_Layer ["Federation Engine (ActivityPub)"]
        Inbox["ActivityPub Inbox"]
        Outbox["ActivityPub Outbox"]
        Rep["Reputation System"]
    end

    subgraph External ["The Federations"]
        Remote["Remote Instances (Mastodon/Other Splitters)"]
    end

    %% Connections
    UI -- "HTTPS/JSON (Auth)" --> Auth
    UI -- "REST API (Posts/Social)" --> Social
    Social --> E2EE
    
    Auth & Social --> DB
    Social --> Cache
    
    Cache -- "Worker Jobs" --> Outbox
    Outbox -- "Signed JSON-LD" --> Remote
    Remote -- "Incoming Activities" --> Inbox
    Inbox --> DB
    Inbox --> Rep

    %% Styling and Colors
    classDef browser fill:#ffffff,stroke:#333,stroke-width:2px,stroke-dasharray: 5 5;
    classDef client fill:#e1f5fe,stroke:#01579b,stroke-width:2px,color:#000;
    classDef api fill:#f3e5f5,stroke:#4a148c,stroke-width:2px,color:#000;
    classDef data fill:#fff3e0,stroke:#e65100,stroke-width:2px,color:#000;
    classDef fed fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px,color:#000;
    classDef ext fill:#fafafa,stroke:#212121,stroke-dasharray: 5 5,color:#000;

    class Browser browser;
    class UI,IDB client;
    class Auth,Social,E2EE api;
    class DB,Cache data;
    class Inbox,Outbox,Rep fed;
    class Remote ext;
```

## Architecture Layers

### 1. Browser Layer (Client Side)
**Frontend UI** (Next.js + Tailwind + Shadcn)
- User interface components
- DID generation and key management
- Client-side encryption for E2EE messages

**IndexedDB**
- Private key storage (never sent to server)
- Offline content cache
- Timeline caching for offline reading

### 2. Backend API Layer (Go + Echo)
**Identity & Auth Module**
- Challenge-response authentication
- DID verification
- Signature validation

**Social Module**
- Post creation and management
- Timeline aggregation (Home/Local/Federated)
- Interaction handling (likes, replies, reposts)

**E2EE Extension Module**
- Encrypted message storage (ciphertext only)
- Public key distribution
- Message thread management

### 3. Data Layer
**PostgreSQL**
- Primary data store for all entities
- User profiles, posts, interactions
- Federation activities and remote actors

**Redis**
- Job queues for async federation delivery
- Caching for performance
- Nonce storage for authentication
- Activity deduplication

### 4. Federation Layer (ActivityPub)
**ActivityPub Inbox**
- Receives incoming federated activities
- HTTP signature verification
- Activity deduplication
- Stores remote posts and interactions

**ActivityPub Outbox**
- Sends outgoing activities to remote instances
- HTTP signature signing
- Retry mechanism with exponential backoff
- Delivery tracking

**Reputation System**
- Monitors instance health
- Tracks delivery success/failure rates
- Circuit breaker for problematic instances

### 5. External Federation
**Remote Instances**
- Other Splitter instances
- Mastodon servers
- Any ActivityPub-compatible platform

## Data Flow

### Authentication Flow
1. User enters credentials in **Frontend UI**
2. **Auth Module** generates nonce challenge
3. Frontend signs nonce with private key from **IndexedDB**
4. **Auth Module** verifies signature using public key from **PostgreSQL**
5. Session token issued on success

### Post Creation & Federation Flow
1. User creates post in **Frontend UI**
2. **Social Module** stores post in **PostgreSQL**
3. Job enqueued in **Redis** queue
4. **Outbox Worker** picks job and signs activity
5. Signed JSON-LD sent to **Remote Instances**
6. Remote instances send to their **Inbox**
7. **Reputation System** tracks delivery success

### Incoming Federation Flow
1. **Remote Instance** sends activity to **Inbox**
2. **Inbox** verifies HTTP signature
3. Activity deduplicated via **Redis**
4. Activity and content stored in **PostgreSQL**
5. **Reputation System** updated

### E2EE Messaging Flow
1. User composes message in **Frontend UI**
2. Message encrypted client-side using recipient's public key
3. **E2EE Module** stores only ciphertext in **PostgreSQL**
4. For remote recipients, encrypted activity sent via **Outbox**

## Technology Stack

| Layer | Technologies |
|-------|-------------|
| **Frontend** | Next.js, React, Tailwind CSS, Shadcn UI |
| **Client Storage** | IndexedDB, Web Crypto API |
| **Backend** | Go 1.21+, Echo framework |
| **Database** | PostgreSQL 15 (Neon Cloud) |
| **Cache/Queue** | Redis |
| **Federation** | ActivityPub, JSON-LD, HTTP Signatures |
| **Encryption** | Ed25519 (DID), bcrypt (passwords), E2EE (messages) |

## Security Features

- **DID-based Identity**: Decentralized identifiers with client-side key generation
- **Challenge-Response Auth**: No passwords sent over the wire
- **HTTP Signatures**: All federated requests cryptographically signed
- **E2EE Messaging**: Server cannot decrypt messages
- **Reputation System**: Automatic blocking of malicious instances
- **Activity Deduplication**: Prevents replay attacks
