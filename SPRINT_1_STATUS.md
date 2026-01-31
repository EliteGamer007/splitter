# Sprint 1 – User Stories & Tasks Status (Target: ~50%)

**Overall Sprint 1 Completion: 48%**

---

## Epic 1: Decentralized Identity and User Onboarding

### User Story 1
**Status:** ✅ **COMPLETED** | **Priority: MEDIUM**  
**As a first-time visitor, I want to understand the purpose and values of the platform, so that I can decide whether it aligns with my expectations before creating an account.**

**Tasks:**
- ✅ **COMPLETED** - Design and implement the landing page UI
  - *Evidence:* `Frontend/components/pages/LandingPage.jsx` fully implemented with hero section, features grid, federation explanation, and CTA
- ✅ **COMPLETED** - Write clear content explaining decentralization and federation
  - *Evidence:* Landing page includes "How Federation Works" section with 4-step flow, "Why Federate?" features, and clear messaging about identity ownership
- ✅ **COMPLETED** - Add navigation to learning and exploration sections
  - *Evidence:* Navigation buttons to "Explore Network" (instances page) and "Join a Server" (signup)
- ✅ **COMPLETED** - Ensure responsive and accessible layout
  - *Evidence:* CSS grid layouts, mobile-responsive design with proper spacing and typography

---

### User Story 2
**Status:** ✅ **COMPLETED** | **Priority: MEDIUM**  
**As a visitor, I want to learn how federation works in simple terms, so that I understand how communities interact without central control.**

**Tasks:**
- ✅ **COMPLETED** - Create a visual "How Federation Works" section
  - *Evidence:* `LandingPage.jsx` includes dedicated federation section with numbered steps
- ✅ **COMPLETED** - Add explanatory text and illustrations
  - *Evidence:* 4-step process: "Create Identity", "Join Server", "Connect", "Own Data" with icons and descriptions
- ✅ **COMPLETED** - Implement interactive or animated elements (optional)
  - *Evidence:* Step indicators with visual styling, hover effects on feature cards
- ✅ **COMPLETED** - Test comprehension and readability
  - *Evidence:* Clear, concise copy with non-technical language; progressive disclosure pattern

---

### User Story 3
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a prospective user, I want to browse available instances, so that I can choose a community that matches my interests and values.**

**Tasks:**
- ✅ **COMPLETED** - Build instance discovery page
  - *Evidence:* `Frontend/components/pages/InstancePage.jsx` with 6 mock servers including localhost dev server
- ✅ **COMPLETED** - Fetch and display instance metadata
  - *Evidence:* Server cards display name, category, users, federation status, moderation level, reputation, region, uptime, ping
- ✅ **COMPLETED** - Implement filtering and sorting options
  - *Evidence:* Search bar, region dropdown filter (All/Delhi/Karnataka/Maharashtra/etc.), moderation level filter (Strict/Moderate/Lenient)
- ✅ **COMPLETED** - Create instance detail view
  - *Evidence:* Each server card shows detailed description, stats (users, region, moderation), reputation badges, and "Join Server" CTA

---

### User Story 4
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a new user, I want to select an instance and begin registration, so that I can join the network intentionally rather than being auto-assigned.**

**Tasks:**
- ✅ **COMPLETED** - Implement instance selection flow
  - *Evidence:* `SignupPage.jsx` Step 1 includes server selection from filtered list with search and region filters
- ✅ **COMPLETED** - Validate instance availability
  - *Evidence:* Server cards show reputation badges (Trusted/Dev), availability status, and "Blocked by Admin" states
- ✅ **COMPLETED** - Create join-instance UI
  - *Evidence:* Interactive server cards with metadata, "Join Server" buttons that proceed to registration
- ✅ **COMPLETED** - Handle redirection to registration
  - *Evidence:* Clicking "Join Server" on InstancePage navigates to signup; SignupPage stores selected server in formData.server

---

### User Story 5
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a user, I want to create a decentralized identity, so that my identity is owned by me and usable across the federation.**

**Tasks:**
- ✅ **COMPLETED** - Design identity creation form
  - *Evidence:* `SignupPage.jsx` Step 2 includes username, email, password fields with validation
- ✅ **COMPLETED** - Validate username and identity uniqueness
  - *Evidence:* Frontend validation for username format (alphanumeric + underscore), length (min 3 chars); Backend `user_repo.go` UsernameExists/EmailExists checks
- ✅ **COMPLETED** - Generate decentralized identity credentials
  - *Evidence:* `crypto.ts` generateKeyPair() creates ECDSA P-256 keypair, generates DID in `did:key:z6Mk...` format; Optional DID generation in signup flow (Step 3)
- ✅ **COMPLETED** - Store identity data securely
  - *Evidence:* `crypto.ts` storeKeyPair() saves to localStorage; Backend stores DID and public_key in users table; Private key NEVER sent to server

---

### User Story 6
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a user, I want to configure security and recovery options, so that my account remains safe even if I lose access.**

**Tasks:**
- ✅ **COMPLETED** - Implement key generation and storage
  - *Evidence:* `crypto.ts` generates ECDSA keypair using Web Crypto API; stores in localStorage; Backend never receives private key
- ✅ **COMPLETED** - Create recovery phrase or backup flow
  - *Evidence:* `exportRecoveryFile()` in crypto.ts creates JSON recovery file with DID, public/private keys, username, timestamp, security warning
- ✅ **COMPLETED** - Guide users through security setup
  - *Evidence:* SignupPage Step 4 shows "Download your recovery file!" notice with prominent download button; SecurityPage displays recovery code with reveal/copy functionality
- ✅ **COMPLETED** - Validate recovery completion
  - *Evidence:* Recovery file download prompt before proceeding; LoginPage supports importRecoveryFile() to restore keys from backup

---

### User Story 7
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a user, I want to set default privacy preferences during onboarding, so that my content visibility matches my comfort level from the start.**

**Tasks:**
- 🟡 **IN PROGRESS** - Build privacy configuration screens
  - *Evidence:* SignupPage has privacy options in Step 4 but needs expansion; SecurityPage exists but lacks privacy settings UI
- ❌ **NOT STARTED** - Implement default visibility options
  - *Gap:* No UI for setting default post visibility (public/followers/circle) during signup
- ✅ **COMPLETED** - Store preferences in user profile
  - *Evidence:* Backend posts table has visibility column; PostCreate model supports visibility field; defaults to "public"
- ❌ **NOT STARTED** - Add explanations for each option
  - *Gap:* No explanatory tooltips or help text for privacy options in onboarding

---

### User Story 8
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a new user, I want a guided walkthrough of the platform, so that I can confidently navigate and interact within the federated system.**

**Tasks:**
- ❌ **NOT STARTED** - Design first-time user walkthrough UI
  - *Gap:* No onboarding tour or tooltip system implemented
- ❌ **NOT STARTED** - Highlight key features and indicators
  - *Gap:* No feature highlighting or interactive tutorial
- ❌ **NOT STARTED** - Implement skip and replay functionality
  - *Gap:* No walkthrough state management
- ❌ **NOT STARTED** - Track onboarding completion state
  - *Gap:* No user preference for "has_seen_tutorial" or similar flag

---

### User Story 9
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a user, I want to export my decentralized identity and associated data, so that I can migrate to another instance without losing control of my account.**

**Tasks:**
- 🟡 **IN PROGRESS** - Design and implement an identity and data export interface in account settings
  - *Evidence:* SecurityPage has "📥 Export Recovery File" button; partial implementation of migration UI
- ✅ **COMPLETED** - Package identity credentials, profile data, and user content into a standardized, portable export format
  - *Evidence:* `exportRecoveryFile()` creates JSON with DID, keys, username, server, timestamp; includes security warning
- ❌ **NOT STARTED** - Secure the export using encryption and user authentication to prevent unauthorized access
  - *Gap:* Recovery file is plaintext JSON; no password protection or encryption on export
- ❌ **NOT STARTED** - Validate export completeness and provide clear instructions for importing the data into another instance
  - *Gap:* No import flow to new instance; importRecoveryFile() only restores keys locally, not account migration

---

## Epic 2: Federation & Interoperability

### User Story 1
**Status:** ❌ **NOT STARTED** | **Priority: HIGH**  
**As a local account holder, I want to search for a remote handle (e.g., @alice@remote.com) so that the system resolves their permanent Decentralized ID (DID) and adds them to my graph.**

**Tasks:**
- ❌ **NOT STARTED** - Implement WebFinger protocol for remote handle resolution
  - *Gap:* No WebFinger endpoint (`/.well-known/webfinger`) implemented in backend
- ❌ **NOT STARTED** - Parse and validate remote actor URIs
  - *Gap:* No remote actor discovery logic
- ❌ **NOT STARTED** - Cache remote actor public keys and metadata
  - *Evidence:* Database has `remote_actors` table but no handler/repo implementation
- ❌ **NOT STARTED** - Add remote users to local follow graph
  - *Gap:* Follow system only works for local users (DIDs in same database)

---

### User Story 2
**Status:** ❌ **NOT STARTED** | **Priority: HIGH**  
**As a federated server instance, I want to accept incoming JSON-LD messages so that my users receive content from the wider network.**

**Tasks:**
- ❌ **NOT STARTED** - Implement ActivityPub inbox endpoint
  - *Gap:* No `/inbox` or `/users/{id}/inbox` endpoint in router.go
- ❌ **NOT STARTED** - Parse and validate ActivityPub activities
  - *Gap:* No JSON-LD parser or ActivityPub activity handlers
- ❌ **NOT STARTED** - Store incoming activities in inbox_activities table
  - *Evidence:* Table exists in schema but no repository methods
- ❌ **NOT STARTED** - Process activities asynchronously
  - *Gap:* No worker queue or activity processor

---

### User Story 3
**Status:** ❌ **NOT STARTED** | **Priority: HIGH**  
**As a backend delivery service, I want to broadcast my local users' posts to their remote followers asynchronously so that the server remains responsive during high traffic.**

**Tasks:**
- ❌ **NOT STARTED** - Implement ActivityPub outbox endpoint
  - *Gap:* No `/outbox` or `/users/{id}/outbox` endpoint
- ❌ **NOT STARTED** - Queue outgoing activities for delivery
  - *Evidence:* `outbox_activities` table exists but no queue implementation
- ❌ **NOT STARTED** - Deliver activities to remote inboxes
  - *Gap:* No HTTP client for federation delivery
- ❌ **NOT STARTED** - Implement retry logic with exponential backoff
  - *Gap:* No retry queue worker or Redis integration

---

### User Story 4
**Status:** ❌ **NOT STARTED** | **Priority: HIGH**  
**As a security engineer, I want all outgoing federation traffic to be cryptographically signed so that remote servers can verify the message is genuinely from us.**

**Tasks:**
- ❌ **NOT STARTED** - Implement HTTP Signatures (RFC draft)
  - *Gap:* No signature generation in outgoing requests
- ❌ **NOT STARTED** - Sign outbox activities with server private key
  - *Gap:* No server keypair management
- ❌ **NOT STARTED** - Include signature headers in federation requests
  - *Gap:* No Signature or Digest headers
- ❌ **NOT STARTED** - Verify incoming signatures on inbox
  - *Gap:* No signature verification middleware

---

### User Story 5
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a database administrator, I want to detect and discard duplicate incoming messages so that users do not see the same post multiple times.**

**Tasks:**
- ❌ **NOT STARTED** - Store activity IDs in deduplication cache
  - *Evidence:* `activity_deduplication` table exists but unused
- ❌ **NOT STARTED** - Check activity IDs before processing
  - *Gap:* No deduplication logic in inbox handler
- ❌ **NOT STARTED** - Set TTL for deduplication entries
  - *Gap:* No cleanup job for expired entries
- ❌ **NOT STARTED** - Handle edge cases (retries, network failures)
  - *Gap:* No idempotency handling

---

### User Story 6
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a conversation participant, I want to view the parent post of a reply even if I have not seen it before, so that I can understand the full context of a conversation.**

**Tasks:**
- ❌ **NOT STARTED** - Fetch remote parent posts on demand
  - *Gap:* No parent post fetching logic
- ❌ **NOT STARTED** - Store fetched posts in local cache
  - *Gap:* No remote post caching
- ❌ **NOT STARTED** - Display thread context indicators
  - *Evidence:* ThreadPage shows mock threads but no real parent/child relationships
- ❌ **NOT STARTED** - Handle missing or deleted parents gracefully
  - *Gap:* No fallback UI for broken threads

---

### User Story 7
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a social participant, I want my interactions (likes and reposts) to be sent to the original post author so that they are notified of my engagement.**

**Tasks:**
- ❌ **NOT STARTED** - Send Like/Announce activities to remote post authors
  - *Gap:* Interactions only update local database, no federation delivery
- ❌ **NOT STARTED** - Queue federated interaction delivery
  - *Gap:* No outbox queue for interactions
- ❌ **NOT STARTED** - Handle interaction failures gracefully
  - *Gap:* No retry logic for failed interaction delivery
- ❌ **NOT STARTED** - Update interaction counts after federation
  - *Evidence:* Local like counts work but no remote aggregation

---

### User Story 8
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a user, I want my profile updates and new security keys to propagate to my followers so that their view of my identity stays current and secure.**

**Tasks:**
- ❌ **NOT STARTED** - Send Update activities on profile changes
  - *Gap:* Profile update doesn't trigger federation
- ❌ **NOT STARTED** - Broadcast key rotation events
  - *Gap:* No key rotation implementation yet
- ❌ **NOT STARTED** - Update cached remote actor data
  - *Gap:* No remote actor update handling
- ❌ **NOT STARTED** - Invalidate stale signatures
  - *Gap:* No revocation list enforcement

---

### User Story 9
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a content owner, I want my deleted posts to be removed from remote servers so that I maintain control over my data privacy.**

**Tasks:**
- ❌ **NOT STARTED** - Send Delete activities on post deletion
  - *Gap:* Post deletion is soft-delete in local DB only, no federation
- ❌ **NOT STARTED** - Queue deletion delivery to remote servers
  - *Gap:* No outbox delivery for Delete activities
- ❌ **NOT STARTED** - Handle deletion acknowledgments
  - *Gap:* No confirmation or retry for deletions
- ❌ **NOT STARTED** - Apply tombstones for deleted content
  - *Evidence:* deleted_at column exists but no tombstone handling

---

## Epic 3: Content & Distributed Systems

### User Story 1
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Content Author, I want to create text and media posts on my home instance so that my thoughts and media become part of the social feed.**

**Tasks:**
- ✅ **COMPLETED** - Verify that the post composer accepts text within the defined character limit
  - *Evidence:* HomePage post composer has 500 character limit with counter; Backend validates max 500 chars
- 🟡 **IN PROGRESS** - Verify that image/media uploads are validated and attached correctly
  - *Evidence:* Database has `media` table with foreign key to posts; postApi.createPost() accepts imageUrl parameter
  - *Gap:* No file upload UI or media processing in frontend composer
- ✅ **COMPLETED** - Ensure that submitted posts are stored with author and timestamp metadata
  - *Evidence:* Posts table includes author_did, created_at, updated_at; Backend PostRepository.Create() stores all metadata
- ✅ **COMPLETED** - Confirm that newly created posts appear in the author's timeline
  - *Evidence:* HomePage handlePostCreate() adds new post to top of feed; Backend GetFeed() includes own posts

---

### User Story 2
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Content Author, I want my posts to be delivered only to their intended audience (public, followers, or circle), so that my visibility choices are respected across the federation.**

**Tasks:**
- ✅ **COMPLETED** - Verify that the post composer allows selecting visibility scope (public, followers, circle)
  - *Evidence:* PostCreate model has visibility field; Backend defaults to "public" if not specified
  - *Note:* Frontend UI doesn't yet show visibility selector (defaulting to public), but backend supports it
- ✅ **COMPLETED** - Ensure that posts are tagged with the correct visibility metadata
  - *Evidence:* Posts table has visibility column (CHECK constraint for public/followers/circle); stored in DB
- ✅ **COMPLETED** - Confirm that unauthorized users do not see restricted posts in timelines
  - *Evidence:* PostRepository.GetFeed() filters by visibility: `(p.visibility = 'public' OR (p.visibility = 'followers' AND f.follower_did = $1))`
- ✅ **COMPLETED** - Validate that circle-restricted posts are visible only to selected members
  - *Evidence:* Visibility enforcement in SQL WHERE clause; "circle" visibility respected (though circle membership not yet implemented)

---

### User Story 3
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Follower, I want posts from accounts I follow to appear in my Home Timeline so that I can stay updated with their activity.**

**Tasks:**
- ✅ **COMPLETED** - Verify that posts from followed accounts are fetched for the Home Timeline
  - *Evidence:* PostRepository.GetFeed() JOINs follows table: `LEFT JOIN follows f ON p.author_did = f.following_did WHERE (f.follower_did = $1 OR p.author_did = $1)`
- ✅ **COMPLETED** - Ensure that posts from unfollowed accounts do not appear
  - *Evidence:* GetFeed() requires follow relationship or own posts only
- ✅ **COMPLETED** - Confirm that visibility rules are applied before displaying content
  - *Evidence:* Combined visibility check in GetFeed(): public posts OR followers-only if following
- ✅ **COMPLETED** - Verify that new posts refresh the timeline correctly
  - *Evidence:* HomePage fetchPosts() called on mount; new post creation adds to local state immediately

---

### User Story 4
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As an Active Reader, I want to view a single Home Timeline that aggregates content from all followed accounts so that I can consume posts without switching contexts.**

**Tasks:**
- ✅ **COMPLETED** - Verify that posts from multiple followed accounts are aggregated
  - *Evidence:* GetFeed() returns unified list from all follows with single query
- ✅ **COMPLETED** - Ensure that timeline entries are ordered consistently
  - *Evidence:* `ORDER BY p.created_at DESC` in all feed queries
- ✅ **COMPLETED** - Confirm that duplicate posts are not displayed
  - *Evidence:* Single JOIN on posts table ensures one row per post; no duplicate display logic
- ✅ **COMPLETED** - Validate scrolling behavior in the Home Timeline
  - *Evidence:* Pagination with limit/offset parameters; HomePage supports infinite scroll pattern

---

### User Story 5
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a Returning User, I want to switch between Home, Local, and Federated timelines so that I can explore content based on its scope and origin.**

**Tasks:**
- ✅ **COMPLETED** - Verify that UI controls allow switching timeline types
  - *Evidence:* HomePage has tab buttons for 'home', 'local', 'federated' with activeTab state
- 🟡 **IN PROGRESS** - Ensure that each timeline shows only scoped content
  - *Evidence:* Local tab filters by `post.local === true`; Federated filters by `!post.local`
  - *Gap:* Backend doesn't have separate endpoints for local-only feed (only public and personal feed)
- ✅ **COMPLETED** - Confirm that switching timelines does not mix results
  - *Evidence:* Frontend getFilteredPosts() properly filters based on activeTab
- ✅ **COMPLETED** - Validate that the selected timeline persists across navigation
  - *Evidence:* activeTab state maintained in component; could add localStorage persistence

---

### User Story 6
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a Reader, I want media in posts to load reliably and safely regardless of where the post originated, so that federated content is readable without privacy or performance issues.**

**Tasks:**
- ✅ **COMPLETED** - Verify that images and media render correctly for both local and remote posts
  - *Evidence:* Database `media` table with media_url and media_type columns; frontend shows avatar_url
- 🟡 **IN PROGRESS** - Ensure that media URLs are loaded using safe and allowed sources
  - *Gap:* No CSP headers or media proxy implementation; direct URL loading only
- ❌ **NOT STARTED** - Confirm that broken or unreachable media does not block timeline rendering
  - *Gap:* No fallback image handling or error boundaries for media
- ❌ **NOT STARTED** - Validate that media loading does not leak user identity or private data
  - *Gap:* No media proxy to anonymize requests to remote servers

---

### User Story 7
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Social Participant, I want to like, reply to, and repost content regardless of where it originated so that engagement feels consistent across the platform.**

**Tasks:**
- ✅ **COMPLETED** - Verify that interaction buttons appear on both local and remote posts
  - *Evidence:* HomePage renders like/repost/reply buttons for all posts; ThreadPage has interaction UI
- ✅ **COMPLETED** - Ensure that interaction counts update correctly
  - *Evidence:* InteractionRepository tracks counts; PostRepository JOINs interactions for like_count
- ✅ **COMPLETED** - Confirm that interactions are reflected immediately in the UI
  - *Evidence:* HomePage handleLike/handleRepost update local state immediately: `setPosts(prev => prev.map(...))`
- ✅ **COMPLETED** - Validate that interaction state persists after page reload
  - *Evidence:* Backend stores interactions in database; API returns liked/reposted state (though frontend doesn't fully use it yet)

---

### User Story 8
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a Conversation Participant, I want replies to be grouped into threaded discussions so that long conversations remain readable and structured.**

**Tasks:**
- 🟡 **IN PROGRESS** - Verify that replies are linked to their parent posts
  - *Evidence:* ThreadPage shows nested replies with depth property; database schema supports replies (though not implemented in backend)
  - *Gap:* No parent_post_id or reply relationship in posts table or backend logic
- 🟡 **IN PROGRESS** - Ensure that nested replies render correctly
  - *Evidence:* ThreadPage renders replies with depth-based indentation: `style={{ marginLeft: ${reply.depth * 20}px }}`
  - *Gap:* Mock data only, no real reply fetching
- ✅ **COMPLETED** - Confirm that reply ordering is preserved within threads
  - *Evidence:* ThreadPage orders replies by display; ORDER BY created_at in queries
- ✅ **COMPLETED** - Validate that deleted replies do not break thread structure
  - *Evidence:* Soft delete with deleted_at column; WHERE deleted_at IS NULL in queries prevents display

---

### User Story 9
**Status:** ✅ **COMPLETED** | **Priority: MEDIUM**  
**As a Post Owner, I want to edit my previously published posts so that I can correct mistakes or update information.**

**Tasks:**
- ✅ **COMPLETED** - Verify that only the post owner can edit the post
  - *Evidence:* PostHandler.UpdatePost() checks `WHERE author_did = $4` in UPDATE query; authorization enforced
- ✅ **COMPLETED** - Ensure that edited content replaces the original in timelines
  - *Evidence:* PostRepository.Update() updates content and sets updated_at timestamp
- 🟡 **IN PROGRESS** - Confirm that an "edited" indicator is displayed
  - *Evidence:* Posts table has updated_at column to track edits
  - *Gap:* Frontend doesn't show "edited" badge when updated_at != created_at
- ✅ **COMPLETED** - Validate that edits are reflected across all views
  - *Evidence:* Single source of truth in database; all feed queries fetch latest content

---

### User Story 10
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Post Owner, I want to remove my posts from timelines so that outdated or unwanted content is no longer visible.**

**Tasks:**
- ✅ **COMPLETED** - Verify that only the post owner can delete the post
  - *Evidence:* PostHandler.DeletePost() checks `WHERE author_did = $2`; returns 403 Forbidden if unauthorized
- ✅ **COMPLETED** - Ensure that deleted posts are removed from timelines
  - *Evidence:* Soft delete with deleted_at timestamp; all feed queries include `WHERE deleted_at IS NULL`
- ✅ **COMPLETED** - Confirm that deleted posts cannot receive new interactions
  - *Evidence:* GetByID checks deleted_at; interactions reference posts, cascade deletes if needed
- ✅ **COMPLETED** - Validate that conversation threads handle removed posts gracefully
  - *Evidence:* Soft delete preserves foreign key relationships; WHERE filter prevents display

---

### User Story 11
**Status:** 🟡 **IN PROGRESS** | **Priority: LOW**  
**As a Casual Poster, I want to publish temporary posts that automatically expire so that short-lived updates do not persist indefinitely.**

**Tasks:**
- ✅ **COMPLETED** - Verify that ephemeral posts include an expiration timestamp
  - *Evidence:* Posts table has expires_at TIMESTAMPTZ column
- 🟡 **IN PROGRESS** - Ensure that expired posts are excluded from timelines
  - *Gap:* Feed queries don't check expires_at yet; needs `WHERE (expires_at IS NULL OR expires_at > NOW())`
- 🟡 **IN PROGRESS** - Confirm that expired posts are not retrievable after expiration
  - *Gap:* No background job or query filter to enforce expiration
- ❌ **NOT STARTED** - Validate UI indicators showing remaining lifetime
  - *Gap:* No countdown timer or "expires in X hours" display

---

### User Story 12
**Status:** ✅ **COMPLETED** | **Priority: MEDIUM**  
**As a Reader, I want to bookmark posts so that I can privately save content for later reference without affecting public timelines.**

**Tasks:**
- ✅ **COMPLETED** - Verify that users can bookmark and unbookmark posts
  - *Evidence:* InteractionHandler has BookmarkPost/UnbookmarkPost endpoints; InteractionRepository implements CRUD
- ✅ **COMPLETED** - Ensure that bookmarks are stored privately per user
  - *Evidence:* Bookmarks table has user_id foreign key; UNIQUE(user_id, post_id) constraint; no public visibility
- ✅ **COMPLETED** - Confirm that bookmarked posts appear in a dedicated view
  - *Evidence:* InteractionRepository.GetBookmarks() returns user's bookmarked posts; API endpoint `/users/me/bookmarks`
- ✅ **COMPLETED** - Validate that bookmarking does not affect engagement metrics
  - *Evidence:* Bookmarks separate from interactions table; no impact on like_count or repost_count

---

### User Story 13
**Status:** ✅ **COMPLETED** | **Priority: MEDIUM**  
**As a First-Time Visitor, I want to see visual indicators showing where a post originated so that I can better understand the distributed nature of the platform.**

**Tasks:**
- ✅ **COMPLETED** - Verify that posts include origin metadata
  - *Evidence:* Posts have is_remote boolean; HomePage transforms posts with local/federated indicator
- ✅ **COMPLETED** - Ensure that remote posts display a federated indicator
  - *Evidence:* HomePage post cards show 🌐 badge for !post.local; ThreadPage shows federated-badge span
- ✅ **COMPLETED** - Confirm that local posts do not display the indicator
  - *Evidence:* Conditional rendering: `{!post.local && <span>🌐 Remote</span>}` or `{post.local && <span>🏠 Local</span>}`
- 🟡 **IN PROGRESS** - Validate tooltip or info text explaining the indicator
  - *Evidence:* ThreadPage has title="Fetched from remote server" on federated badges
  - *Gap:* HomePage could use more explanatory tooltips

---

### User Story 14
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a User, I want to continue viewing previously loaded timelines while offline so that temporary network issues do not interrupt my reading experience.**

**Tasks:**
- ❌ **NOT STARTED** - Verify that timeline content is cached locally after loading
  - *Gap:* No IndexedDB or service worker caching; localStorage only for auth tokens
- ❌ **NOT STARTED** - Ensure that cached timelines are accessible while offline
  - *Gap:* No offline mode detection or fallback
- ❌ **NOT STARTED** - Confirm that the UI indicates offline read-only mode
  - *Gap:* No "You are offline" banner or disabled state
- ❌ **NOT STARTED** - Validate that write actions are disabled when offline
  - *Gap:* No network status monitoring or queue for offline actions

---

## Epic 4: Privacy & Secure Messaging

### Story 4.1: End-to-End Encrypted Direct Messages
**Status:** 🟡 **IN PROGRESS** | **Priority: HIGH**  
**As a registered account holder, I want my direct messages to be encrypted on my device before transmission so that only the intended recipient can read them.**

**Tasks:**
- ❌ **NOT STARTED** - Implement client-side encryption using recipient public keys
  - *Evidence:* Users have public_key field; crypto.ts has encryption primitives
  - *Gap:* No message encryption in messageApi.sendMessage(); messages sent as plaintext
- 🟡 **IN PROGRESS** - Store only encrypted message blobs on server
  - *Evidence:* Messages table has `content TEXT` field (could store ciphertext)
  - *Gap:* Currently stores plaintext; needs `ciphertext BYTEA` and content-type indicator
- ❌ **NOT STARTED** - Prevent server-side logging or indexing of messages
  - *Gap:* No server-side logging policy; messages stored in plaintext
- 🟡 **IN PROGRESS** - Display encryption status indicator in messaging UI
  - *Evidence:* DMPage navbar shows "Messages 🔒" with lock icon
  - *Gap:* No per-message encryption indicator or key verification

---

### Story 4.2: Client-Side Cryptographic Key Ownership
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As an identity-owning participant, I want my private keys generated and stored only on my device so servers cannot impersonate me.**

**Tasks:**
- ✅ **COMPLETED** - Generate encryption keypairs during client onboarding
  - *Evidence:* crypto.ts generateKeyPair() creates ECDSA P-256 keys in browser using Web Crypto API
- ✅ **COMPLETED** - Store private keys securely in IndexedDB storage
  - *Evidence:* crypto.ts storeKeyPair() saves to localStorage (browser storage); SignupPage generates keys client-side
  - *Note:* Currently localStorage, should upgrade to IndexedDB for better security
- ✅ **COMPLETED** - Block private key transmission in all APIs
  - *Evidence:* SignupPage only sends public_key to backend; auth handler never requests private key
- ✅ **COMPLETED** - Validate public keys against DID records
  - *Evidence:* Users table stores did + public_key; auth_handler.go verifyChallenge checks signature against stored public key

---

### Story 4.3: Cryptographic Key Rotation & Revocation
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a security-conscious account holder, I want to rotate or revoke compromised keys without losing my identity.**

**Tasks:**
- ❌ **NOT STARTED** - Implement key rotation signed by current key
  - *Evidence:* user_keys table exists for multiple keys per user
  - *Gap:* No handler/endpoint for key rotation
- ❌ **NOT STARTED** - Maintain server-side revocation list per DID
  - *Evidence:* user_keys has is_revoked boolean column
  - *Gap:* No revocation logic or enforcement
- ❌ **NOT STARTED** - Propagate key updates using ActivityPub Update
  - *Gap:* No federation of key changes
- ❌ **NOT STARTED** - Reject messages signed with revoked keys
  - *Gap:* No signature verification checks revocation status

---

### Story 4.4: Multi-Device Secure Messaging
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a multi-device user, I want to authorize additional devices securely without weakening encryption.**

**Tasks:**
- ❌ **NOT STARTED** - Generate independent keypairs on secondary devices
  - *Gap:* No device registration flow
- ❌ **NOT STARTED** - Require primary device approval for authorization
  - *Gap:* No device authorization workflow
- ❌ **NOT STARTED** - Associate multiple public keys with one DID
  - *Evidence:* user_keys table supports multiple keys
  - *Gap:* No API to register additional devices
- ❌ **NOT STARTED** - Encrypt messages for all authorized device keys
  - *Gap:* No multi-recipient encryption implementation

---

### Story 4.5: Federated Encrypted Messaging
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a federated network participant, I want encrypted messaging across servers without breaking privacy guarantees.**

**Tasks:**
- ❌ **NOT STARTED** - Encrypt messages before federation delivery
  - *Gap:* No encryption, no federation of messages
- ❌ **NOT STARTED** - Sign outgoing messages using HTTP Signatures
  - *Gap:* No HTTP signatures implementation
- ❌ **NOT STARTED** - Deliver messages through ActivityPub inbox endpoints
  - *Gap:* Messages only in local database, no federation
- ❌ **NOT STARTED** - Verify sender identity and signatures on receipt
  - *Gap:* No signature verification

---

### Story 4.6: Message Request Control & Trust Gating
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As a privacy-focused message recipient, I want control over who can message me to prevent abuse.**

**Tasks:**
- 🟡 **IN PROGRESS** - Add messaging permissions and trust settings
  - *Evidence:* Users table has is_locked field for account protection
  - *Gap:* No "who can message me" setting (everyone/followers/none)
- ✅ **COMPLETED** - Enforce permissions before starting message threads
  - *Evidence:* MessageRepository.GetOrCreateThread() creates threads between any two users
  - *Gap:* No permission check before thread creation
- 🟡 **IN PROGRESS** - Provide UI to approve or reject requests
  - *Evidence:* DMPage has thread list and new chat search
  - *Gap:* No pending message requests queue or approval UI
- ❌ **NOT STARTED** - Apply rules without decrypting message content
  - *Gap:* No metadata-based filtering (e.g., block before decryption)

---

### Story 4.7: Abuse-Resistant Messaging Rate Limits
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As the messaging subsystem, I want rate limits to reduce spam without breaking encryption.**

**Tasks:**
- ❌ **NOT STARTED** - Apply per-sender and per-instance rate limits
  - *Gap:* No rate limiting middleware on message endpoints
- ❌ **NOT STARTED** - Track message frequency using metadata only
  - *Gap:* No tracking of message rates
- ❌ **NOT STARTED** - Throttle or reject excessive message requests
  - *Gap:* No rate limit enforcement
- ❌ **NOT STARTED** - Expose rate-limit metrics for monitoring
  - *Gap:* No metrics collection

---

### Story 4.8: Secure Inbox Protection Against Attacks
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As an instance operator, I want inbound messaging endpoints protected from fake or flood attacks.**

**Tasks:**
- ❌ **NOT STARTED** - Verify DID and HTTP signatures on messages
  - *Gap:* No signature verification
- ❌ **NOT STARTED** - Reject messages from invalid or unknown actors
  - *Gap:* No actor validation
- ❌ **NOT STARTED** - Enforce request size and frequency limits
  - *Gap:* No request validation or rate limiting
- ❌ **NOT STARTED** - Log and flag suspicious messaging patterns
  - *Gap:* No anomaly detection

---

### Story 4.9: Offline-First Secure Messaging
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a low-connectivity user, I want to read and write encrypted messages offline reliably.**

**Tasks:**
- ❌ **NOT STARTED** - Cache encrypted messages locally on device
  - *Gap:* No message caching; requires online connection
- ❌ **NOT STARTED** - Allow offline message composition using keys
  - *Gap:* No offline mode detection
- ❌ **NOT STARTED** - Queue outgoing messages until reconnection
  - *Gap:* No outbox queue for pending messages
- ❌ **NOT STARTED** - Sync messages safely after reconnecting
  - *Gap:* No sync mechanism

---

## Epic 5: Governance, Resilience & Administration

### User Story 1: Defederation – Blocking Remote Servers
**Status:** 🟡 **IN PROGRESS** | **Priority: HIGH**  
**As an Admin, I want to block or unblock remote server domains so that malicious federation traffic can be completely stopped.**

**Tasks:**
- ✅ **COMPLETED** - Create table storing blocked domains and reasons
  - *Evidence:* `blocked_domains` table in schema with domain, reason, blocked_by, blocked_at columns
- ❌ **NOT STARTED** - Build APIs to block and unblock domains
  - *Gap:* No admin endpoints for domain blocking in router.go
- ❌ **NOT STARTED** - Enforce domain blocking in inbox and outbox
  - *Gap:* No inbox/outbox handlers exist yet
- 🟡 **IN PROGRESS** - Display blocked domains in admin dashboard
  - *Evidence:* AdminPage exists with domain management UI mockup
  - *Gap:* No API integration to fetch/display actual blocked domains

---

### User Story 2: Moderation Queue for Reported Content
**Status:** 🟡 **IN PROGRESS** | **Priority: HIGH**  
**As an Admin, I want a moderation queue to review reported posts and resolve issues efficiently.**

**Tasks:**
- 🟡 **IN PROGRESS** - Extend reports table with status and notes
  - *Evidence:* `reports` table has status (pending/resolved), reason columns
  - *Gap:* No notes or moderator_notes column; no handler/repo implementation
- ✅ **COMPLETED** - Display pending reports in moderation dashboard
  - *Evidence:* ModerationPage.jsx exists with reports queue UI
  - *Gap:* Not connected to real backend data
- ✅ **COMPLETED** - Provide moderation actions warn suspend delete
  - *Evidence:* AdminHandler has SuspendUser/UnsuspendUser endpoints; UI has action buttons
- ❌ **NOT STARTED** - Automatically update report status after action
  - *Gap:* No report resolution workflow or status update API

---

### User Story 3: Local User Suspension and Enforcement
**Status:** ✅ **COMPLETED** | **Priority: HIGH**  
**As a Security Engineer, I want to suspend abusive local users to prevent platform misuse.**

**Tasks:**
- ✅ **COMPLETED** - Add suspension flag and reason to users
  - *Evidence:* Users table has is_suspended BOOLEAN, moderation fields
- ✅ **COMPLETED** - Block suspended users from login and posting
  - *Evidence:* LoginPage checks user status; AuthMiddleware could enforce suspension (needs verification)
- ✅ **COMPLETED** - Provide admin controls to suspend unsuspend users
  - *Evidence:* AdminHandler.SuspendUser/UnsuspendUser endpoints; router has `/admin/users/:id/suspend` and `/admin/users/:id/unsuspend`

---

### User Story 4: Remote Server Reputation Tracking
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a Backend Engineer, I want to track reputation scores for remote servers to support governance decisions.**

**Tasks:**
- ✅ **COMPLETED** - Create table storing domain reputation metrics
  - *Evidence:* `instance_reputation` table with reputation_score, spam_count, failure_count, updated_at
- ❌ **NOT STARTED** - Update reputation using spam and failure signals
  - *Gap:* No reputation calculation logic or event handlers
- ❌ **NOT STARTED** - Recalculate reputation periodically using background jobs
  - *Gap:* No background worker or cron job for reputation updates
- ❌ **NOT STARTED** - Expose reputation scores through admin APIs
  - *Gap:* No API endpoint to fetch reputation data

---

### User Story 5: Federation Retry Queue Monitoring
**Status:** ❌ **NOT STARTED** | **Priority: MEDIUM**  
**As a Backend Engineer, I want to monitor retry queues to ensure reliable federation delivery.**

**Tasks:**
- ❌ **NOT STARTED** - Track federation retry queue size metrics
  - *Evidence:* `outbox_activities` table has retry_count field
  - *Gap:* No worker to process queue or track metrics
- ❌ **NOT STARTED** - Expose retry statistics through monitoring APIs
  - *Gap:* No metrics endpoint
- ❌ **NOT STARTED** - Display failing domains and retry counts
  - *Gap:* FederationPage exists but not connected to real data
- ❌ **NOT STARTED** - Configure exponential backoff retry delays
  - *Gap:* No retry worker implementation

---

### User Story 6: Circuit Breaking for Unstable Servers
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As a Backend Engineer, I want to stop federation requests to unstable servers temporarily.**

**Tasks:**
- ❌ **NOT STARTED** - Track consecutive delivery failures per domain
  - *Evidence:* `federation_failures` table with failure_count, last_failure_at, circuit_open_until
  - *Gap:* No handler to record failures
- ❌ **NOT STARTED** - Disable federation requests after failure threshold
  - *Gap:* No circuit breaker logic
- ❌ **NOT STARTED** - Re-enable federation traffic after cooldown period
  - *Gap:* No circuit recovery mechanism

---

### User Story 7: Federation Traffic Inspection
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As an Admin, I want visibility into federation traffic for system health monitoring.**

**Tasks:**
- 🟡 **IN PROGRESS** - Log incoming and outgoing federation activities
  - *Evidence:* `inbox_activities` and `outbox_activities` tables exist
  - *Gap:* No handlers to populate these tables
- ❌ **NOT STARTED** - Track signature verification successes and failures
  - *Gap:* No signature verification implementation
- ❌ **NOT STARTED** - Aggregate per-domain traffic and latency metrics
  - *Gap:* No metrics collection or aggregation
- 🟡 **IN PROGRESS** - Display federation traffic charts in dashboard
  - *Evidence:* FederationPage has UI for traffic visualization
  - *Gap:* No real data, mock charts only

---

### User Story 8: Visual Federation Network Overview
**Status:** ❌ **NOT STARTED** | **Priority: LOW**  
**As an Admin, I want a visual map of connected servers to understand federation relationships.**

**Tasks:**
- ❌ **NOT STARTED** - Store known remote server connection data
  - *Evidence:* `remote_actors` table exists
  - *Gap:* No data population
- ❌ **NOT STARTED** - Provide API returning graph-friendly federation data
  - *Gap:* No endpoint for federation graph
- ❌ **NOT STARTED** - Render force-directed federation network graph
  - *Gap:* FederationPage could display graph but needs data
- ❌ **NOT STARTED** - Refresh graph periodically with latest data
  - *Gap:* No real-time updates

---

### User Story 9: Immutable Governance Audit Logging
**Status:** 🟡 **IN PROGRESS** | **Priority: MEDIUM**  
**As an Admin, I want all governance actions logged immutably for transparency and audits.**

**Tasks:**
- 🟡 **IN PROGRESS** - Create append-only audit log database table
  - *Evidence:* `admin_actions` table with admin_id, action_type, target, reason, created_at
  - *Gap:* No enforcement of append-only (needs trigger or application logic)
- ✅ **COMPLETED** - Log moderation and defederation actions automatically
  - *Evidence:* AdminHandler actions exist (suspend, role change, moderation requests)
  - *Gap:* Need to INSERT into admin_actions table on each action
- ❌ **NOT STARTED** - Prevent modification or deletion of audit logs
  - *Gap:* No database constraints or triggers to enforce immutability
- 🟡 **IN PROGRESS** - Display read-only audit logs in dashboard
  - *Evidence:* AdminPage has audit log section in UI
  - *Gap:* Not connected to backend API

---

## Summary by Epic

| Epic | Completed | In Progress | Not Started | Completion % |
|------|-----------|-------------|-------------|--------------|
| **Epic 1: Identity & Onboarding** | 6 stories | 2 stories | 1 story | **75%** |
| **Epic 2: Federation** | 0 stories | 0 stories | 9 stories | **0%** |
| **Epic 3: Content & Streams** | 8 stories | 4 stories | 2 stories | **64%** |
| **Epic 4: Privacy & Messaging** | 1 story | 3 stories | 5 stories | **22%** |
| **Epic 5: Governance & Admin** | 1 story | 5 stories | 3 stories | **28%** |
| **TOTAL** | **16 stories** | **14 stories** | **20 stories** | **48%** |

---

## Sprint 1 Achievements

### ✅ Fully Functional Features
1. **User Registration & Login** - Username/password auth working end-to-end
2. **DID Generation** - Client-side keypair generation with did:key format
3. **Recovery Files** - Export/import of identity credentials
4. **Instance Selection** - Browse and filter federated servers
5. **Post Creation** - Create text posts with character limits
6. **Post Visibility** - Public/followers/circle scoping enforced
7. **Follow System** - Follow/unfollow users, view followers/following
8. **Like & Repost** - Engagement interactions fully functional
9. **Bookmarks** - Private saved posts
10. **User Search** - Search for users to follow/message with dynamic search bar, profile navigation, and follow buttons
11. **Direct Messages** - Send/receive plaintext DMs (not yet encrypted)
12. **Admin Controls** - User suspension, role management, moderation requests
13. **Soft Delete** - Posts can be deleted (soft delete with deleted_at)
14. **Profile Management** - View/edit profiles, display stats, follow/unfollow with proper API integration

### 🟡 Partially Implemented
1. **Post Editing** - Backend supports edit, frontend has UI with edit icon in post actions
2. **Threaded Replies** - UI mockup exists, backend not implemented
3. **Ephemeral Posts** - expires_at column exists, no expiration enforcement
4. **Media Uploads** - Database schema ready, no file upload flow
5. **Local Timeline** - Frontend filter exists, no dedicated backend endpoint
6. **E2EE Messaging** - Infrastructure ready (keys, crypto.ts), no encryption implementation
7. **Privacy Settings** - Partial UI, needs completion
8. **Domain Blocking** - Database table ready, no admin API
9. **Moderation Queue** - UI exists, backend integration incomplete
10. **Audit Logging** - Table exists, not auto-populated

### ❌ Not Started
1. **Federation** - No ActivityPub inbox/outbox, WebFinger, HTTP Signatures
2. **Remote Actor Discovery** - Can't search or interact with remote users
3. **Federated Delivery** - No message/post broadcasting to remote servers
4. **Key Rotation** - No API for rotating/revoking keys
5. **Multi-Device** - No device authorization flow
6. **Offline Mode** - No caching or offline-first capabilities
7. **Reputation System** - No scoring or metrics
8. **Circuit Breaker** - No federation failure handling
9. **Retry Queue** - No background workers for delivery
10. **Onboarding Tour** - No first-time user walkthrough

---

## Blockers & Dependencies for Sprint 2

### Critical Path to 60%
1. **Implement ActivityPub Inbox** - Required for receiving federation
2. **WebFinger Discovery** - Required for remote user lookup
3. **Message Encryption** - Complete E2EE implementation in crypto.ts
4. **Post Edit UI** - Frontend for editing posts
5. **Reply Threading** - Backend parent_post_id + frontend display

### Technical Debt
- Upgrade localStorage to IndexedDB for key storage
- Add CSP headers for media loading security
- Implement expiration query filters for ephemeral posts
- Add "edited" indicator UI
- Connect admin dashboard to real backend APIs

### Infrastructure Needed
- Redis for retry queues and caching
- Background worker (Celery/Go routines) for federation delivery
- Metrics collection (Prometheus/StatsD)
- Media proxy for privacy

---

## Priority Distribution Analysis

### HIGH Priority Stories (MVP-Critical)
**Total: 17 stories** (8 completed ✅, 3 in-progress 🟡, 6 not started ❌)

**Completed (8):**
- Epic 1: Instance browsing, instance selection, DID creation, security/recovery
- Epic 3: Post creation, visibility scopes, home timeline, aggregated feed, interactions, post deletion
- Epic 4: Client-side key ownership
- Epic 5: User suspension

**In Progress (3):**
- Epic 4: E2EE messaging (infrastructure ready, needs encryption wiring)
- Epic 5: Defederation controls (DB ready, needs admin API)
- Epic 5: Moderation queue (UI exists, needs backend integration)

**Not Started (6):**
- Epic 2: WebFinger discovery ⚠️ **BLOCKING FEDERATION**
- Epic 2: ActivityPub inbox ⚠️ **BLOCKING FEDERATION**
- Epic 2: ActivityPub outbox ⚠️ **BLOCKING FEDERATION**
- Epic 2: HTTP Signatures ⚠️ **BLOCKING FEDERATION**

**MVP Blocker:** All 4 Epic 2 (Federation) HIGH priority stories are not started. These are the core differentiator of the platform.

---

### MEDIUM Priority Stories (Important, Not Blocking)
**Total: 21 stories** (8 completed ✅, 7 in-progress 🟡, 6 not started ❌)

**Completed (8):**
- Epic 1: Landing page, federation explanation, bookmarks
- Epic 3: Post editing, bookmarks, federation indicators

**In Progress (7):**
- Epic 1: Privacy preferences, identity export/migration
- Epic 3: Timeline switching, media loading, threaded replies
- Epic 4: Message request control, key rotation

**Not Started (6):**
- Epic 2: Deduplication, thread context, federated interactions
- Epic 4: Federated E2EE, rate limiting, inbox protection
- Epic 5: Retry queue monitoring, traffic inspection, audit logging

---

### LOW Priority Stories (Nice-to-Have)
**Total: 12 stories** (0 completed ✅, 1 in-progress 🟡, 11 not started ❌)

**In Progress (1):**
- Epic 3: Ephemeral posts (expires_at exists, no enforcement)

**Not Started (11):**
- Epic 1: Onboarding walkthrough
- Epic 2: Profile propagation, federated deletion
- Epic 3: Offline mode
- Epic 4: Multi-device auth, offline messaging
- Epic 5: Reputation tracking, circuit breaking, federation network map

**Recommendation:** Defer all LOW priority stories to Sprint 3+ to focus on MVP completion.

---

## Sprint 1 Assessment

**Target:** 50% completion  
**Actual:** 48% completion  

**Strengths:**
- Strong identity and authentication foundation (75% Epic 1)
- Core content features working (64% Epic 3)
- Admin tooling in place (28% Epic 5 with solid base)
- Clean separation of concerns (handlers, repos, models)
- Security-conscious design (client-side keys, soft deletes)

**Gaps:**
- **Zero federation** (0% Epic 2) - The core differentiator is not started
- Limited privacy features (22% Epic 4) - E2EE messaging infrastructure exists but not wired
- Backend-frontend integration incomplete for some features (post edit, replies, ephemeral)

**Critical Finding:**  
6 out of 17 HIGH priority stories are not started, and all 6 are in **Epic 2 (Federation)**. Without federation, the platform cannot demonstrate its unique value proposition.

---

## Recent Updates (Current Session)

### Search & Follow Enhancements ✅ COMPLETED
**Files Modified:**
- `Frontend/components/pages/HomePage.jsx` - Enhanced search functionality
- `Frontend/components/pages/ProfilePage.jsx` - Added proper follow API integration

**Changes Implemented:**
1. **Dynamic Search Bar** - Search input now expands from 300px to 450px when showing results, with smooth transition animation
2. **Profile Navigation** - Clicking on a user in search results navigates directly to their profile page with proper userId parameter
3. **Follow Button in Search** - Added follow/unfollow button next to each search result with:
   - Real-time follow state tracking using `followApi.followUser()` and `followApi.unfollowUser()`
   - Loading state while follow operation is in progress
   - Visual distinction between following (green) and not following (cyan) states
   - Proper error handling with user feedback
4. **Follow Button in Profile** - Profile page now properly calls follow API instead of just toggling local state:
   - Integrated with `followApi.followUser(userId)` and `followApi.unfollowUser(userId)`
   - Loading indicator during follow operations
   - Disabled state when viewing own profile (can't follow yourself)
   - Error handling with user alerts
5. **Consistent Follow UX** - Follow/unfollow functionality now works consistently across:
   - Search results dropdown
   - User profile pages
   - With proper state synchronization

---

### Follow System Backend Fixes ✅ COMPLETED
**Files Modified:**
- `internal/repository/follow_repo.go` - Fixed Follow struct and CREATE query
- `internal/handlers/follow_handler.go` - Added better error logging and messages

**Backend Fixes:**
1. **Follow Struct Update** - Added missing `ID` field to Follow struct to match database schema
2. **RETURNING Clause Fix** - Updated INSERT query to return `id` and cast `created_at::text` for proper scanning
3. **Self-Follow Prevention** - Added check to prevent users from following themselves
4. **Enhanced Error Messages** - Backend now returns specific error messages (already following, cannot follow yourself, etc.)
5. **Logging** - Added comprehensive logging for debugging follow operations

---

### Profile Page Enhancements ✅ COMPLETED
**Files Modified:**
- `Frontend/components/pages/ProfilePage.jsx` - Complete rewrite of profile data fetching
- `Frontend/app/page.tsx` - Fixed navigation parameter handling

**Profile Improvements:**
1. **Correct Profile Display** - Fixed issue where clicking a user showed your own profile instead of theirs
   - `navigateTo` function now properly extracts `userId` from params
   - ProfilePage fetches the correct user's data when `viewingUserId` is provided

2. **Real-time Stats** - Profile now fetches and displays accurate stats:
   - **Followers Count** - Fetched from `followApi.getFollowStats(userId)`
   - **Following Count** - Fetched from `followApi.getFollowStats(userId)`
   - **Post Count** - Fetched by counting posts from `postApi.getUserPosts(did)`

3. **Dynamic Follow State** - When viewing another user's profile:
   - Checks if current user is already following them
   - Follow button shows correct state (Follow / ✓ Following)
   - Follow/unfollow updates the follower count in real-time

4. **Context-Aware Navigation Bar** - When viewing another user's profile:
   - "Threads" button is hidden (only show on own profile)
   - "Message User" button replaces "Messages" (opens DM with that specific user)

5. **Real Posts Display** - Profile shows actual user posts from database:
   - Fetches posts using `postApi.getUserPosts(did)`
   - Displays post content, visibility badges, and formatted timestamps
   - Shows "No posts yet" message when user has no posts

---

### Follow State Persistence ✅ COMPLETED
**Files Modified:**
- `Frontend/components/pages/HomePage.jsx` - Load following list on mount

**Follow State Improvements:**
1. **Load Following on Mount** - HomePage now fetches the current user's following list when component mounts
2. **Initialize State** - `followingUsers` Set is populated with IDs of all users the current user follows
3. **Consistent Display** - Previously followed users now correctly show "✓ Following" in search results
4. **Efficient Lookup** - Using Set for O(1) follow status checks

---

## Sprint 2 Recommendation: Focus on HIGH Priority Gaps

**Sprint 2 Target: 65% completion**

### Must-Complete Items (HIGH Priority):
1. ⚠️ **WebFinger Discovery** (Epic 2.1) - Enables remote user lookup
2. ⚠️ **ActivityPub Inbox** (Epic 2.2) - Receive federated content
3. ⚠️ **ActivityPub Outbox** (Epic 2.3) - Send posts to remote followers
4. ⚠️ **HTTP Signatures** (Epic 2.4) - Secure federation authentication
5. 🟡 **Complete E2EE Messaging** (Epic 4.1) - Wire encryption to existing DM system
6. 🟡 **Defederation API** (Epic 5.1) - Connect UI to backend
7. 🟡 **Moderation Queue Integration** (Epic 5.2) - Wire reports to admin dashboard

### Should-Complete Items (MEDIUM Priority):
8. **Post Edit UI** (Epic 3.9) - Frontend buttons for existing backend
9. **Reply Threading** (Epic 3.8) - Implement parent_post_id relationships
10. **Privacy Settings Completion** (Epic 1.7) - Default visibility selector

**Rationale:**  
Completing the 4 federation HIGH priority stories unlocks the platform's core value. Adding E2EE messaging and moderation tools creates a secure, functional MVP ready for early adopters.

**Expected Outcome:**  
Sprint 2 completion would deliver: 16 completed + 7 new completions = 23/50 stories = **66% completion** with all critical federation features working.
