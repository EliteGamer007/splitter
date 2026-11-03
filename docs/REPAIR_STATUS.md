# Production release report — 3 November 2026

## Deployment

The repaired application is deployed to the existing production services:

| Component | Production target | Verification |
|---|---|---|
| Frontend | https://splitter-social.vercel.app | Vercel production deployment ready; page and logo return 200 |
| Primary API | https://splitter-m0kv.onrender.com/api/v1 | Health, database readiness, public feed and federation status return 200 |
| Historical secondary | https://splitter-2.onrender.com | Suspended; automatic deployment disabled; its database retained |
| Database | Existing Neon production branch | Additive migrations committed after a verified encrypted backup |

The Vercel project is `splitter`, linked to the production fork [EliteGamer007/Splitter-frontend](https://github.com/EliteGamer007/Splitter-frontend). The team upstream remains credited. The backend is [EliteGamer007/splitter](https://github.com/EliteGamer007/splitter). Backend automatic deployment is disabled so future schema upgrades can be applied deliberately before a manual application release. Vercel follows the frontend production branch.

The old `splitter-red-phi.vercel.app` address is obsolete. The historical audit in `PROJECT_STATUS_REPORT_2026-10-31.md` describes the pre-repair state. It is not the status of this release.

## Security and credential changes

- Correct Ed25519 challenge verification, nonce expiry/replay handling, current account-role checks and administrator authorization boundaries.
- Atomic key registration/rotation, protected device enrollment and password-proven recovery for legacy invalid signing keys.
- Shared visibility checks for posts, replies, interactions and media; locked-account follow approval; public profile responses exclude private account fields.
- Encrypted web-client sends/edits omit plaintext. Browser keys and caches are scoped to account/API origin, and recovery exports are encrypted and verified.
- Rotated the production database-owner password through the authenticated Neon console. The replacement works on both direct and pooled connections; the previous password is rejected.
- Replaced the short production JWT secret with a cryptographically generated secret. Previous sessions are invalidated and users must sign in again.
- With explicit approval, rotated all four legacy `admin`/`admin1` passwords using bcrypt cost 12. Their roles, signing keys and message-encryption keys were preserved. A live administrator login with a replacement password returned 200.
- With explicit approval, disabled password login only for the reserved `@split` system bot. Its identity, role and content remain intact; the repaired bot service can use it without a human login credential.
- Removed real database passwords from current scripts/debug output. Old Git history can contain the revoked values; no disruptive history rewrite was performed.
- Removed the five obsolete production database secrets from GitHub. CI uses disposable PostgreSQL, and production credentials remain in Render/Neon. A separate runtime database role was not provisioned; the existing role was retained.
- Standard frontend response security headers are active, including framing restrictions, content-type protection, referrer policy and restricted camera/microphone/geolocation permissions.
- Corrected the privacy text to disclose Vercel Analytics, operational logging, metadata, recovery and retention limits accurately.

New administrator credentials and the deployment backup are kept in Windows-encrypted local storage, outside Git. Recovery instructions are provided to the operator locally; no current passwords or tokens are included in this report.

## Data preservation and free-tier controls

Before migration, a complete custom-format database backup was encrypted with AES-GCM and verified by decryption and checksum. Its encryption key is protected by Windows current-user DPAPI. Account/post/message counts were unchanged by migration: **4,085 users, 2,269 posts and 92 messages**.

Compacting the existing outbox reclaimed **182,255,616 bytes (about 174 MiB)** without deleting logical records. The primary database measured **238,600,192 bytes** afterwards. The legacy second database is retained and still counts toward the Neon project's shared storage allowance.

The hosted configuration uses one active Splitter Render web service, a four-connection database pool with zero minimum idle connections, verified database TLS, bounded page sizes and request deadlines. There is no required worker or scheduled keep-alive/population job. Guest pages avoid decorative backend traffic; unread checks pause while hidden or offline.

Images retain the 5 MiB input allowance with bounded processing, durable database storage and capped derivatives. Shared media defaults are 32 MiB total and 2 MiB per account; the primary database uses a 256 MiB **soft** growth threshold. Existing content remains readable when an upload budget is reached.

`@split` uses the configured Groq provider and an available `qwen/qwen3.8-27b` model. Its credential was verified and one bounded synthetic reply succeeded without sending user content. The old Llama model was no longer listed by the provider. Application limits remain ten attempts per day, three per account and one active request, with finite response/time limits and no automatic retries.

## Application and presentation

Navigation/deep links, feed actions, pagination, loading/error states, follow approval, account/privacy settings and mobile layout were repaired while preserving the peach/dark themes and existing features. Protected images now distinguish frontend branding from API media and authenticate private story images.

Fresh visitors see the primary instance; configured/custom peers remain supported. The retired secondary is no longer offered as a default signup destination. The landing page and README describe implemented capabilities without unsupported claims about network size, universal graph portability or peer-to-peer operation. [DEMO.md](DEMO.md) provides a short walkthrough and team-aware resume wording.

## Verification evidence

- Backend complete suite on an explicit disposable local database, command build and vet passed. The backend release CI also passed complete tests, race checks and command builds with disposable PostgreSQL.
- Frontend `npm run check` passed: 28 suites, 82 tests, two snapshots, TypeScript and optimized production build. ESLint had zero errors and 16 existing hook-dependency warnings.
- All six desktop/mobile guest-navigation, refresh/back, deep-link and login-error browser checks passed. Earlier local API checks covered posting/replies/likes, locked-follow approval and encrypted DMs.
- Production checks verified health/readiness/status, public-feed reads, CORS for the canonical frontend, unauthorized owner-route rejection, the live logo/security headers and a rotated administrator login.
- The provider's model catalog and one synthetic completion were checked. This is not a load test or an external federation compatibility certification.

Release changes are reviewable in [backend PR #5](https://github.com/EliteGamer007/splitter/pull/5), [frontend PR #1](https://github.com/EliteGamer007/Splitter-frontend/pull/1), [instance selection PR #2](https://github.com/EliteGamer007/Splitter-frontend/pull/2) and [presentation PR #3](https://github.com/EliteGamer007/Splitter-frontend/pull/3). `/api/v1/status` reports the exact running backend revision and capabilities.

## Operating limits

Free Render services sleep and can take 50 seconds or more to wake. Neon/Render allowances are shared with other databases or services in the account, and dashboard metrics can lag. The database growth threshold is cached and soft; it is not a guarantee against arbitrary traffic or every provider quota. Monitor actual usage after release.

Historical encrypted messages still need their original private keys or recovery backup. Files lost from an old ephemeral Render disk cannot be reconstructed by deployment. These repairs preserve existing stored data and keys; they do not promise exhaustive security certification or universal ActivityPub interoperability.
