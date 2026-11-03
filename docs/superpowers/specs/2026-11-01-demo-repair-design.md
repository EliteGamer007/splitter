# Splitter: reliable portfolio demo on free hosting

Status: approved direction with user amendments on 1 November 2026. Preserve functionality, distinctive features and appearance; one hosted instance. This document does not claim that fixes have been implemented.

## Controlling user amendments

The user approved proceeding with cost-saving repairs but explicitly prohibited sacrificing features or changing the appearance. Deploy one primary Render instance; preserve configurable federation with other peers and test it with a disposable local peer, rather than require a second hosted service. Preserve DID login, E2EE, recovery, stories, circles, moderation, federation and AI capabilities. Repair unfinished controls rather than removing them. Existing peach/dark styling is retained. Feature availability may depend on configured peers/AI credentials, but no feature is removed or silently replaced by a mock. Resource caps must preserve existing 5 MiB photo input support through bounded processing, not reject previously supported input merely to make testing easy.

## Intent and decisions

The user wants the existing team project repaired into a consistent, credible rÃ©sumÃ© demonstration: working pages, secure core behavior, reliable deployment and restrained Render/Neon usage. The user explicitly selected **free-tier hosting**. Retain Go/Echo, PostgreSQL/Neon, Next.js/Vercel, existing branding and the existing two-instance identities/URLs. The current live frontend is https://splitter-social.vercel.app.

This is a repair of the existing product, not a rewrite or a migration to another hosting platform. Preserve existing accounts, posts and encrypted messages. Do not automatically purge bots, posts, uploaded media or old encryption keys to save space. Do not promise unlimited traffic, zero cold starts, no outages or absolute quota compliance without provider usage access.

## Approaches considered

1. **Repair existing services with one default primary and an on-demand second instance (recommended).** Preserves the project and federation demonstration while avoiding unnecessary duplicate service-hours. Uses durable, strictly budgeted existing database media storage; no paid workers, new hosted databases, cron seeders or keep-alive services.
2. Restore the original always-active multi-service setup. Rejected for the selected budget: two continuously running free Render services consume approximately 1,440â€“1,488 hours/month against a shared 750-hour allowance; background workers are not free.
3. Rewrite as a mock/static showcase or move providers. Rejected because it would discard functioning backend work and fail the request to repair the real application. A small built-in read-only demo explanation may help visitors, but must not impersonate a working API.

The user selected the one-hosted-instance variation of option 1. Any earlier proposal to leave features disabled, remove controls, reject restricted follows instead of supporting requests, or reduce accepted image input to 1 MiB is superseded by the controlling amendments above. Federation works when configured; no synthetic traffic or keep-alive jobs are needed.

## Scope and acceptance criteria

### Authentication and authorization

- Verify DID challenge signatures with the account's Ed25519 public key; enforce expiry, single use, atomic consumption and suspension checks. Reject missing, malformed, wrong-user, replayed and invalid signatures.
- Remove unconditional startup admin creation/password resets, bulk encryption-key resets, silent schema changes and unbounded post deduplication. Existing admins remain unchanged. Provide an explicit controlled bootstrap command that requires supplied credentials and never logs them.
- Require a strong production JWT secret instead of a known fallback. Align configured token lifetimes; check current account status/role for sensitive actions. No unauthorized mutation may occur after a 401/403 response.
- Enforce post visibility on direct reads, author/profile lists, parent-thread hydration, replies, interactions, federation and media access. Public responses must use public user DTOs and omit email/private attributes.
- Enforce locked-account follow behavior and recipient DM policy. Either implement the existing request/approval surface end to end or explicitly reject pending follows until supported; never auto-accept a restricted request.
- Add bounded public/authenticated write limits, request/body/query timeouts and validation. Client errors must be useful without disclosing SQL, tokens, passwords or raw internal errors.

### Messaging and browser key safety

- Encrypted sends/edits must never transmit or store plaintext alongside ciphertext. Fail closed while keys are unavailable; show a clear recovery/key-mismatch state. An explicitly selected legacy plaintext conversation must be clearly labelled and never advertised as E2EE.
- Scope all local keys, timelines, threads, drafts and offline queues by backend origin plus account ID; clear sensitive session data on logout. Do not accidentally reuse another account's key material.
- Preserve legacy key material in an account-specific migration when ownership is verifiable; do not erase ambiguous legacy material silently. A fresh device must not overwrite a server encryption key automatically. Recovery/import must validate that public/private keys match the account; export must use a real encrypted file and reject weak passphrases explicitly.
- Existing static ECDH encryption is not Signal Protocol. Update labels to accurately describe its limits. Preserve ciphertext and historical keys; do not promise to recover already lost keys or plaintext already exposed historically.
- Provide real recovery controls/fingerprints, correct paired-key rotation and clear device/key mismatch UX. No fabricated security values.

### Federation and optional features

- Federation is explicit configuration and remains enabled by default to preserve the user-requested feature; only configured peers and actual user events cause traffic. When disabled, outbound hooks and inbound mutation endpoints are consistently inactive; discovery may serve accurate read-only capability information. Disabled UI features show a concise explanation.
- When explicitly enabled for a demonstration: bind HTTP signer to actor/key owner, verify body Digest, require request-target/host/date/digest signatures, enforce a bounded freshness window and replay controls, authorize ownership of deletes, respect audience visibility and recipient policy.
- Restrict remote fetches to validated HTTPS public destinations; validate all redirects and DNS addresses, bound body sizes/timeouts/concurrency. Permit only explicitly configured local test peers in development. Never allow an attacker-selected origin to receive a JWT.
- Deliver only on real user events; bounded queue/retries, no idle peer-health polling. Optional request-triggered retry work has a strict deadline and batch limit. If guaranteed offline delivery requires a worker, explain that limitation rather than silently promising it.
- AI replies and population/keep-alive schedules stay off by default. Unsupported AI providers/models are not invoked and disabled controls do not masquerade as successful actions. No new paid API or schedule is introduced.
- Story expiration, read state and cleanup work without continuously waking Neon: filter expired stories on reads, perform bounded opportunistic cleanup and document any manual cleanup command.

### Website consistency and visitor experience

- Preserve the peach/dark Splitter visual identity and existing social flows; repair layouts instead of creating an unrelated design.
- Correct landing logo navigation, browser back/forward/refresh and shareable profile/thread views using validated URL state or existing App Router patterns. Unknown destinations render a useful page rather than blank content.
- Fix instance selection in development and production. One canonical selected backend is used by every request. Do not wake the secondary backend to populate decorative counts; fetch its data only when selected/requested.
- Fix privacy-state mappings, real recovery UI, expired-token/logout behavior, permission errors, loading/empty/error states, mobile overflow, image fallback and form validation.
- Replace invented instance statistics, fake recovery/security values and placeholder links with real bounded data, honest labels or removed controls. Password reset must describe available recovery rather than claim an email was sent.
- Review landing, instance picker, signup, both login modes, home, profile/follows/circle, thread/replies, DMs, security, stories, hashtags/search, moderation, federation and admin pages. Verify unauthorized users cannot enter privileged views and server checks remain authoritative.
- Cold-start UX allows one controlled wake-up sequence (up to 90 seconds total with cancellation), no overlapping retry loops, and a retry action on failure. Never describe a sleeping service as permanent data loss.

## Resource budgets

Provider references checked 1 November 2026: [Render free services](https://render.com/docs/free), [Neon plans](https://neon.com/docs/introduction/plans). Provider limits may change. Current account usage and other projects sharing the quotas are unknown.

- Render: 750 service-hours/month shared across a workspace; free services sleep after 15 idle minutes and cannot use persistent disks or paid background workers. No keep-alive traffic. Keep the second instance demand-driven; the app cannot guarantee quota safety if users keep both awake continuously.
- Neon currently documents 100 CU-hours/project/month, 1 GB Postgres storage/project and 5 GB egress/project; branches/computes share a project's allowance. Do not assume two branch endpoints have independent budgets. Preserve scale-to-zero.
- Default DB pool: maximum 4 connections per process, minimum 0, 60-second idle lifetime, bounded acquisition/query deadlines, escaped connection parameters and a Neon-compatible safe query mode. At most 8 application connections if both instances are active. Remove unnecessary liveness/idle DB queries.
- Feed/search/thread defaults 20 items; hard maximum 50. DM pages default 30, maximum 50. Debounce search 400 ms. Bounded response bodies and remote fetches. No unbounded N+1 profile/count fetching.
- Public non-personal metadata may be cached for 60â€“300 seconds with request coalescing. Private content never enters shared/public caches. Private browser caches are account/instance scoped and cleared appropriately.
- No unread polling when hidden, offline or logged out. Visible authenticated polling at most once per 60 seconds with one request in flight, jitter/backoff and cancellation; prefer focus/user actions for refresh. No guest loop that keeps Render/Neon awake.
- Durable media: reuse database-backed bytes, not Render filesystem persistence. Enforce 1 MiB input, decoded image pixel limits, bounded decoding/concurrency and at most 256 KiB stored per image (avatar at most 128 KiB). Use accepted formats with explicit limits; reject oversized media clearly rather than exhaust memory. No video hosting.
- Stored media budget: 32 MiB per application database, 2 MiB per account, atomically enforced; configure lower limits if baseline data is already large. Fail new uploads cleanly at the cap without deleting existing media. Existing over-budget media remains readable.
- Soft application DB write budget: default 256 MiB per database, cached request-triggered size measurement, with clear read-only/quota responses for growth operations. This leaves headroom but is not a substitute for measuring full project/branch/index/history usage. A pre-rollout usage check must record actual database sizes; if already over the budget, do not purge data or silently raise the budget.
- Limits apply to bots as users. No bulk auto-registration, automated posts, mass seeding, synthetic production load or external AI calls. A tiny explicit seed fixture is only for a disposable test database.
- Build pipelines run relevant checks once per source change; exclude caches, node_modules, local uploads, secrets and audit copies from image contexts. Free blueprint contains web services only; no paid starter/worker defaults.

The application can prevent its own runaway jobs and reject excess work. It cannot enforce Render workspace service-hours, project-wide Neon CU-hours or arbitrary visitors' total bandwidth perfectly without provider credentials/metrics. Document actual usage verification and graceful provider-limit/cold-start handling; never claim an unconditional no-outage guarantee.

## Database and deployment safety

- Use a disposable PostgreSQL database for migrations, negative permission tests and complete two-instance scenarios. Docker CLI exists locally, but its engine was not running during preparation; start/resolve Docker before relying on DB test results. Never fall back to the `.env` production database for tests.
- Establish one ordered baseline/upgrade path. Migrations are transactional where supported, checksummed/versioned, serialized with a migration lock and recorded only after success. Include SQL files in the image; preserve existing applied versions and detect incompatible legacy state. Migration failure must fail deployment preparation.
- Migrations are explicit commands, not web-server startup side effects. Existing accounts/messages/posts remain intact. Take a recoverable snapshot/backup before any live schema work; use additive compatibility changes first.
- Fix current dependencies and supported Go/Node versions, keep one canonical frontend lockfile, repair lint, separate Playwright/Jest, require all meaningful suites and production builds in CI. No blanket `continue-on-error` for release checks or ignored TS errors.
- Add process-only liveness and cached/deadlined database readiness/version reporting. A probe must not create an idle DB traffic loop. UI/config advertise the current frontend/API URLs and capability flags truthfully.
- Produce reviewed deployment artifacts/configuration for the existing services. Determine actual hosting repository/branch/commit and deployment access before rollout; do not create a replacement Site or change the project's hostname.
- Verify staging/local completion, then roll out backward-compatible backend changes before the frontend. Use the provider's existing deployment mechanism, verify deployed commit IDs/health, and perform limited disposable-account live smoke tests with cleanup if authorized. Do not expose credentials in logs or commit environment files.
- If hosting credentials are unavailable, complete local fixes, tested artifacts and exact rollout instructions; identify the access needed honestly instead of claiming the live site changed.

## Verification gates and deliverables

1. Regression tests fail first for the actual audit defects; prove rejected auth/privacy/admin/federation operations cause no mutation.
2. Empty DB and existing-schema upgrades, rollback-on-error and repeated migrations pass against disposable Postgres. Both instance configurations have isolated test data.
3. Focused/full Go tests, vet, builds, race checks on relevant shared state; frontend lint, explicit typecheck, real component tests, Playwright browser flows and production build pass. Dependency advisories are updated/triaged by actual applicability.
4. Browser checks cover guest and authenticated flows on desktop/mobile widths, navigation/refresh, empty/offline/cold-start states, two accounts on the same browser, fresh-device key mismatch, forbidden private post/media reads and moderation roles.
5. Resource tests prove pool caps, query timeouts, pagination caps, upload quotas under concurrent requests, request cancellation and no polling while hidden/logged out. A small controlled local concurrency run records latency, memory and query counts; no production load tests.
6. Deliver final changed-file summary, test evidence, resource settings, known provider constraints, remaining credentials/rollout steps and updated deployment/status docs. Report which fixes are local versus actually deployed.

## Builder discretion and boundaries

Follow existing repository conventions; split oversized handlers only when it helps tested repair. Introduce small repository interfaces where needed for handler-level tests. Avoid broad dependency-heavy rewrites. Implementation may be phased by security, resources/migrations, UI and release readiness, but each phase must keep a running app and the full accepted scope remains outstanding until verified.

Exact internal helper names, test fixture organization, layout spacing and dependency patch versions are implementer decisions. No destructive production cleanup, paid plan changes, new providers, disabling core security checks to obtain green tests, or claims that historical ciphertext is recoverable without its keys.
