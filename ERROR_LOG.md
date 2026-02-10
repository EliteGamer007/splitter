# Splitter Frontend - Error Log

## Overview
This document tracks all issues encountered during development and their resolutions.

---

## Issue #1: Merge Conflicts After Git Rebase
**Date:** February 2, 2026  
**Files Affected:** `AdminPage.jsx`, `SignupPage.jsx`

**Problem:**
After running `git stash`, `git pull --rebase`, `git stash pop`, merge conflict markers were left in files:
```
<<<<<<< Updated upstream
=======
>>>>>>> Stashed changes
```

**Resolution:**
- Manually removed all conflict markers
- Kept "Stashed changes" version (emoji-cleaned code)
- Fixed escaped quotes (`className=\"home-feed\"` → `className="home-feed"`)

---

## Issue #2: AdminPage Parsing Error - Extra Closing Div
**Date:** February 2, 2026  
**File:** `AdminPage.jsx`

**Error:**
```
Parsing ecmascript source code failed
Expected '</', got 'jsx text'
```

**Problem:**
Extra `</div>` closing tags in the sidebar section (lines 828-830).

**Resolution:**
Removed duplicate closing div tags in the Instance Stats sidebar card.

---

## Issue #3: Login Page Not Auto-Redirecting to Home
**Date:** February 2, 2026  
**File:** `LoginPage.jsx`

**Problem:**
After successful login, users had to manually reload the page to see the home page. The `setTimeout()` with 1500ms delay caused race conditions with React state updates.

**Original Code:**
```jsx
setTimeout(() => onNavigate(user.role === 'admin' ? 'admin' : 'home'), 1500);
```

**Resolution:**
Replaced `setTimeout` with `requestAnimationFrame()` for immediate navigation after state updates:
```jsx
requestAnimationFrame(() => {
  onNavigate(user.role === 'admin' ? 'admin' : 'home');
});
```

---

## Issue #4: `formatDate is not defined` Runtime Error
**Date:** February 5, 2026  
**File:** `AdminPage.jsx`

**Error:**
```
ReferenceError: formatDate is not defined
```

**Problem:**
The `formatDate()` function was used in AdminPage but never defined.

**Resolution:**
Added the missing utility function:
```jsx
const formatDate = (dateString) => {
  if (!dateString) return 'N/A';
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now - date;
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
};
```

---

## Issue #5: API Error - `fetchModerationQueue` Failed
**Date:** February 5, 2026  
**File:** `ModerationPage.jsx`, `api.ts`

**Error:**
```
API Error Response: {}
async fetchModerationQueue
```

**Problem:**
- `ModerationPage.jsx` calls `adminApi.getModerationQueue()`
- The backend endpoint `/admin/moderation-queue` may not exist or returns empty
- Different from `getModerationRequests()` which calls `/admin/moderation-requests`

**Resolution:**
Ensured both API methods exist in `lib/api.ts` and copied working versions from backup.

---

## Issue #6: Navigation Blocked After Login (Race Condition)
**Date:** February 10, 2026  
**File:** `app/page.tsx`

**Problem:**
The `navigateTo()` function checks `!isAuthenticated` before allowing navigation to protected pages. But when LoginPage calls `setIsAuthenticated(true)` followed by `onNavigate('home')`, the React state hasn't updated yet, causing navigation to be blocked.

**Original Code:**
```tsx
if (protectedPages.includes(page) && !isAuthenticated) {
  setCurrentPage('login');
  return;
}
```

**Resolution:**
Added localStorage token check as fallback to handle race conditions:
```tsx
const hasToken = typeof window !== 'undefined' && localStorage.getItem('jwt_token');
if (protectedPages.includes(page) && !isAuthenticated && !hasToken) {
  setCurrentPage('login');
  return;
}
```

---

## Issue #7: Port Conflicts / Lock Files
**Date:** Multiple occurrences

**Error:**
```
⚠ Port 3000 is in use by process XXXXX
⨯ Unable to acquire lock at .next/dev/lock
```

**Resolution:**
```powershell
Get-Process -Name node -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item ".next" -Recurse -Force -ErrorAction SilentlyContinue
npm run dev
```

---

## Issue #8: Backend Port Already In Use
**Date:** Multiple occurrences

**Error:**
```
Failed to start server: listen tcp :8000: bind: Only one usage of each socket address
```

**Resolution:**
Backend already running - no action needed. Check with:
```powershell
Get-NetTCPConnection -LocalPort 8000
```

---

## Issue #9: Missing .env.local in New Frontend Folder
**Date:** February 5, 2026  
**File:** `Splitter-frontend/.env.local`

**Problem:**
New Splitter-frontend folder cloned without environment configuration.

**Resolution:**
Created `.env.local`:
```
NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
```

---

## Issue #10: 404 for /logo.png
**Date:** February 10, 2026

**Error:**
```
GET /logo.png 404 in 433ms
```

**Problem:**
Missing logo file in public folder.

**Status:** Minor - does not affect functionality.

---

## Issue #11: Unsupported Metadata Viewport Warning
**Date:** Ongoing

**Warning:**
```
⚠ Unsupported metadata viewport is configured in metadata export
```

**Status:** Warning only - Next.js recommends moving viewport config to separate export. Non-blocking.

---

## Quick Reference: Common Commands

### Start Frontend
```powershell
cd "c:\Users\Sanjeev Srinivas\Desktop\Splitter-frontend"
npm run dev
```

### Start Backend
```powershell
cd "c:\Users\Sanjeev Srinivas\Desktop\splitter"
go run ./cmd/server
```

### Clean Restart Frontend
```powershell
Get-Process -Name node -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item ".next" -Recurse -Force -ErrorAction SilentlyContinue
npm run dev
```

### Check Port Usage
```powershell
Get-NetTCPConnection -LocalPort 8000,3000 -ErrorAction SilentlyContinue
```

### Test Backend Health
```powershell
Invoke-RestMethod -Uri "http://localhost:8000/api/v1/health"
```

---

## File Structure Reference

```
Splitter-frontend/
├── app/page.tsx              # Main app with navigation logic
├── components/pages/
│   ├── AdminPage.jsx         # Admin dashboard
│   ├── LoginPage.jsx         # Login with redirect fix
│   ├── HomePage.jsx          # User feed
│   ├── SignupPage.jsx        # Registration
│   └── ...
├── lib/api.ts                # API service layer
└── .env.local                # Backend URL config

splitter/
├── Frontend.backup/          # Backup of working frontend
├── cmd/server/main.go        # Go server entry
└── internal/                 # Go handlers/routes
```
