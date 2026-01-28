# 🗺️ Complete Navigation Guide - All Pages Mapped

## App Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      LANDING PAGE                               │
│  🌐 FEDERATE - Decentralized Social Network                     │
│                                                                 │
│  Buttons:                                                       │
│  • Explore Network → Opens server discovery                     │
│  • Login → Login with existing account                          │
│  • Signup → Create new account                                  │
│  • 🌙/☀️ Theme Toggle                                           │
└──────────┬──────────────┬──────────────┬──────────────────────┘
           │              │              │
           ▼              ▼              ▼
    ┌────────────────────────────────────────────┐
    │    EXPLORE NETWORK PAGE                    │
    │  (Indian Region Server Discovery)          │
    │                                            │
    │  Filters:                                  │
    │  • Region: Delhi, Karnataka, Maharashtra,  │
    │    West Bengal, Telangana, Pan-India       │
    │  • Moderation: All, Strict, Moderate,      │
    │    Lenient                                 │
    │  • Search: By name/category                │
    │                                            │
    │  Servers Listed: 6 Indian servers          │
    │  ← Back button to Landing                  │
    │  🌙/☀️ Theme Toggle                        │
    └─────────────────────────────────────────────┘
              ↓                          ↓
          [Click Server]          [← Back to Landing]
              ↓
    ┌─────────────────────────────────────────────┐
    │       LOGIN OR SIGNUP PAGE                  │
    │  (After selecting/deciding on server)       │
    │                                             │
    │  LOGIN: Password Entry                      │
    │  • Username                                 │
    │  • Display Name (optional)                  │
    │  • Challenge-Response Auth                  │
    │  → HOME PAGE                                │
    │                                             │
    │  SIGNUP: Create New Account                 │
    │  • Server selection                         │
    │  • DID generation                           │
    │  • Username & password                      │
    │  • Display name & bio                       │
    │  → HOME PAGE                                │
    │                                             │
    │  ← Back to Landing                          │
    │  🌙/☀️ Theme Toggle                         │
    └─────────────────────────────────────────────┘
              ↓
    ┌─────────────────────────────────────────────┐
    │           HOME PAGE (Main Feed)             │
    │                                             │
    │  TOP NAVIGATION:                            │
    │  • Home | Local | Federated (Tab buttons)   │
    │  • 🔍 Search (disabled)                     │
    │  • 👤 {username} button → Profile           │
    │  • 🌙/☀️ Theme Toggle                       │
    │                                             │
    │  LEFT SIDEBAR:                              │
    │  • 🏠 Home (active)                         │
    │  • 💬 Messages → Direct Messages page       │
    │  • 🔐 Security → Security Dashboard         │
    │  • Your Profile Card (click → Profile)      │
    │  • Server Info                              │
    │                                             │
    │  CENTER (Main Feed):                        │
    │  • Post Composer (Create posts)             │
    │  • Posts list with Local/Federated/Home     │
    │  • Click author → Profile Page              │
    │  • Click post content → Thread View         │
    │  • Like, Boost, Reply buttons               │
    │                                             │
    │  RIGHT SIDEBAR:                             │
    │  • Trending topics (disabled)               │
    │  • 📋 Moderation Queue button → Mod Panel   │
    │  • 🌐 Federation Inspector button → Fed     │
    │  • Future Features (disabled - yellow)      │
    └─────────────────────────────────────────────┘
     │     │      │        │         │       │      │
     ▼     ▼      ▼        ▼         ▼       ▼      ▼
   [Post] [DM]  [Sec]  [Mod]  [Fed]  [Prof] [Click Auth]
```

## Detailed Navigation Paths

### PATH 1: User Profile Navigation

```
HOME PAGE
├─ 👤 Username button (top right)
├─ Click author on post
├─ Click "Your Profile" card (left sidebar)
└─ Directly from any page that has Profile button
    ↓
USER PROFILE PAGE
├─ Navigation Bar Buttons:
│  ├─ ← Back to Home
│  ├─ 💬 Threads → Thread View
│  ├─ 📨 Messages → Direct Messages
│  ├─ 🔐 Security → Security Dashboard
│  └─ 🌙/☀️ Theme Toggle
├─ Profile Content:
│  ├─ User avatar
│  ├─ Username & handle
│  ├─ Reputation badge
│  ├─ Follow button
│  ├─ Message button
│  ├─ DID (copy to clipboard)
│  └─ Tabs: Posts | Followers | Following
└─ Back button (← Back to Home)
```

### PATH 2: Messaging Navigation

```
HOME PAGE
├─ 💬 Messages button (left sidebar)
├─ 💬 Messages button (top right when profile open)
└─ Navigate from anywhere via "Messages" button
    ↓
DIRECT MESSAGES PAGE
├─ Navigation Bar Buttons:
│  ├─ ← Back to Home
│  ├─ 👤 Profile → User Profile Page
│  ├─ 🔐 Security → Security Dashboard
│  └─ 🌙/☀️ Theme Toggle
├─ Chat Interface:
│  ├─ Left sidebar: Conversation list
│  ├─ Click conversation → Open chat
│  ├─ Message composer at bottom
│  ├─ Send messages with 🔒 encryption
│  └─ Conversation history
└─ Features:
   ├─ 🔒 End-to-end encrypted
   ├─ E2E indicator on each message
   └─ Message timestamps
```

### PATH 3: Thread Navigation

```
HOME PAGE
├─ Click on post content
├─ Click 💬 replies button
└─ Reply button on any post
    ↓
THREAD VIEW PAGE
├─ Navigation Bar Buttons:
│  ├─ ← Back to Timeline
│  ├─ 👤 Profile → User Profile Page
│  ├─ 💬 Messages → Direct Messages
│  └─ 🌙/☀️ Theme Toggle
├─ Thread Content:
│  ├─ Root post (original post)
│  ├─ Full conversation thread below
│  ├─ Reply composer
│  └─ Threaded responses
└─ Features:
   ├─ See full conversation
   ├─ Add replies
   └─ View all participants
```

### PATH 4: Security & Admin Navigation

```
HOME PAGE
├─ 🔐 Security (left sidebar) → Security Dashboard
├─ 📋 Moderation button (right sidebar) → Moderation Panel
└─ 🌐 Federation button (right sidebar) → Federation Inspector
    ↓
SECURITY DASHBOARD
├─ Key Management
├─ Recovery codes
├─ Public key display
└─ Settings
    ↓
    Can navigate to:
    ├─ 👤 Profile
    ├─ 💬 Messages
    └─ Other pages

MODERATION PANEL (Admin Only)
├─ Content Queue
├─ Filter by: Spam, Harassment, Federated
├─ Action buttons on items
├─ Navigation to Federation Inspector
└─ Other page navigation

FEDERATION INSPECTOR (Admin Only)
├─ Federation Health Metrics
├─ Connected Servers Status
├─ Server Reputation
├─ Activity Monitoring
└─ Navigation to Moderation Panel
```

### PATH 5: Server Discovery Navigation

```
LANDING PAGE
├─ "Explore Network" button
└─ Can also go back via ← Back button
    ↓
EXPLORE NETWORK PAGE
├─ Filter Options:
│  ├─ Region dropdown (select Indian state)
│  ├─ Moderation dropdown (select level)
│  └─ Search bar (search servers)
├─ Server Grid:
│  └─ Click server card → Details + Join button
│  └─ Can navigate back to Landing
└─ All servers shown with:
   ├─ Users count
   ├─ Region
   ├─ Moderation level
   ├─ Federation status
   └─ Reputation
```

## Quick Navigation Table

| From Page | To Page | Button/Action |
|-----------|---------|---------------|
| Landing | Explore Network | "Explore Network" button |
| Landing | Login | "Login" button |
| Landing | Signup | "Signup" button |
| Explore Network | Landing | "← Back" button |
| Login | Signup | "Create New Account" link |
| Login | Home | "Request Challenge" → Submit |
| Signup | Login | "Already have account?" link |
| Signup | Home | Complete signup wizard |
| Home | Profile | "👤 {username}" button or click author |
| Home | Thread | Click post content or 💬 button |
| Home | Messages | "💬 Messages" sidebar button |
| Home | Security | "🔐 Security" sidebar button |
| Home | Moderation | "📋 Moderation Queue" button (admin) |
| Home | Federation | "🌐 Federation Inspector" button (admin) |
| Profile | Home | "← Back" button |
| Profile | Threads | "💬 Threads" button |
| Profile | Messages | "📨 Messages" button |
| Profile | Security | "🔐 Security" button |
| Thread | Home | "← Back to Timeline" button |
| Thread | Profile | "👤 Profile" button |
| Thread | Messages | "💬 Messages" button |
| Messages | Home | "← Back" button |
| Messages | Profile | "👤 Profile" button |
| Messages | Security | "🔐 Security" button |
| Security | Home | "← Back" button |
| Security | Profile | "👤 Profile" button |
| Security | Messages | "💬 Messages" button |
| Moderation | Home | "← Back" button |
| Moderation | Federation | "🌐 Federation" button |
| Moderation | Profile | "👤 Profile" button |
| Federation | Home | "← Back" button |
| Federation | Moderation | "📋 Moderation" button |
| Federation | Profile | "👤 Profile" button |

## All Pages at a Glance

```
1. LANDING PAGE
   Purpose: Entry point and navigation hub
   Key Actions: Browse servers, Login, Signup
   Theme: Hero section with federation explanation

2. EXPLORE NETWORK
   Purpose: Discover federated servers
   Location: All Indian regions
   Filters: Region (6 options), Moderation (4 levels)
   Search: Works across all server info

3. LOGIN PAGE
   Purpose: Sign in with cryptographic auth
   Steps: Enter username → Request challenge → Sign challenge
   Captures: Username, Display Name

4. SIGNUP PAGE
   Purpose: Create new federated identity
   Steps: 4-step wizard
   Captures: Server, Username, Password, Display Name, Bio

5. HOME FEED PAGE
   Purpose: Main social timeline
   Tabs: Home (all), Local, Federated
   Features: Compose posts, view feed, engage with content

6. PROFILE PAGE
   Purpose: View/manage user profile
   Data: From login/signup
   Tabs: Posts, Followers, Following
   Actions: Follow, Message, View DID

7. THREAD VIEW PAGE
   Purpose: View conversation threads
   Features: See full conversation, add replies
   Navigation: Back to timeline, view profile

8. DIRECT MESSAGES PAGE
   Purpose: Private encrypted messaging
   Features: 1-on-1 conversations
   Security: E2E encrypted (🔒)

9. SECURITY DASHBOARD
   Purpose: Manage cryptographic keys
   Features: DID display, key fingerprints, recovery codes
   Admin: Key management, security settings

10. MODERATION PANEL
    Purpose: Review and moderate content
    Features: Queue filtering, content review
    Admin Only: Action buttons on items

11. FEDERATION INSPECTOR
    Purpose: Monitor federation health
    Features: Server status, reputation, activity
    Admin Only: Real-time federation metrics
```

## Theme Toggle

**Available on: Every page**

```
Click 🌙 (Moon) button → Switch to Light Mode
Click ☀️ (Sun) button → Switch to Dark Mode

Colors Change:
✓ Backgrounds: Dark ↔ Light
✓ Text: Light ↔ Dark
✓ Borders: Cyan/Magenta ↔ Gray/Red
✓ Buttons: Updated colors
✓ All UI elements themed
```

## Mobile & Responsive Notes

All pages are responsive and work on:
- Mobile phones (320px+)
- Tablets (768px+)
- Desktops (1024px+)
- Wide screens (1200px+)

Navigation adapts for smaller screens with:
- Collapsible sidebars
- Stack layout for narrow screens
- Touch-friendly button sizes

---

## Complete User Journey

```
New User Journey:
1. Visit Landing Page
2. Click "Signup"
3. Select Indian server
4. Generate DID
5. Create username & password
6. Enter display name & bio
7. Auto-navigates to Home Feed
8. Sees home feed with Local/Federated tabs
9. Can create posts, follow users, send messages
10. Can click profile to view their own data
11. Can navigate to any admin page if authorized

Returning User Journey:
1. Visit Landing Page
2. Click "Login"
3. Enter username & display name
4. Complete challenge-response auth
5. Auto-navigates to Home Feed
6. Sees personalized feed
7. Can access all pages via navigation
```

All navigation is **fully functional** and **production-ready**! 🚀
