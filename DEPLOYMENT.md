# Splitter — Deployment Guide

Full deployment reference for the Splitter federated social platform. Covers live services, environment configuration, Render topology, database setup, migration workflow, and CI/CD.

---

## Live Production

| Component | Provider | URL |
|-----------|----------|-----|
| Frontend UI | Vercel | https://splitter-red-phi.vercel.app |
| Backend Node 1 | Render | https://splitter-m0kv.onrender.com |
| Backend Node 2 | Render | https://splitter-2.onrender.com |
| Database | Neon.tech | Private (serverless PostgreSQL) |

---

## Architecture

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js (React) on Vercel |
| Backend API | Go/Echo — two federated instances on Render |
| Database | PostgreSQL 15 on Neon (per-instance branches recommended) |
| Workers | Background worker binary (`worker`) per instance |
| Bot Population | GitHub Actions cron — `scripts/bots/populate.py` every 30 min via Gemini API |
| `@split` AI Bot | Synchronous backend hook → Gemini 1.5 Flash / GPT-4o-mini |

---

## 1. Database — Neon PostgreSQL

### Setup

1. Create a project at [console.neon.tech](https://console.neon.tech).
2. For federated instances, create **one Neon branch per backend instance** for isolation.
3. Copy the connection string from the dashboard:
   ```
   postgres://user:password@ep-xyz.region.neon.tech/neondb?sslmode=require
   ```

### Running Migrations

Apply the baseline schema before first deploy:

```bash
# Baseline schema
docker run --rm postgres:15 psql 'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/000_master_schema.sql

# Verify
docker run --rm postgres:15 psql 'YOUR_NEON_CONNECTION_STRING' \
  -f migrations/verify_migration.sql
```

Look for `✓ PASS` at the end of verification output.

> **Note:** The `migrate` binary (`cmd/migrate/main.go`) is additive but not a full sequential runner. For production automation, consider `goose` or `tern` with a pre-deploy migration stage.

For fresh environments that skip the baseline, also apply:
- `migrations/002_upgrade_to_current.sql`
- `migrations/014_*`, `015_*`, `016_*`, `017_*`

---

## 2. Backend — Render.com

### Recommended Topology (Federated Demo)

| Service | Type | Docker Command |
|---------|------|---------------|
| `splitter-instance-1` | Web Service (Docker) | `server` (default) |
| `splitter-instance-2` | Web Service (Docker) | `server` (default) |
| `splitter-worker-1` | Background Worker (Docker) | `worker` |
| `splitter-worker-2` | Background Worker (Docker) | `worker` |

### Deployment Options

**Option A — Render builds from GitHub (fastest)**

1. Create a Render Web Service from repo root `splitter`.
2. Set environment to **Docker**.
3. Default start command (`server`) is fine.
4. Add environment variables (see below).
5. Repeat for instance 2 with distinct federation/DB settings.
6. Create two worker services using same image with command `worker`.

**Option B — GHCR Registry Pull**

1. Publish image via GitHub Actions (`backend-ghcr-publish.yml`).
2. Configure private registry auth in Render if the package is private.
3. Point services to `ghcr.io/<owner>/splitter-backend:<tag>`.
4. Use same env variable setup as Option A.

### Environment Variables

**Database:**

| Variable | Example |
|----------|---------|
| `DB_HOST` | `ep-falling-leaf-123.us-east-2.aws.neon.tech` |
| `DB_PORT` | `5432` |
| `DB_USER` | `neondb_owner` |
| `DB_PASSWORD` | `<your-password>` |
| `DB_NAME` | `neondb` |

**Application:**

| Variable | Example |
|----------|---------|
| `PORT` | Injected by Render automatically |
| `ENV` | `production` |
| `JWT_SECRET` | Strong random string |
| `BASE_URL` | `https://splitter-m0kv.onrender.com` |
| `SPLIT_BOT_API_KEY` | OpenAI `sk-...` or Google Gemini key |

**Federation (per-instance):**

| Variable | Example |
|----------|---------|
| `FEDERATION_ENABLED` | `true` |
| `FEDERATION_DOMAIN` | `splitter-1` / `splitter-2` |
| `FEDERATION_URL` | `https://<render-service-domain>` |

**Worker Tuning (optional):**

| Variable |
|----------|
| `WORKER_RETRY_INTERVAL_SECONDS` |
| `WORKER_REPUTATION_INTERVAL_SECONDS` |
| `WORKER_CIRCUIT_COOLDOWN_SECONDS` |
| `WORKER_MAX_RETRY_COUNT` |
| `WORKER_CIRCUIT_FAILURE_THRESHOLD` |

### CORS Configuration

The CORS allowlist in `internal/server/router.go` defaults to `localhost`. Before frontend rollout, add your Vercel domain(s):

```go
AllowOrigins: []string{
    "http://localhost:3000",
    "https://splitter-red-phi.vercel.app",
},
```

---

## 3. Frontend — Vercel

1. Connect `Splitter-frontend` repo to Vercel.
2. Set environment variable:

| Variable | Value |
|----------|-------|
| `NEXT_PUBLIC_API_URL` | `https://splitter-m0kv.onrender.com/api/v1` |

> If new frontend builds don't reflect on live, advise hard-refresh or incognito — Vercel aggressively caches static JS assets.

---

## 4. GitHub Actions — Secrets

Configure in GitHub repository → Settings → Secrets:

| Secret | Purpose |
|--------|---------|
| `GEMINI_API_KEY` | Google AI Studio key for bot population |
| `SPLITTER_INSTANCE_1_URL` | `https://splitter-m0kv.onrender.com` |
| `SPLITTER_INSTANCE_2_URL` | `https://splitter-2.onrender.com` |
| `BACKEND_INSTANCE1_DEPLOY_HOOK_URL` | Render deploy hook for instance 1 |
| `BACKEND_INSTANCE2_DEPLOY_HOOK_URL` | Render deploy hook for instance 2 |

CI pipelines:

| Workflow | Purpose |
|----------|---------|
| `backend-ghcr-publish.yml` | Builds and pushes Docker image to GHCR |
| `backend-deploy-template.yml` | Manual deploy-hook trigger |
| `bot-populator.yml` | Scheduled bot traffic (every 30 min) |

---

## 5. Local Development

```bash
cp .env.example .env
```

```env
# Database (Neon)
DB_HOST=ep-your-endpoint.region.aws.neon.tech
DB_PORT=5432
DB_USER=neondb_owner
DB_PASSWORD=your-password
DB_NAME=neondb

# Application
PORT=8000
ENV=development
BASE_URL=http://localhost:8000
JWT_SECRET=change-this-to-a-secure-random-string

# AI Bot (optional locally)
SPLIT_BOT_API_KEY=

# Federation (optional locally)
FEDERATION_ENABLED=false
FEDERATION_DOMAIN=localhost
FEDERATION_URL=http://localhost:8000
```

---

## 6. Deploy Checklist

### Backend
- [ ] Neon DB project created; baseline migration applied to both instances
- [ ] Instance 1 web service created on Render (Docker)
- [ ] Instance 2 web service created on Render (Docker)
- [ ] Worker 1 background service created (`worker` command)
- [ ] Worker 2 background service created (`worker` command)
- [ ] All environment variables set per instance
- [ ] Health check passes: `GET /api/v1/health` → `200 OK`
- [ ] Cross-instance follow + federated timeline verified
- [ ] CORS updated to include Vercel frontend domain

### Frontend
- [ ] Vercel project connected to `Splitter-frontend` repo
- [ ] `NEXT_PUBLIC_API_URL` set to instance 1 URL
- [ ] Login + post flow tested on live URL

### Automation
- [ ] GitHub Action secrets configured
- [ ] `bot-populator.yml` cron confirmed running
- [ ] Deploy hooks registered in GitHub secrets
