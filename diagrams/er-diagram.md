# Entity-Relationship Diagram

This diagram shows the complete database schema for the Splitter federated social media platform, including all tables, relationships, and key constraints.

## Core Tables

### Users & Identity Domain

```mermaid
erDiagram
    users ||--o{ user_keys : has
    users ||--o{ bookmarks : creates
    
    users {
        uuid id PK
        text username
        text email UK
        text password_hash
        text did UK
        text role
        boolean is_suspended
    }
    
    user_keys {
        uuid id PK
        uuid user_id FK
        text public_key
        text key_type
    }
    
    remote_actors {
        uuid id PK
        text actor_uri UK
        text username
        text instance_domain
        text public_key
    }
```

### Social Relationships Domain

```mermaid
erDiagram
    follows {
        uuid id PK
        text follower_did
        text following_did
        text status
    }
```

### Content & Posts Domain

```mermaid
erDiagram
    posts ||--o{ media : has
    posts ||--o{ interactions : receives
    posts ||--o{ bookmarks : saved_in
    
    posts {
        uuid id PK
        text author_did
        text content
        text visibility
        timestamptz created_at
    }
    
    media {
        uuid id PK
        uuid post_id FK
        text media_url
        text media_type
    }
    
    interactions {
        uuid id PK
        uuid post_id FK
        text actor_did
        text interaction_type
    }
    
    bookmarks {
        uuid id PK
        uuid user_id FK
        uuid post_id FK
    }
```

### Messaging Domain

```mermaid
erDiagram
    message_threads ||--o{ messages : contains
    
    message_threads {
        uuid id PK
        uuid participant_a_id FK
        uuid participant_b_id FK
        timestamptz created_at
    }
    
    messages {
        uuid id PK
        uuid thread_id FK
        uuid sender_id FK
        uuid recipient_id FK
        text content
        boolean is_read
    }
```

### Federation Domain

```mermaid
erDiagram
    inbox_activities {
        uuid id PK
        text activity_id UK
        text actor_uri
        text activity_type
        jsonb payload
    }
    
    outbox_activities {
        uuid id PK
        text activity_type
        jsonb payload
        text target_inbox
        text status
    }
    
    activity_deduplication {
        text activity_id PK
        timestamptz processed_at
    }
```

### Moderation & Governance Domain

```mermaid
erDiagram
    users ||--o{ moderation_requests : requests
    posts ||--o{ reports : reported_in
    
    moderation_requests {
        uuid id PK
        uuid user_id FK
        text status
        uuid reviewed_by FK
    }
    
    reports {
        uuid id PK
        text reporter_did
        uuid post_id FK
        text status
    }
    
    admin_actions {
        uuid id PK
        text admin_id
        text action_type
        text target
    }
    
    blocked_domains {
        text domain PK
        text reason
    }
    
    instance_reputation {
        text domain PK
        int reputation_score
        int spam_count
    }
    
    federation_failures {
        text domain PK
        int failure_count
    }
```

## Key Relationships

- **Users** create and manage multiple **User Keys** for device authentication
- **Users** create **Bookmarks** for posts they want to save
- **Users** can request to become moderators via **Moderation Requests**
- **Users** participate in **Message Threads** and send/receive **Messages**
- **Posts** can have multiple **Media** attachments
- **Posts** receive **Interactions** (likes, reposts) from users
- **Posts** can be **Bookmarked** by users
- **Posts** can be **Reported** for moderation
- **Message Threads** contain multiple **Messages** between two participants

## Cardinality Legend

- `||--o{` : One-to-Many relationship
- `PK` : Primary Key
- `FK` : Foreign Key
- `UK` : Unique Key
