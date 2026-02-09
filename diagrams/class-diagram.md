# Class Diagram

This diagram shows the object-oriented design of the Splitter federated social media platform, including all major entities, value objects, services, and their relationships organized by Epic.

```mermaid
classDiagram
    direction TD

    %% --- Epic 1: Identity & Autonomy ---
    class User {
        <<Entity>>
        +UUID ID
        +String Username
        +String Domain
        +String DID (did:key)
        +PublicKey CurrentPublicKey
        +Boolean IsLocal
        +authenticate(signature, nonce)
        +exportDigitalSuitcase()
    }

    class IdentityKeys {
        <<LocalOnly - Browser>>
        +PrivateKey PrivateKey
        +PublicKey PublicKey
        +KeyRotationHistory History
        +sign(data)
        +rotateKeys()
    }

    %% D. Authentication Boundary (New)
    class AuthChallenge {
        <<ValueObject>>
        +String nonce
        +DateTime expiresAt
    }

    %% --- Epic 3 & 4: Social & Messaging ---
    class Post {
        <<Entity>>
        +UUID ID
        +String Content
        +String AuthorDID
        +DateTime CreatedAt
        +Boolean IsEphemeral
        +String Visibility
        +PostType Type
        +applyTombstone()
    }

    class Interaction {
        <<ValueObject>>
        +Type (Like/Repost/Reply)
        +UUID TargetID
        +DateTime Timestamp
    }

    class EncryptedMessage {
        <<Entity>>
        +UUID ID
        +Byte[] Ciphertext
        +String RecipientDID
        +String SenderDID
    }

    %% C. Timeline Representation (New)
    class Timeline {
        <<Entity>>
        +String type (Home/Local/Federated)
        +String ownerDID
    }

    %% --- Epic 2: Federation Infrastructure (Expanded) ---
    class ActivityPubObject {
        <<Interface>>
        +String Context
        +String Type
        +Object Actor
        +serializeToJSONLD()
    }

    class InboxActivity {
        <<Entity>>
        +UUID id
        +String activityID
        +JSON payload
        +DateTime receivedAt
    }

    class OutboxActivity {
        <<Entity>>
        +UUID id
        +String activityType
        +String targetDomain
        +int retryCount
    }

    class RetryQueue {
        <<Service>>
        +enqueue(activity)
        +processWithBackoff()
    }

    class FederationNode {
        <<Entity>>
        +String Domain
        +Float ReputationScore
        +Boolean IsBlocked
        +updateReputation(signal)
    }

    %% --- Epic 5: Governance & Moderation (New) ---
    class Report {
        <<Entity>>
        +id
        +reporterDID
        +targetPostID
        +String reason
        +String status
    }

    class AdminAction {
        <<Entity>>
        +id
        +adminID
        +actionType
        +target
        +timestamp
    }

    %% --- Relationships ---
    
    %% Core Social & Identity
    User "1" *-- "1" IdentityKeys : ManagedBy (Browser)
    User "1" -- "*" Post : Creates
    User "1" -- "*" Interaction : Performs
    Post "1" -- "*" Interaction : Receives
    User "1" -- "*" EncryptedMessage : Owns
    User "1" -- "1" Timeline : Views
    Timeline "1" o-- "*" Post : Aggregates
    User ..> AuthChallenge : "Responds to"

    %% Governance Connections
    Report --> Post : "Flags"
    AdminAction --> User : "Moderates"
    AdminAction ..> Report : "Resolves"

    %% Federation Connections
    ActivityPubObject <|-- Post : Extends
    ActivityPubObject <|-- Interaction : Extends
    InboxActivity ..> ActivityPubObject : "Contains"
    OutboxActivity ..> ActivityPubObject : "Wraps"
    RetryQueue --> OutboxActivity : "ManagesDelivery"
    FederationNode "1" -- "*" ActivityPubObject : Exchanges
```

## Class Categories

### Epic 1: Identity & Autonomy

**User** (Entity)
- Core user entity with DID-based identity
- Supports authentication via signature verification
- Can export digital suitcase for portability

**IdentityKeys** (LocalOnly - Browser)
- Private key management in browser
- Never sent to server
- Supports key rotation

**AuthChallenge** (ValueObject)
- Challenge-response authentication
- Time-limited nonce for security

### Epic 3 & 4: Social & Messaging

**Post** (Entity)
- User-generated content
- Supports ephemeral posts (stories)
- Visibility controls (public, followers, circle)
- Tombstone support for deletion

**Interaction** (ValueObject)
- Likes, reposts, and replies
- Immutable interaction records

**EncryptedMessage** (Entity)
- End-to-end encrypted messaging
- Server stores only ciphertext

**Timeline** (Entity)
- Aggregates posts for different views
- Types: Home, Local, Federated

### Epic 2: Federation Infrastructure

**ActivityPubObject** (Interface)
- Base interface for ActivityPub activities
- JSON-LD serialization

**InboxActivity** (Entity)
- Incoming federated activities
- Deduplication and verification

**OutboxActivity** (Entity)
- Outgoing federated activities
- Retry tracking

**RetryQueue** (Service)
- Manages failed delivery retries
- Exponential backoff

**FederationNode** (Entity)
- Remote server representation
- Reputation scoring
- Domain blocking

### Epic 5: Governance & Moderation

**Report** (Entity)
- Content moderation reports
- Status tracking

**AdminAction** (Entity)
- Audit log for admin actions
- Moderates users and resolves reports

## Key Relationships

### Composition & Aggregation
- User **owns** IdentityKeys (composition, browser-only)
- Timeline **aggregates** Posts (aggregation)

### Associations
- User **creates** Posts (1:N)
- User **performs** Interactions (1:N)
- Post **receives** Interactions (1:N)
- User **owns** EncryptedMessages (1:N)

### Dependencies
- User **responds to** AuthChallenge
- Report **flags** Post
- AdminAction **moderates** User
- AdminAction **resolves** Report

### Inheritance
- Post **extends** ActivityPubObject
- Interaction **extends** ActivityPubObject

### Federation
- InboxActivity **contains** ActivityPubObject
- OutboxActivity **wraps** ActivityPubObject
- RetryQueue **manages delivery** of OutboxActivity
- FederationNode **exchanges** ActivityPubObject

## Design Patterns

- **Entity**: Domain objects with identity (User, Post, etc.)
- **Value Object**: Immutable objects without identity (Interaction, AuthChallenge)
- **Service**: Stateless operations (RetryQueue)
- **Interface**: Contract for polymorphism (ActivityPubObject)
