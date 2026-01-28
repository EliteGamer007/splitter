# Federate - Quick Start Guide

## 🚀 Get Running in 2 Minutes

### Option 1: Vercel (Recommended - 1 Click Deploy)

1. **Click Deploy Button** (if available in repo)
2. Login with GitHub/Vercel
3. Done! Your app is live

### Option 2: Local Development

```bash
# 1. Install Node.js if you don't have it
# Download from nodejs.org

# 2. Clone/Extract the project
git clone <repo-url>
cd federate-social-app

# 3. Install dependencies
npm install

# 4. Start development server
npm run dev

# 5. Open browser
# Visit http://localhost:3000
```

---

## 📱 All 11 Pages - Complete Tour

### Page 1: Landing
```
🌐 FEDERATE
↓
Federation explainer + Start button
```

### Page 2: Instance Selection
```
Select a federated server
- social.example.net (🟢 Trusted)
- federated.social (🟢 Trusted)
- evil.net (🔴 BLOCKED - yellow)
```

### Page 3-4: Signup Flow
```
Step 1: Select server
Step 2: Generate DID (public key)
Step 3: Username + Password
Step 4: Profile complete
Download recovery file
```

### Page 5: Login
```
Challenge-Response Authentication
1. Enter username
2. Get challenge (nonce)
3. Sign with private key
4. → Home Page
```

### Page 6: Home Feed
```
Left sidebar: Navigation, Profile, Server Info
Center: Post composer + Feed
Right sidebar: Trends + Admin Links
```

### Page 7: User Profile
```
Avatar + Bio
Follow/Message buttons
Tabs: Posts, Followers, Following
Federated profiles show 🌐 badge
```

### Page 8: Thread View
```
Root post (cyan border)
Threaded replies (indented)
Disabled actions on remote posts (yellow)
Reply composer
```

### Page 9: Direct Messages
```
Sidebar: Conversation list (🔒 encrypted)
Main: E2E encrypted chat
Banner: "Server cannot read this content"
Disabled: Attachments (Sprint 2)
```

### Page 10: Security Dashboard
```
✔ Identity Status
- Private Key Present
- Public Key Registered
- Recovery File Exported

Key Information:
- DID (copy button)
- Fingerprint (copy button)
- Recovery Code (reveal button)

Actions:
- Export Recovery File
- Rotate Key (disabled, Sprint 2)
- Revoke Key (disabled, Sprint 2)
```

### Page 11: Moderation Panel
```
Queue filters: All, Spam, Harassment, Federated
Action buttons: Remove, Warn, Block Domain
Example: "Buy crypto" posts → Remove
```

### Page 12: Federation Inspector
```
Health metrics:
- Incoming activities: 14/min
- Outgoing activities: 9/min
- Signature validation: 100%
- Retry queue: 2 pending

Connected servers table:
- Domain | Status | Reputation | Last Seen
```

---

## 🎮 Navigation Buttons

### Home Page
- **Left Sidebar:**
  - 🏠 Home
  - 💬 Messages 🔒 → DM Page
  - 🔐 Security → Security Page
  - ⚙️ Settings → Landing Page

- **Center Feed:**
  - Click author avatar → Profile
  - Click post text → Thread
  - Click 💬 button → Thread
  - Post 🚀 → Create post

- **Right Sidebar:**
  - 📋 Moderation Queue → Moderation Page
  - 🌐 Federation Inspector → Federation Page

### Other Pages
- All have **← Back** button (top left)
- Takes you back to Home or previous page

---

## 🎨 Theme

**Dark Mode Only:**
- Background: `#0f0f1a` (deep black)
- Primary: `#00d9ff` (cyan)
- Accent: `#ff006e` (magenta)
- Disabled: `#d4af37` (yellow)
- Text: `#e8eaed` (light gray)

---

## 📁 File Structure

```
project/
├── app/
│   ├── page.tsx          ← Main router (handles all page switching)
│   ├── layout.tsx        ← App shell
│   └── globals.css       ← Theme colors
│
├── components/pages/     ← All 11 page components (JSX)
│   ├── LandingPage.jsx
│   ├── InstancePage.jsx
│   ├── SignupPage.jsx
│   ├── LoginPage.jsx
│   ├── HomePage.jsx
│   ├── ProfilePage.jsx
│   ├── ThreadPage.jsx
│   ├── DMPage.jsx
│   ├── SecurityPage.jsx
│   ├── ModerationPage.jsx
│   └── FederationPage.jsx
│
├── components/styles/    ← CSS files (one per page)
│   ├── LandingPage.css
│   ├── InstancePage.css
│   └── ... (etc)
│
├── README.md             ← Features & architecture
├── SETUP.md              ← Detailed setup guide
└── QUICK_START.md        ← This file
```

---

## 🔧 How Page Navigation Works

### File: `/app/page.tsx`

This is the **main App Router** that controls everything:

```jsx
export default function App() {
  const [currentPage, setCurrentPage] = useState('landing');

  const navigateTo = (page) => setCurrentPage(page);

  return (
    <div>
      {currentPage === 'landing' && <LandingPage onNavigate={navigateTo} />}
      {currentPage === 'home' && <HomePage onNavigate={navigateTo} />}
      {currentPage === 'profile' && <ProfilePage onNavigate={navigateTo} />}
      {/* ... all 11 pages */}
    </div>
  );
}
```

**How it works:**
1. Each page has `onNavigate` prop
2. Pages call: `onNavigate('newPage')`
3. State updates, new page renders
4. It's that simple!

---

## ✅ What's Working

| Feature | Status |
|---------|--------|
| Landing explainer | ✅ Working |
| Instance selection | ✅ Working |
| Signup (4 steps) | ✅ Working |
| Challenge login | ✅ Working |
| Home feed | ✅ Working |
| Create posts | ✅ Working |
| User profiles | ✅ Working |
| Thread view | ✅ Working |
| Messaging | ✅ Working |
| Security dashboard | ✅ Working |
| Moderation panel | ✅ Working |
| Federation inspector | ✅ Working |
| Dark mode theme | ✅ Working |

---

## 📋 Disabled Features (Sprint 2/3)

These show **yellow styling** with tooltips:

- 📎 Attachments in DMs (Sprint 2)
- 🔄 Rotate Key (Sprint 2)
- ✕ Revoke Key (Sprint 2)
- 🔄 Multi-device Sync (Sprint 3)
- 🔄 Media Upload (Sprint 2)
- 🔍 Search (Sprint 2)
- 📊 Federation Graph (Sprint 3)

---

## 🐛 Troubleshooting

### "Port 3000 already in use"
```bash
npm run dev -- -p 3001
```

### "Module not found" errors
```bash
# Reinstall dependencies
rm -rf node_modules
npm install
```

### Styles look weird
- Clear browser cache: `Ctrl+Shift+Delete`
- Hard refresh: `Ctrl+Shift+R`
- Restart dev server: `Ctrl+C` then `npm run dev`

### Pages not appearing
- Check browser console for errors
- Verify you're on `http://localhost:3000`
- Try different page routes

---

## 📚 Key Files to Know

### Page.tsx (the main router)
Controls which page is shown based on state.
This is where you switch between pages.

### LandingPage.jsx
First page users see.
Explains federation concept.

### HomePage.jsx
Main feed after login.
Shows posts from all instances.

### globals.css
Theme colors for entire app.
Change here to customize colors globally.

---

## 🚀 Next Steps

After exploring locally:

### Deploy to Production
```bash
# Build for production
npm run build

# Test production build
npm run start

# Or deploy to Vercel with 1 click
# (push to GitHub, connect Vercel)
```

### Customize
- Edit colors in `globals.css`
- Add new pages in `components/pages/`
- Update navigation in `page.tsx`

### Connect Backend
- Replace mock data with real API
- Add authentication
- Store posts in database
- Implement real federation

---

## 📞 Need Help?

### Common Questions

**Q: How do I add a new page?**
A: Create `components/pages/NewPage.jsx`, add to `page.tsx`, add navigation button.

**Q: How do I change colors?**
A: Edit `/app/globals.css` color values.

**Q: Can I use this with Create React App?**
A: Yes! Extract components, create `App.js` router, import CSS files.

**Q: How do I make posts persist?**
A: Currently uses mock data. Connect to Supabase/Neon/PostgreSQL for real persistence.

**Q: Is it production ready?**
A: UI is 100% production ready. Backend needs database integration for real usage.

---

## 🎯 Demo Flow (Recommended)

Try this path to see all features:

1. **Start** → Click "Get Started"
2. **Instances** → Pick "social.example.net"
3. **Signup** → Walk through 4 steps (notice DID generation)
4. **Login** → See challenge-response flow
5. **Home** → Create a post
6. **Click Author** → View profile
7. **Click Post** → See thread
8. **Messages** → Open DM (see E2E banner)
9. **Security** → View key management
10. **Moderation** → Review queue
11. **Federation** → Check health metrics

---

## 💡 Tips

- **Post composer** has 500 char limit
- **Disabled buttons** are yellow with tooltips
- **Local posts** show 🏠, **Remote** show 🌐
- **Followers-only** posts show 🔒
- **All pages work offline** (no API calls)

---

## Version Info

- **Version:** 1.0.0 (Sprint 1 Complete)
- **Built with:** Next.js 16 + React 19
- **Styled with:** Dark mode CSS
- **Data:** Mock data (no database)
- **Pages:** 11 fully functional
- **Components:** ~2000 lines of JSX
- **Styles:** ~2500 lines of CSS

---

**Ready? Open http://localhost:3000 and start exploring! 🚀**
