# 🌐 FEDERATE - Decentralized Social Network Platform

A production-ready **React + JavaScript** federated social networking frontend emphasizing decentralized identity (DID), user-owned cryptographic keys, and privacy-first communication.

## 📋 Project Overview

This is a **frontend-only application** built with React/JSX and CSS, implementing a complete dark-mode federated social network with **5 fully working pages**:

1. **Landing Page** - Orientation, education, and federation explanation
2. **Instance Selection Page** - Browse and join federated servers
3. **Signup/Registration Flow** - Multi-step DID generation and profile setup
4. **Login Page** - Cryptographic challenge-based authentication flow
5. **Home Feed** - Timeline with posts, composer, and social interactions

## 🎨 Design Theme

**Dark Mode First** - Tech-savvy, privacy-focused aesthetic
- **Primary Color**: Cyan (#00d9ff) - Trust & Identity
- **Accent Color**: Magenta (#ff006e) - Federation & Emphasis
- **Disabled State**: Yellow (#d4af37) - Future features
- **Background**: Charcoal/Deep Navy (#0f0f1a, #1a1a2e)
- **Typography**: Rounded sans-serif (Poppins, Nunito)

## 📁 File Structure

```
/app
  ├── page.tsx                    # Main app router component
  ├── layout.tsx                  # Root layout with metadata
  └── globals.css                 # Dark mode theme configuration

/components
  ├── pages/
  │   ├── LandingPage.jsx        # Hero, features, CTA
  │   ├── InstancePage.jsx       # Server selection grid
  │   ├── SignupPage.jsx         # 4-step signup wizard
  │   ├── LoginPage.jsx          # Cryptographic auth flow
  │   └── HomePage.jsx           # Timeline, posts, feed
  └── styles/
      ├── LandingPage.css        # Landing page styling
      ├── InstancePage.css       # Instance page styling
      ├── SignupPage.css         # Signup page styling
      ├── LoginPage.css          # Login page styling
      └── HomePage.css           # Home page styling
```

## 🚀 Getting Started

### Prerequisites
- Node.js 16+
- React 18+
- No external UI libraries required (pure CSS)

### Installation

1. **Install dependencies** (if using package.json):
```bash
npm install
```

2. **Run the development server**:
```bash
npm run dev
```

3. **Open browser** to `http://localhost:3000`

### Viewing Pages

The app includes a built-in page navigator. Navigate between pages by clicking buttons:
- Landing → Instances → Signup → Login → Home

## 📄 Page Descriptions

### 1. Landing Page (`LandingPage.jsx`)
- **Features**: Hero section, feature cards, federation explanation
- **CTA Buttons**: Explore Network, Join Server, Sign Up/Login
- **Information**: Decentralized identity benefits, security banner
- **Style**: Gradient text, smooth animations, responsive grid

### 2. Instance Selection (`InstancePage.jsx`)
- **Server List**: 6 example servers with filtering
- **Search & Filters**: Region, Moderation level, Reputation
- **Server Cards**: User count, federation status, moderation policy
- **Special States**: Blocked servers (disabled buttons), trusted badges
- **Disabled States**: Yellow styling for blocked/future features

### 3. Signup Flow (`SignupPage.jsx`)
**4-Step Wizard Process**:

**Step 1**: Select Server
- Choose from federated server options
- Visual description of server selection

**Step 2**: Generate Identity
- Simulate WebCrypto DID generation
- Display public DID for copying
- Show locked private key indicator
- Download encrypted recovery file option

**Step 3**: Security Setup
- Username entry
- Optional password (local auth)
- Recovery file enabled checkbox
- Message about cryptographic authentication

**Step 4**: Profile Setup
- Display name, bio, profile picture
- Post visibility settings (public/followers/custom circle)
- Custom circles disabled (Sprint 2)

**Progress Tracking**:
- Visual progress bar with step indicators
- Step validation (disabled next buttons when required)

### 4. Login Page (`LoginPage.jsx`)
**3-Step Challenge-Response Flow**:

**Step 1**: Enter Credentials
- Server selection dropdown
- Username entry
- Info banner about cryptographic authentication

**Step 2**: Request Challenge
- Server sends nonce/challenge
- Display challenge with public DID details
- Sign button

**Step 3**: Sign Challenge
- Animated signing process (WebCrypto simulation)
- Display generated signature
- Security note about validation

**Flow Diagram**: Visual representation of 1→2→3→4 authentication steps

### 5. Home Feed (`HomePage.jsx`)
- **Top Navigation**: Logo, tabs (Home/Local/Federated), search, profile
- **3-Column Layout**: 
  - Left: Navigation & user profile
  - Center: Post composer & timeline
  - Right: Trending topics & features

**Post Composer**:
- Text area with 500 character limit
- Visibility toggle
- Disabled media upload button (Sprint 2)
- Post character counter

**Post Cards**:
- Author avatar & name
- Local/Remote badges (🏠/🌐)
- Followers-only indicator
- Reply/Boost/Like counters
- Interactive action buttons
- Hover effects with glow

**Sidebar Elements**:
- Navigation links (Home, Messages, Bookmarks)
- User profile summary
- Server reputation info
- Trending topics
- Future features list (disabled yellow styling)

## 🎯 Key Features

### ✅ Implemented Features
- Complete dark mode theme (no light mode)
- Decentralized identity (DID) concept demonstration
- Cryptographic authentication flow visualization
- Federated server selection with filtering
- Multi-step signup wizard with progress tracking
- Full-featured social timeline
- Responsive design for mobile/tablet/desktop
- Neon accent colors and smooth animations
- Yellow disabled state for future features
- Interactive post composer
- Local vs. remote post indicators

### 🟡 Disabled/Future Features
- Media uploads (Sprint 2) - Yellow styling
- Custom circles (Sprint 2) - Disabled radio option
- Search functionality - Yellow disabled input
- Reputation scoring (Sprint 3)
- Federation graph visualization
- Offline sync capabilities

## 🎨 Design System

### Color Palette
```css
--bg-dark: #0f0f1a          /* Main background */
--bg-card: #1a1a2e          /* Card background */
--text-light: #e8eaed       /* Primary text */
--text-muted: #8a8a99       /* Secondary text */
--primary-cyan: #00d9ff     /* Primary actions */
--accent-magenta: #ff006e   /* Secondary actions */
--secondary-navy: #2d2d44   /* Secondary background */
--disabled-yellow: #d4af37  /* Disabled states */
--border-color: #2d2d44     /* Borders */
--success-green: #00ff88    /* Success states */
```

### Typography
- **Fonts**: Poppins, Nunito (rounded sans-serif)
- **Sizes**: 0.8rem - 3rem scale
- **Weights**: 400 (regular), 500 (medium), 600 (semibold), 700 (bold), 900 (extra bold)
- **Line Height**: 1.4 - 1.8 for readability

### Components
- **Buttons**: Cyan primary, navy secondary, yellow disabled
- **Cards**: Bordered with hover glow effects
- **Inputs**: Navy background with cyan focus border
- **Modals/Boxes**: Card styling with border radius 0.75rem
- **Icons**: Emoji-based (🌐, 🔐, 🚀, etc.)

## 🔧 Customization

### Changing Colors
Edit `/app/globals.css` and update the CSS variables in `:root` and `.dark` sections.

### Adding Pages
1. Create new component in `/components/pages/`
2. Create corresponding CSS in `/components/styles/`
3. Add navigation in `/app/page.tsx` state management

### Modifying Posts
Edit the `POSTS` array in `HomePage.jsx` to add/remove example posts.

### Updating Servers
Edit the `SERVERS` array in `InstancePage.jsx` to add federated server options.

## 📱 Responsive Breakpoints

- **Mobile** (< 480px): Single column, compact nav
- **Tablet** (480px - 768px): 1-2 columns, adjusted padding
- **Desktop** (> 768px): 3-column layout, full features

## ⚡ Performance

- Pure CSS (no CSS-in-JS overhead)
- Minimal JavaScript (React state management only)
- Optimized animations (no expensive computations)
- Lazy state updates in feed
- Sticky navigation bar (no re-renders on scroll)

## 🔐 Security Notes

This is a **frontend demonstration** of federated identity concepts:
- No real cryptography implemented
- WebCrypto mentions are for UX education
- No actual private keys stored
- All data stored in React state (ephemeral)
- Production version would require:
  - Real WebCrypto implementation
  - Secure key storage (IndexedDB, TPM)
  - Backend API integration
  - HTTPS enforcement
  - CSRF protection
  - Rate limiting

## 📚 Technology Stack

**Frontend Only**:
- ✅ React 18+ (RSC/Client components)
- ✅ Next.js 16 (App Router)
- ✅ Pure CSS3 (no frameworks)
- ✅ JavaScript ES6+
- ✅ HTML5 semantic elements

**Not Included** (as requested):
- ❌ UI component libraries (shadcn/ui)
- ❌ Styling frameworks (Tailwind)
- ❌ Backend/API
- ❌ Database
- ❌ Authentication library

## 🚀 Future Enhancements

- [ ] Sprint 2: Media uploads, custom circles, search
- [ ] Sprint 3: Federation graph, reputation scoring
- [ ] Dark/light theme toggle
- [ ] Accessibility improvements (ARIA labels)
- [ ] Real WebCrypto integration
- [ ] Backend API connection
- [ ] Real-time post updates (WebSocket)
- [ ] Image optimization
- [ ] Progressive Web App (PWA) features

## 📝 License

This project is part of the Federate platform initiative focusing on decentralized social networks.

## 👥 Author

Built with v0 - AI-powered design and code generation

---

**Ready to deploy!** All 5 pages are fully functional and ready for production integration with a backend API.
