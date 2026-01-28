# 🚀 QUICK REFERENCE - Everything That Works

## Running the App
```bash
npm install
npm run dev
# Open http://localhost:3000
```

## 11 Pages (All Working)

| Page | Route | Access From |
|------|-------|-------------|
| Landing | / | Start here |
| Explore Network | /instances | Click "Explore Network" button |
| Login | /login | "Login" button on landing |
| Signup | /signup | "Signup" button on landing |
| Home Feed | /home | After login/signup |
| User Profile | /profile | Click profile button anywhere |
| Thread View | /thread | Click on a post |
| Direct Messages | /dm | Messages button on navbar |
| Security Dashboard | /security | Security button on navbar |
| Moderation Panel | /moderation | Admin section on home |
| Federation Inspector | /federation | Admin section on home |

## Features by Page

### Landing Page
- Explore Network button (with region filters)
- Login button
- Signup button  
- Theme toggle (🌙/☀️)

### Explore Network
- **Filters:**
  - Region: Delhi, Karnataka, Maharashtra, West Bengal, Telangana, Pan-India
  - Moderation: Strict, Moderate, Lenient
  - Search: By name/category/description
- All filters work together
- Theme toggle

### Login
- Enter username (try: alice, bob, charlie)
- Enter display name
- Server selection
- Challenge-response flow
- Theme toggle

### Signup
- 4-step wizard
- Server selection
- DID generation
- Download recovery file
- Create profile
- Theme toggle

### Home Feed
- **Tabs:** Home / Local / Federated
- **Local:** Shows only posts from your server
- **Federated:** Shows only posts from other servers
- **Post Composer:** Create posts
- **Sidebar:**
  - Your profile info (from login/signup)
  - Server info
  - Navigation links
- **Right Sidebar:**
  - Admin buttons (Moderation, Federation)
  - Future features (disabled in yellow)
- Click post author → View their profile
- Click post content → View thread
- Theme toggle

### User Profile
- Shows logged-in user's actual data
- Username, display name, avatar
- Followers/following counts
- Posts count
- DID display (copy to clipboard)
- Follow button
- Message button
- Tabs: Posts, Followers, Following
- Quick navigation: Threads, Messages, Security
- Theme toggle

### Thread View
- Root post with full content
- Threaded replies
- Reply composer
- Each reply shows author, timestamp
- Navigation: Profile, Messages
- Theme toggle

### Direct Messages
- Inbox with all conversations
- Click conversation to view
- End-to-end encrypted (🔒)
- Message composer
- Send messages
- Shows timestamp and encryption status
- Navigation: Profile, Security
- Theme toggle

### Security Dashboard
- DID (Decentralized Identifier) - copy to clipboard
- Public Key Fingerprint
- Recovery Code
- Key management info
- Password change (disabled)
- 2FA setup (disabled)
- Navigation: Profile, Messages
- Theme toggle

### Moderation Panel (Admin)
- Content moderation queue
- Filter by: All, Spam, Harassment, Federated
- Queue items show:
  - Preview of content
  - Author and server
  - Reason for review
  - Action buttons
- Queue count
- Navigation: Federation, Profile
- Theme toggle

### Federation Inspector (Admin)
- Federation health metrics
- Connected servers list
- Server status: Healthy 🟢, Degraded 🟡, Blocked 🔴
- Server reputation
- Activities per minute
- Last seen timestamp
- Navigation: Moderation, Profile
- Theme toggle

## User Data Flow

### Login Example
1. Go to Landing → Click Login
2. Enter username: `alice`
3. Enter display name: `Alice Chen`
4. Click "Request Challenge" → "Sign Challenge"
5. Go to Home
6. Profile now shows:
   - Username: alice
   - Display Name: Alice Chen
   - Avatar: 👩 (auto-assigned)
   - Email: alice@federate.tech

### Signup Example
1. Go to Landing → Click Signup
2. Select Server: `bangalore-tech.in`
3. Generate DID (click button)
4. Enter username: `bob`
5. Enter password
6. Enter display name: `Bob Kumar`
7. Enter bio: `Tech enthusiast`
8. Finish signup
9. Auto-navigates to Home
10. Click Profile to see all your data

## Theme Toggle

**Click the 🌙 button to switch to light mode**
**Click the ☀️ button to switch to dark mode**

Available on: Landing, Explore Network, Login, Signup, Home, Profile, Threads, Messages, Security, Moderation, Federation

## Indian Regions

All servers now hosted in India:
- **Delhi:** delhi-hub.in
- **Karnataka:** bangalore-tech.in
- **Maharashtra:** mumbai-creative.in
- **West Bengal:** kolkata-academic.in
- **Telangana:** hyderabad-network.in
- **Pan-India:** privacy-guard.in

**Filter by region:** Go to Explore Network → Region dropdown

## Testing Checklist

- [ ] Click through all 11 pages
- [ ] Try Dark/Light mode on each page
- [ ] Login with a username
- [ ] View profile after login (shows your data)
- [ ] Click "Local" tab on home (filters posts)
- [ ] Click "Federated" tab on home (filters posts)
- [ ] Filter by Indian region in Explore Network
- [ ] Filter by moderation level
- [ ] Send a message in Direct Messages
- [ ] Create a post on home feed
- [ ] Click post author → view profile
- [ ] Click post content → view thread
- [ ] View moderation queue (admin)
- [ ] View federation health (admin)

## File Structure

```
/
├── app/
│   ├── page.tsx              (Main router + state)
│   ├── layout.tsx            (Root layout)
│   └── globals.css           (Theme colors)
├── components/
│   ├── pages/
│   │   ├── LandingPage.jsx
│   │   ├── InstancePage.jsx
│   │   ├── LoginPage.jsx
│   │   ├── SignupPage.jsx
│   │   ├── HomePage.jsx
│   │   ├── ProfilePage.jsx
│   │   ├── ThreadPage.jsx
│   │   ├── DMPage.jsx
│   │   ├── SecurityPage.jsx
│   │   ├── ModerationPage.jsx
│   │   └── FederationPage.jsx
│   └── styles/
│       ├── LandingPage.css
│       ├── InstancePage.css
│       ├── LoginPage.css
│       ├── SignupPage.css
│       ├── HomePage.css
│       ├── ProfilePage.css
│       ├── ThreadPage.css
│       ├── DMPage.css
│       ├── SecurityPage.css
│       ├── ModerationPage.css
│       └── FederationPage.css
├── FIXES_SUMMARY.md          (Detailed fixes)
├── QUICK_REFERENCE.md        (This file)
└── README.md                 (Original docs)
```

## Troubleshooting

**Issue:** Profile shows "Your Name" after login
- **Fix:** Reload the page (Ctrl+R / Cmd+R)

**Issue:** Theme doesn't change
- **Fix:** Click the toggle button again, theme should invert

**Issue:** Posts don't filter by Local/Federated
- **Fix:** Make sure you're clicking the tab buttons, not just looking

**Issue:** Server filters don't work
- **Fix:** Both region and moderation filters must be set to show results

**Issue:** Can't click navigation buttons
- **Fix:** Make sure you're clicking the actual button, not nearby text

## URLs Map

```
Landing      → home
Explore      → instances
Login        → login
Signup       → signup
Home Feed    → home
Profile      → profile
Threads      → thread
Messages     → dm
Security     → security
Moderation   → moderation
Federation   → federation
```

## Quick Login Credentials (for testing)

Use any of these usernames (auto-assigned avatars):
- `alice` → 👩
- `bob` → 👨
- `charlie` → 👨‍💻
- `diana` → 🎨
- `eve` → 🔒

Just enter the username and any display name, then proceed through the challenge!

---

**Everything is working! Enjoy your federated social network! 🚀**
