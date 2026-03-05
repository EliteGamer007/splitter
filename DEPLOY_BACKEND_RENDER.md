# Backend Deployment Feasibility: Render (Docker)

## Feasibility Verdict

**Status: Feasible now, with 2 deployment caveats.**

What is already ready:
- Dockerized backend image in [Dockerfile](Dockerfile) (`server`, `worker`, `migrate` binaries built)
- CI and image publish pipeline in [.github/workflows/backend-ghcr-publish.yml](.github/workflows/backend-ghcr-publish.yml)
- Manual deploy-hook template in [.github/workflows/backend-deploy-template.yml](.github/workflows/backend-deploy-template.yml)
- Existing two-instance local topology in [docker-compose.instances.yml](docker-compose.instances.yml)

Caveats to handle before production cutover:
1. **CORS allowlist is hard-coded to localhost** in [internal/server/router.go](internal/server/router.go). For Vercel frontend access, add deployed frontend domain(s) to allowed origins (or make origin list env-driven).
2. **Migration strategy is manual/non-versioned for cloud deploys**. The `migrate` binary in [cmd/migrate/main.go](cmd/migrate/main.go) is additive but not a full sequential migration runner.

---

## Recommended Render Topology

For federated demo with two backend instances:
- `splitter-instance-1` (Web Service, Docker)
- `splitter-instance-2` (Web Service, Docker)
- `splitter-worker-1` (Background Worker, Docker command: `worker`)
- `splitter-worker-2` (Background Worker, Docker command: `worker`)

Database:
- Prefer **separate Neon DB/branch per instance** for realistic federation isolation.

---

## Environment Variables (per backend instance)

Required:
- `PORT` (Render injects it; keep service bound to it)
- `ENV=production`
- `DB_HOST`
- `DB_PORT=5432`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET` (strong random)

Federation (instance-specific):
- `FEDERATION_ENABLED=true`
- `FEDERATION_DOMAIN=<public-domain-or-instance-id>`
- `FEDERATION_URL=https://<render-service-domain>`
- `BASE_URL=https://<render-service-domain>`

Worker tuning (optional):
- `WORKER_RETRY_INTERVAL_SECONDS`
- `WORKER_REPUTATION_INTERVAL_SECONDS`
- `WORKER_CIRCUIT_COOLDOWN_SECONDS`
- `WORKER_MAX_RETRY_COUNT`
- `WORKER_CIRCUIT_FAILURE_THRESHOLD`

---

## Deployment Paths

## Option A (fastest): Render builds from GitHub repo
1. Create Render Web Service from repo root `splitter`.
2. Environment: Docker.
3. Start command: default image command (`server`) is fine.
4. Add env vars above.
5. Repeat for instance 2 with distinct federation/db settings.
6. Create 2 worker services using same Docker image and command `worker`.

## Option B (registry-first): GHCR image pull
1. Publish image from GitHub Actions (`backend-ghcr-publish.yml`).
2. In Render, configure private registry auth (if package is private).
3. Point services to `ghcr.io/<owner>/splitter-backend:<tag>`.
4. Same env variable setup as Option A.

---

## Migration Workflow (safe practical approach)

Because cloud DB migration is manual in this repo:
1. Before first deploy, apply SQL baseline to each target DB:
   - `migrations/000_master_schema.sql`
   - `migrations/002_upgrade_to_current.sql`
   - additional migration files used by integration setup (014, 015, 016, 017)
2. Verify with smoke query / health endpoint.
3. Deploy web + workers.

For repeatable automation later, introduce a proper migration runner (`goose`/`tern`), then add pre-deploy migration stage.

---

## Render Setup Checklist (Backend)

- [ ] Create instance 1 web service
- [ ] Create instance 2 web service
- [ ] Create worker 1 (`worker` command)
- [ ] Create worker 2 (`worker` command)
- [ ] Configure per-instance DB and federation envs
- [ ] Apply DB schema/migrations to both DBs
- [ ] Health check passes at `/api/v1/health`
- [ ] Verify cross-instance follow + DM federation
- [ ] Configure deploy hooks and map into GitHub secrets:
  - `BACKEND_INSTANCE1_DEPLOY_HOOK_URL`
  - `BACKEND_INSTANCE2_DEPLOY_HOOK_URL`

---

## Immediate Next Step

After backend services are up on Render, update backend CORS to allow your Vercel domain(s) before frontend rollout.
