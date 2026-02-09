# Splitter Diagrams

This directory contains comprehensive visual documentation for the Splitter federated social media platform using Mermaid diagrams.

## Available Diagrams

### 1. [Entity-Relationship Diagram](er-diagram.md)
Database schema visualization showing all 19 tables, relationships, and constraints organized by domain:
- Users & Identity
- Social Relationships
- Content & Posts
- Messaging
- Federation
- Moderation & Governance

### 2. [Sequence Diagram](sequence-diagram.md)
Complete interaction flows covering 9 major workflows:
- User signup with DID generation
- Challenge-response authentication
- Post creation with media
- Federation outbox/inbox delivery
- Timeline loading
- Post interactions
- End-to-end encrypted messaging
- Admin moderation and defederation

### 3. [Activity Diagram](activity-diagram.md)
User journey flowchart showing complete workflows from signup through all major features:
- Onboarding & authentication
- Timeline viewing
- Post creation & federation
- User interactions
- Search & discovery
- Direct messaging (E2EE)
- Identity management
- Content moderation

### 4. [Class Diagram](class-diagram.md)
Object-oriented design organized by Epic:
- **Epic 1**: Identity & Autonomy (User, IdentityKeys, AuthChallenge)
- **Epic 2**: Federation Infrastructure (ActivityPubObject, Inbox/Outbox, RetryQueue, FederationNode)
- **Epic 3 & 4**: Social & Messaging (Post, Interaction, EncryptedMessage, Timeline)
- **Epic 5**: Governance & Moderation (Report, AdminAction)

### 5. [Use Case Diagram](usecase-diagram.md)
Actors and their interactions with the platform:
- **Actors**: Local User, Remote Instance, Instance Admin, Security Engineer
- **Use Cases**: 20+ use cases organized by Epic
- **Relationships**: Include relationships between use cases

### 6. [Architecture Diagram](architecture-diagram.md)
5-layer system architecture with data flows:
- **Browser Layer**: Frontend UI, IndexedDB
- **API Layer**: Identity & Auth, Social Module, E2EE Extension
- **Data Layer**: PostgreSQL, Redis
- **Federation Layer**: ActivityPub Inbox/Outbox, Reputation System
- **External**: Remote federated instances

### 7. [State Diagram](state-diagram.md)
Entity lifecycles showing state transitions for:
- User lifecycle (Unregistered → Active → Moderator → Admin)
- Post lifecycle (Composing → Published → Edited/Deleted/Expired)
- Follow relationships
- Message lifecycle (E2EE flow)
- Federation activities (Outbound/Inbound)
- Moderation requests
- Content reports
- Session states

## Diagram Format

All diagrams use [Mermaid](https://mermaid.js.org/) syntax and will render automatically on GitHub, GitLab, and other platforms that support Mermaid.

## Viewing Diagrams

- **On GitHub**: Click any diagram file above - Mermaid renders automatically
- **Locally**: Use VS Code with the "Markdown Preview Mermaid Support" extension
- **Other viewers**: Most modern markdown viewers support Mermaid

## Related Documentation

- [Database Schema Documentation](../DATABASE_SCHEMA.md)
- [Coding Standards](../CODING_STANDARDS.md)
- [Troubleshooting Guide](../TROUBLESHOOTING.md)
- [Full Stack Documentation](../FULL_STACK_DOCUMENTATION.md)
