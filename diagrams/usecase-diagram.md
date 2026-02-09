# Use Case Diagram

This diagram shows all the use cases in the Splitter federated social media platform, organized by Epic and showing the actors who interact with each use case.

```mermaid
flowchart LR
    subgraph Actors ["Actors"]
        direction TB
        U["(Human) Local User"]
        RI["(System) Remote Instance"]
        A["(Human) Instance Admin"]
        SE["(Human) Security Engineer"]
    end

    subgraph Splitter ["Splitter Platform"]
        direction TB
        
        subgraph Tier1 ["Identity & Federation"]
            direction TB
            subgraph Epic1["Epic 1: Identity & Autonomy"]
                direction TB
                UC_ID["Generate DID & Keys"]
                UC_Auth["Authenticate (Challenge)"]
                UC_ExpID["Export Identity"]
                UC_Rotate["Rotate Keys"]
                UC_Export["Export Data"]
            end

            subgraph Epic2["Epic 2: Federation"]
                direction TB
                UC_WF["Resolve DID (WebFinger)"]
                UC_Broad["Broadcast Activities"]
                UC_Verify["Verify HTTP Sigs"]
            end
        end

        Tier1 ~~~~ Tier2

        subgraph Tier2 ["Social & Governance"]
            direction TB
            %% Horizontal layout for Epic 3 & 4 to prevent overlap
            subgraph Epic34["Epic 3 & 4: Social & Messaging"]
                direction LR
                UC_Post["Create Post"]
                UC_E2EE["Send E2EE DM"]
                UC_Int["Interact (Like/Reply)"]
                UC_Time["View Timelines"]
                UC_Offline["Offline Reading"]
                UC_Report["Report Content"]
            end

            %% Space between Epic34 and Epic5
            Epic34 ~~~~ Epic5

            subgraph Epic5["Epic 5: Governance"]
                direction TB
                UC_Block["Block/Unblock Domain"]
                UC_Mod["Manage Moderation Queue"]
                UC_Rep["Monitor Reputation"]
                UC_Suspend["Suspend User"]
            end
        end
    end

    %% Actors to Splitter
    Actors ~~~~ Splitter

    %% User Connections
    U --> UC_ID
    U --> UC_Auth
    U --> UC_ExpID
    U --> UC_Rotate
    U --> UC_Export
    U --> UC_Post
    U --> UC_E2EE
    U --> UC_Int
    U --> UC_Time
    U --> UC_Offline
    U --> UC_Report

    %% Remote Instance Connections
    RI --> UC_Verify
    RI --> UC_WF

    %% Admin & Security Connections
    A --> UC_Block
    A --> UC_Mod
    A --> UC_Suspend
    SE --> UC_Rep

    %% Relationship Logic (Includes)
    UC_ID -.->|include| UC_Auth
    UC_Post -.->|include| UC_Broad
```

## Actors

### Local User (Human)
Primary user of the platform who:
- Creates and manages their DID-based identity
- Authenticates using challenge-response
- Creates posts and interacts with content
- Sends encrypted direct messages
- Views timelines and reads content offline
- Reports inappropriate content

### Remote Instance (System)
Federated server that:
- Resolves DIDs via WebFinger
- Verifies HTTP signatures on incoming activities

### Instance Admin (Human)
Administrator who:
- Blocks/unblocks remote domains (defederation)
- Manages the moderation queue
- Suspends problematic users

### Security Engineer (Human)
Technical role that:
- Monitors instance reputation scores
- Analyzes federation health

## Use Cases by Epic

### Epic 1: Identity & Autonomy
- **Generate DID & Keys**: Create decentralized identity in browser
- **Authenticate (Challenge)**: Sign challenge with private key
- **Export Identity**: Export identity for migration
- **Rotate Keys**: Update keypair for security
- **Export Data**: Download all user data (data portability)

### Epic 2: Federation
- **Resolve DID (WebFinger)**: Discover remote users
- **Broadcast Activities**: Send ActivityPub activities to followers
- **Verify HTTP Sigs**: Validate incoming federated requests

### Epic 3 & 4: Social & Messaging
- **Create Post**: Publish text/media content
- **Send E2EE DM**: Send end-to-end encrypted messages
- **Interact (Like/Reply)**: Engage with posts
- **View Timelines**: Browse Home/Local/Federated feeds
- **Offline Reading**: Access cached content without network
- **Report Content**: Flag inappropriate posts

### Epic 5: Governance
- **Block/Unblock Domain**: Manage federation blocklist
- **Manage Moderation Queue**: Review and resolve reports
- **Monitor Reputation**: Track instance health metrics
- **Suspend User**: Temporarily disable user accounts

## Relationships

### Include Relationships
- **Generate DID & Keys** includes **Authenticate**: Identity creation enables authentication
- **Create Post** includes **Broadcast Activities**: Posts are federated via ActivityPub

## Actor Responsibilities

| Actor | Primary Responsibilities |
|-------|-------------------------|
| **Local User** | Identity management, content creation, social interaction |
| **Remote Instance** | Federation protocol compliance, signature verification |
| **Instance Admin** | Moderation, governance, domain management |
| **Security Engineer** | Security monitoring, reputation analysis |
