# Federation Manifest (.well-known)

Splitter is a federated social network built on the ActivityPub protocol. This document details the implementation of federation-specific discovery and communication endpoints.

## Well-Known Endpoints

Splitter implements several standard discovery endpoints under the `/.well-known/` path.

### 1. WebFinger (`/.well-known/webfinger`)
Used to resolve user handles (e.g., `user@example.com`) to ActivityPub Actor URIs.
- **Parameters**: `resource` (e.g., `acct:alice@splitter.social`)
- **Returns**: JSON Resource Descriptor (JRD) with links to the ActivityPub actor.

### 2. NodeInfo (`/.well-known/nodeinfo`)
*(Planned)* Provides metadata about the instance, including version, protocols supported, and usage statistics.

## ActivityPub Implementation

Splitter utilizes a subset of the ActivityPub vocabulary for cross-instance interaction.

### Supported Actor Types
- `Person`: Standard user accounts.

### Supported Activity Types
| Activity | Description |
| --- | --- |
| `Create` | Used for new posts and stories. |
| `Follow` | Requesting to follow another user. |
| `Accept` | Acknowledging a follow request. |
| `Like` | Adding a like interaction to a post. |
| `Announce` | Reposting/Boosting a post. |
| `Undo` | Reversing a previous activity (Unlike, Unfollow). |
| `Delete` | Removing a post or story. |

### Technical Details
- **Signatures**: Splitter requires `Signature` headers on all inbound federation POST requests.
- **Digest**: Inbound requests must include a `Digest` header for the body payload.
- **Shared Inbox**: Splitter supports a `sharedInbox` to optimize delivery of activities to multiple recipients on the same instance.
