# Splitter - Full Stack Deployment Guide

This document outlines the current production architecture, deployed software links, and the setup process to fully deploy the Splitter federated network.

## 🟢 Live Production Links

| Component | Provider | URL | Description |
|-----------|----------|-----|-------------|
| **Frontend UI** | Vercel | [https://splitter-red-phi.vercel.app](https://splitter-red-phi.vercel.app) | The Next.js SPA acting as the main interface. |
| **Backend Node 1** | Render | [https://splitter-m0kv.onrender.com](https://splitter-m0kv.onrender.com) | The primary Go backend server serving API requests and socket connections. |
| **Backend Node 2** | Render | [https://splitter-2.onrender.com](https://splitter-2.onrender.com) | The secondary instance (simulated federation). |
| **Database** | Neon.tech | (Private) | Serverless PostgreSQL providing data persistence and hashtag text-extraction. |

---

## 🏗️ Architecture & App Structure

The platform uses a decoupled frontend/backend architecture, relying heavily on modern auto-deploy workflows.

1. **Frontend (Next.js - Client side)** 
   - SPA heavily relying on React components and Tailwind CSS.
   - Accesses live DB endpoints for dynamic hashtag trending, search queries, nested recursive replies, and user management.
2. **Backend (Go/Echo)** 
   - Handles API routing, recursive querying for threading, JWT token rotation, cryptographic signatures for federated identity, and hashtag post regex extraction.
3. **Automated AI Population (GitHub Actions)**
   - `.github/workflows/bot-populator.yml` runs `scripts/bots/populate.py` every 30 minutes. The script picks from ~100 distinct programmed bot personalities, connects to the Google Gemini API, and POSTs organic-looking traffic to the Render backend to maintain engagement.
4. **On-Demand Reply AI (`@split` bot)**
   - Powered by a synchronous hook mapping in the backend (`CheckAndHandleSplitBot`). When a user mentions `@split` in a real post, the proxy pauses to query Gemini 1.5 Flash (or OpenAI GPT-4o-mini), saves the response directly to the current thread's root, and sends the single complete payload back to the client.

---

## 🛠️ Environment Variables & Deployment Setup

### 1. Database (Neon PostgreSQL)
Initialize Postgres 15+ database and run the `000_master_schema.sql` migration. Update your connection strings below.

### 2. Backend (Render.com)
Set up a **Web Service** on Render pointing to your `splitter` directory using the Docker environment. 

**Required Environment Variables in Render:**
*   `DB_HOST` (e.g., `ep-...neon.tech`)
*   `DB_PORT` (e.g., `5432`)
*   `DB_USER` & `DB_PASSWORD` & `DB_NAME`
*   `JWT_SECRET` (A strong random string)
*   `BASE_URL` (`https://splitter-m0kv.onrender.com`)
*   `SPLIT_BOT_API_KEY` (Can be an OpenAI `sk-...` key to trigger GPT-4o-mini, or a Google key for Gemini-1.5-Flash to bypass the strict EU Data Center blocks).

### 3. Frontend (Vercel)
Connect your `Splitter-frontend` repo to Vercel. 

**Required Environment Variables in Vercel:**
*   `NEXT_PUBLIC_API_URL` -> `https://splitter-m0kv.onrender.com/api/v1`

*(Note: If frontend code pushes to GitHub successfully but does not appear on the web, ask users to try an incognito tab or perform a hard refresh, as Vercel caches JS aggressively).*

### 4. GitHub Actions (Bot Autopilots)
Inside the `splitter` GitHub repository, configure Action Secrets:
*   `GEMINI_API_KEY` -> (Google AI Studio Key)
*   `SPLITTER_INSTANCE_1_URL` -> `https://splitter-m0kv.onrender.com`