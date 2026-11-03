# Splitter

A federated social network built as a college team project with Go, Next.js and PostgreSQL. Splitter combines a social feed, browser-managed identity, encrypted messaging and ActivityPub federation in a peach light theme and a dark theme.

**[Live demo](https://splitter-social.vercel.app)** · [Backend health](https://splitter-m0kv.onrender.com/api/v1/health) · [Deployment guide](DEPLOYMENT.md) · [Demo walkthrough](docs/DEMO.md)

> The November 2026 repair changes are in the local `codex/free-tier-repair` branches. The public links identify the existing deployment; they do not establish that these changes have been deployed. See [repair status](docs/REPAIR_STATUS.md).

## Features

- Social feed with photo posts, likes, reposts, bookmarks, nested replies, search and hashtags.
- Profiles, followers, locked accounts with follow approval, and circle audiences.
- Expiring stories with durable media and viewer state.
- Password/JWT authentication and Ed25519 challenge-response login.
- Browser-side P-256 key agreement and encrypted direct messages; encrypted sends omit plaintext from the API payload. Account-scoped keys and passphrase-encrypted recovery export preserve access across sessions.
- ActivityPub actors, WebFinger, signed delivery, remote follows and federated posts. Federation remains configurable with one hosted primary; an additional peer is optional.
- Role-guarded administration and moderation tools.
- Optional `@split` AI replies to public mentions, with explicit provider configuration and persistent daily budgets.
- Responsive navigation, shareable page URLs, and light/dark themes.

## Architecture

```mermaid
flowchart LR
    Browser[Next.js / React on Vercel] -->|HTTPS REST API| API[Go / Echo on Render]
    API -->|pgx connection pool| DB[(PostgreSQL on Neon)]
    API <-->|Signed ActivityPub| Peer[Optional independent peer]
    API -->|Configured public mentions only| AI[Optional AI provider]
```

The portfolio configuration uses **one Render web service**, one database, and no separate workers or scheduled keep-alive requests. This retains federation support while keeping idle resource use low.

| Layer | Implementation |
|---|---|
| Backend | Go 1.26, Echo v4, pgx v5 |
| Frontend | Next.js 16, React 19, TypeScript/JavaScript, Tailwind CSS, Radix UI |
| Persistence | PostgreSQL; post, avatar and story images stored durably in the database |
| Authentication | bcrypt, HS256 JWT, Ed25519 signatures |
| Messaging | Browser encryption, account/origin-scoped key storage and recovery export |
| Federation | ActivityPub, WebFinger, RSA HTTP signatures |
| Hosting | Vercel, Render and Neon |

## Run locally

Prerequisites: Go matching `go.mod`, Node.js 22.13+ (below 25), npm, and a local PostgreSQL database. Use a separate development database, not the deployed Neon database.

1. Copy `.env.example` to `.env`; set database credentials, `DB_SSLMODE=disable` for local PostgreSQL, and a strong `JWT_SECRET`.
2. Set `BASE_URL` and `FEDERATION_URL` to `http://localhost:8000`, and `ENV=development`.
3. Apply migrations explicitly and start the API:

```sh
go run ./cmd/migrate
go run ./cmd/server
```

The server does not migrate the schema, reset credentials or create administrators on startup. To create the first administrator, use `cmd/bootstrap-admin` with the documented environment variables in [DEPLOYMENT.md](DEPLOYMENT.md).

The frontend is a **separate repository** at [Jaswita/Splitter-frontend](https://github.com/EliteGamer007/Splitter-frontend). In a combined local checkout:

```sh
cd Splitter-frontend
npm ci
```

Create `.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
NEXT_PUBLIC_INSTANCE_1_URL=http://localhost:8000/api/v1
NEXT_PUBLIC_INSTANCE_2_URL=
```

```sh
npm run dev
```

Open [localhost:3000](http://localhost:3000). HTTPS or localhost is required for browser cryptography. No external AI key or second hosted service is needed for the core app.

## Resource controls

The default database pool has four maximum connections and no minimum idle connections. Lists are paginated with a maximum of 50 items. Requests and outbound federation calls have deadlines. Guest pages avoid waking the backend for decorative statistics; unread checks pause when the page is hidden or offline.

Images accept up to 5 MiB of input with bounded decoding and processing. Stored derivatives are limited to 256 KiB (128 KiB for avatars); shared media budgets default to 32 MiB total and 2 MiB per account. A request-triggered 256 MiB database size threshold is a **soft growth guard**, not a provider quota guarantee. Existing content stays readable when a storage budget is reached.

Optional AI replies default to ten attempts per day for the app and three per account, with one active request. Leave the provider key unset to avoid AI usage. Bot-population and keep-alive schedules are unnecessary for the portfolio setup.

## Verification

```sh
go build ./cmd/...
go vet ./cmd/... ./internal/... ./tests/...
```

Database tests only use an explicitly supplied local disposable database with a name beginning `splitter_repair`:

```sh
TEST_DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/splitter_repair_tests?sslmode=disable' go test -p 1 ./cmd/... ./internal/... ./tests/...
```

In the frontend, `npm run check` runs lint, type checking, Jest and a production build. Browser tests live in `__tests__/e2e`; the full local API flow requires `LOCAL_API_E2E=1` and the disposable API on port 18000. See [repair status](docs/REPAIR_STATUS.md) for the actual verification result and remaining deployment work.

## Project layout

```text
cmd/                    server, explicit migrations, first-admin bootstrap
internal/auth/          passwords, tokens and identity verification
internal/federation/    signatures, guarded HTTP client, actors and delivery
internal/handlers/      API handlers and authorization boundaries
internal/repository/    PostgreSQL queries and visibility policies
internal/service/       image processing, stories and optional AI replies
migrations/             ordered SQL migrations
Splitter-frontend/      separate frontend repository in the local workspace
tests/                  unit, integration and disposable-database helpers
docs/                   demo walkthrough, repair status and implementation notes
```

## Team and attribution

This was built collaboratively for a college course. When presenting it, describe the shared architecture and identify your own contributions explicitly. The demo walkthrough includes suggested interview talking points; performance and usage claims should come from measured results.

[MIT license](LICENSE).
