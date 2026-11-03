# Repair status — 3 November 2026

## Release state

The repairs and final presentation polish are in the **local `codex/free-tier-repair` branches** in the backend and separate frontend checkouts. No commit, push, production migration or hosting deployment was performed during this repair. The existing public app is running its previously deployed revision.

Direct HTTP checks on 3 November returned:

| URL | Result |
|---|---|
| https://splitter-social.vercel.app | 200, HTML received |
| https://splitter-m0kv.onrender.com/api/v1/health | 200, `{"status":"ok"}` |
| https://splitter-m0kv.onrender.com/api/v1/status | 404; the repaired capability endpoint is not deployed |

These are reachability checks, not a claim that every production feature or the production database has been verified. The earlier audit remains in `PROJECT_STATUS_REPORT_2026-10-31.md` as historical evidence.

## Repairs in this branch

- Correct Ed25519 challenge verification, nonce expiration/replay handling, current account roles, and administrator authorization boundaries.
- Password-proven recovery for legacy invalid signing keys, atomic key registration/rotation, and protected device enrollment.
- Shared post/reply/media audience checks, follow approval for locked profiles, public profile DTOs without private account fields, and participant/consent checks for messaging.
- Encrypted sends and edits omit plaintext; browser keys and caches are scoped to the account and API origin. Recovery exports are encrypted and validate imported key pairs.
- Durable, bounded post/avatar/story media, image processing limits and transactional media budgets. Stories respect audience, expiry and viewer state.
- Federation request integrity, signer/object ownership checks, guarded remote HTTP, explicit audiences and private-content exclusion from public outboxes.
- Restored optional public `@split` replies with configured provider/model, finite persistent daily budgets, one active request and no idle work.
- Safe startup with no automatic credential reset, schema mutation or destructive cleanup. Explicit transactional migrations and first-administrator bootstrap.
- Four-connection default database pool, bounded pages, request deadlines, visible-page unread checks, and no required second hosted instance or worker.
- Navigation/deep links, feed actions, pagination controls, useful empty/error states, account/privacy settings and mobile layout repairs while retaining the existing visual design.
- Final polish includes authenticated locked-story images, mobile zoom accessibility, nullable legacy profile names, accurate README/setup/deployment documentation, a named frontend package and a team-aware resume/demo walkthrough.
- Removed cron triggers from keep-alive and bot-population workflows; manual tools remain available. The manual health check contacts only the primary service.

## Verification

Final checks were deliberately limited to finishing the release, rather than extending the audit:

- Backend: complete `go test -p 1 -count=1 ./cmd/... ./internal/... ./tests/...` passed against an explicit disposable local PostgreSQL database. The first attempt encountered the stopped local cluster; the recorded successful rerun followed startup recovery.
- Backend command build and `go vet ./cmd/... ./internal/... ./tests/...` passed. The final repository, router and pagination changes passed a focused rerun.
- Frontend: `npm run check` passed: TypeScript, 28 Jest suites / 80 tests / two snapshots, and optimized production build. ESLint reported zero errors and 16 hook-dependency warnings.
- Desktop/mobile smoke: all six Playwright checks passed for guest navigation, refresh/back, privileged deep links and real login-form error handling.
- Earlier implementation browser checks exercised local password login, posting, replies, likes, locked follow approval and encrypted DMs on desktop/mobile; screenshots are in the ignored frontend `.cache/browser-live` folder. This was a local disposable API, not the production service.
- Final story-image regression check passed all three image tests, followed by another successful optimized production build and TypeScript compilation. No external AI request or production database test was used.

## Scope and practical limits

This is a polished course-project release candidate, not a claim of exhaustive security certification or universal ActivityPub compatibility. Full two-server interoperability, provider account usage and historical production-schema compatibility need validation during a deliberate production rollout. AI requires an enabled provider account/model. Lost encryption keys cannot be reconstructed from a password; users need their recovery backup.

Media budgets and the cached database-size guard reduce growth; they do not guarantee compliance with every provider quota under arbitrary traffic. Free-host cold starts remain expected. Existing legacy uploads that were lost from an ephemeral disk cannot be recovered by code.

## Release handoff

Follow [DEPLOYMENT.md](../DEPLOYMENT.md): preserve a backup/recovery point, apply explicit migrations once against the existing database, release the existing primary backend, then release the corresponding frontend revision. Keep identities/secrets and existing accounts intact. Confirm `/api/v1/status` identifies the intended revision and walk through [DEMO.md](DEMO.md) on the public deployment.
