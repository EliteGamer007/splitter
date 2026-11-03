# Splitter deployment

The portfolio configuration targets **one existing Render free web service**, Vercel, and Neon. Apply `render.backend.yaml` settings to the existing primary service; importing it as a new blueprint could create a duplicate service. A second instance and background workers are optional development tools, not required hosted services.

## Deployment identities

| Component | Address |
|---|---|
| Frontend | https://splitter-social.vercel.app |
| Primary API | https://splitter-m0kv.onrender.com/api/v1 |
| Historical peer | https://splitter-2.onrender.com (not a default dependency) |
| Database | Existing private Neon PostgreSQL project |

The old `splitter-red-phi.vercel.app` frontend address is obsolete. The 31 October audit observed both Render APIs and the current frontend working. The local November repair branch has not been deployed or verified against the production database. Never infer a production rollout from a local build.

## Backend configuration

Use Docker from the repository root. The image includes `server`, `migrate`, `bootstrap-admin`, the optional worker binary, and the migration SQL. Keep the default start command `server`.

| Variable | Production setting |
|---|---|
| `ENV` | `production` |
| `PORT` | Supplied by Render |
| `BASE_URL`, `FEDERATION_URL` | `https://splitter-m0kv.onrender.com` |
| `FEDERATION_DOMAIN` | Preserve the deployed identity, normally `splitter-1` |
| `FEDERATION_ENABLED` | `true` to retain federation |
| `CORS_ORIGINS` | `https://splitter-social.vercel.app` |
| `JWT_SECRET` | Existing strong random secret, at least 32 characters |
| `JWT_EXPIRATION_HOURS` | `24` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Existing Neon credentials; port normally 5432 |
| `DB_SSLMODE` | `require` |
| `DB_MAX_CONNS` | `4` (minimum idle connections stay zero) |
| `MEDIA_TOTAL_BUDGET_BYTES` | `33554432` |
| `MEDIA_ACCOUNT_BUDGET_BYTES` | `2097152` |
| `DB_WRITE_BUDGET_BYTES` | `268435456` |

Use the provider-recommended pooled database hostname where applicable. Do not change database names, federation identities or secrets as part of a routine deployment. `FEDERATION_INSTANCE_MAP` is optional for explicitly configured peers; leaving it absent avoids depending on the historical second service. `FEDERATION_LOCAL_PEERS` is a development-only exception for exact local endpoints and must be absent in production.

Render health check: `/api/v1/health`. It is cheap and does not query Neon. `/api/v1/ready` checks database connectivity on demand. `/api/v1/status` exposes version/capability metadata and the optional AI configuration state without credentials. `APP_VERSION` can override Render's `RENDER_GIT_COMMIT` version value.

## Migrate and release

1. Preserve a database backup or provider recovery point and record the currently deployed backend/frontend revisions. Inspect existing legacy schema before the first repaired release; successful fresh-database tests do not prove every historical deployment matches it.
2. Build the backend image and apply migrations **once**, using the same database environment as the server: `migrate -dir /app/migrations` in the image, or `go run ./cmd/migrate` from source. The runner applies the master baseline followed by the ordered upgrades, uses a migration lock, commits successful migrations transactionally, and exits nonzero on failure. Stop the release on an error.
3. Deploy the API to the existing primary service. Startup only validates configuration and opens the database; it performs no automatic account resets, data cleanup or schema alterations.
4. Verify health, readiness, status and CORS for the frontend origin. Then publish the matching frontend revision.
5. Complete the short walkthrough in [docs/DEMO.md](docs/DEMO.md): login, post/reply, profile, follow approval, encrypted DM and both themes. Confirm the public production domain points at the intended Vercel deployment.

Use an additive rollout with a backup. Do not blindly reverse schema changes or delete migration records to roll back an app revision.

To create the first administrator only, supply `BOOTSTRAP_ADMIN_USERNAME`, `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` (12–72 bytes), then run `bootstrap-admin` or `go run ./cmd/bootstrap-admin`. This command refuses to modify an existing account or replace an existing administrator. Remove these temporary environment values afterwards. The API never ships a default administrator password.

## Frontend on Vercel

The frontend is maintained in the separate [Splitter-frontend repository](https://github.com/EliteGamer007/Splitter-frontend). Use Node.js 22 and `npm ci`; build with `npm run build`.

```env
NEXT_PUBLIC_API_URL=https://splitter-m0kv.onrender.com/api/v1
NEXT_PUBLIC_INSTANCE_1_URL=https://splitter-m0kv.onrender.com/api/v1
NEXT_PUBLIC_INSTANCE_2_URL=
```

Public API variables are embedded at build time; changing them requires rebuilding. A custom or explicit second peer remains supported through instance selection. Do not add a second service just to fill an instance card.

## Optional AI and automation

`@split` replies work only with a configured `SPLIT_BOT_API_KEY` and supported provider/model. Set `SPLIT_BOT_PROVIDER` and `SPLIT_BOT_MODEL` explicitly for the provider account you use. Gemini, OpenAI, Groq and an explicitly configured Ollama endpoint are supported. Provider/model availability and billing are external to this app.

`SPLIT_BOT_DAILY_LIMIT=10` and `SPLIT_BOT_USER_DAILY_LIMIT=3` bound persistent attempt reservations, with one active request and no automatic retries. Only public mentions from active users trigger work. Leave the key absent until provider usage is configured; the status endpoint explains that state.

Do not enable keep-alive or population schedules for the portfolio deployment. They consume service hours/database compute and are not necessary for a working social app. The backend CI workflow uses disposable PostgreSQL and needs no production Neon secret. Publishing a GHCR image does not by itself deploy Render or Vercel.

## Free-tier expectations

The application reduces idle traffic, limits pool connections, compresses durable media and uses finite request/AI budgets. The database size guard is a cached soft threshold and does not account for all provider history, branches, transfer or other projects. Monitor the actual Render and Neon dashboards after rollout. Free services can sleep and impose quotas; a cold first request is expected, and no code change can guarantee unlimited availability or usage.

New image bytes survive Render restarts because they are in PostgreSQL. Old files that existed only on an ephemeral disk cannot be recreated by a deployment; retain any available original uploads before replacing a service. Legacy routes only serve database-owned media with the relevant visibility checks.
