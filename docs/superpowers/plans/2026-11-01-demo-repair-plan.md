# Splitter Repair Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement the approved repair task-by-task. Independent bounded domains may use superpowers:dispatching-parallel-agents with non-overlapping file ownership; review integration before release.

**Goal:** Repair the existing application into a functioning, secure and visually consistent résumé demo on one hosted Render instance without sacrificing features.

**Architecture:** Retain Go/Echo, PostgreSQL/Neon and Next.js/Vercel. One production API is the default; configurable federation peers remain supported and are exercised locally. Durable capped media, small pools, request-driven work and cancellation protect free-tier resources.

**Tech Stack:** Go, Echo, pgx/PostgreSQL, Next.js/React, Jest, Playwright, Docker.

**Spec:** docs/superpowers/specs/2026-11-01-demo-repair-design.md; its controlling user amendments override earlier proposals.

## Global constraints

- One hosted instance; no paid workers, keep-alive schedules, unsolicited provider changes or feature cuts.
- Preserve existing accounts/data/ciphertext/key material and the existing visual identity.
- Existing 5 MiB image inputs remain accepted with bounded decoding/storage; no video hosting added.
- Never run tests against the root `.env` database. Product startup must not reset credentials/keys or mutate schema.
- No secrets in source/logs/tool output. Existing documentation edits and untracked user files remain intact.

## Review focus

- Invalid/replayed DID proofs and denied moderators must cause no mutation (Task 1).
- Same-browser account switches and new devices must not leak private caches or overwrite keys (Task 4).
- Private posts/replies/media must remain private through alternate read/interaction/federation paths (Tasks 2, 3).
- Restart/migration failures must preserve existing accounts, media and encrypted data (Task 5).
- Slow services, hidden tabs and concurrent uploads must not trigger runaway polling, quota races or blank screens (Tasks 5, 6).

## Task 1: authentication and administrator boundaries

Files: internal/handlers/auth_handler.go, admin_handler.go; internal/middleware/auth.go; internal/auth/token.go; new handler tests. Keep production constructors compatible, add narrow repository interfaces for tests.
- [ ] Write regression tests for valid/invalid/missing/replayed/expired/wrong-user signatures, suspended users and moderator rejection with no downstream write; run and observe failure.
- [ ] Verify Ed25519 proof with stored public key; atomically consume successful challenges; bound in-memory state and cleanup; honor current status/role and consistent token lifetime; return real errors from access guards.
- [ ] Run handler/auth/security tests and vet; inspect diff.

## Task 2: privacy across all backend read/write surfaces

Files: internal/repository/{post,reply,user,follow,interaction,message,circle}_repo.go; internal/handlers/{post,user,follow,reply,interaction,message,media}_handler.go; models/user.go; migrations for follow requests if needed.
- [ ] Write failed tests covering direct/profile/thread/circle/private media access, public DTO emails, suspended/demoted roles, locked follow requests/approval and DM recipient policy.
- [ ] Implement a reusable viewer-aware visibility guard and query filters; safe public DTOs; complete pending follow request handling; encrypted writes reject mixed/plaintext leakage and preserve legacy read compatibility.
- [ ] Exercise positive/negative roles, participant ownership, audience and empty-page behavior in disposable DB and handler tests; no real data changes.

## Task 3: working and bounded federation

Files: internal/federation/{http_signatures,resolver,delivery,worker,health}.go; internal/handlers/{inbox,actor,webfinger,federation,outbox}_handler.go; router wiring coordinated with Task 5.
- [ ] Add failing signature/body/date/replay/actor-ownership/delete/SSRF/redirect/visibility tests.
- [ ] Enforce signed headers and digest/date bounds; actor/key binding; safe public outbound HTTP client; bounded deliveries/retries; retain peer features and configure enabled behavior accurately.
- [ ] Use a disposable local peer to verify public post/follow/accept/like/undo/delete and recipient policy. Disabled configuration fails mutation endpoints clearly rather than leaving them unprotected.

## Task 4: encrypted messaging and account-scoped browser state

Files: frontend lib/{crypto,api,offlineMessages}.ts (discover actual offline helper), LoginPage.jsx, SignupPage.jsx, DMPage.jsx, SecurityPage.jsx, app/page.tsx auth/cache boundaries; new real-component tests.
- [ ] Add failing tests for plaintext-free encrypted send/edit, pending/error send, two-account key isolation, fresh-device mismatch, valid recovery, weak export password and paired-key rotation.
- [ ] Introduce account/origin-scoped key/cache helpers with verifiable legacy migration and no silent public-key replacement; real encrypted recovery export/import and fingerprints; preserve all messaging/security controls.
- [ ] Verify roundtrip encryption/edit/recovery, offline queue integrity, logout and privacy-state mappings in component/integration tests. Preserve styling.

## Task 5: resource limits, startup, schema and durable media

Files: config/config.go, db/postgres.go, cmd/{server,migrate,worker}/main.go, new cmd/bootstrap-admin; migrations; service/{storage,image,story}_service.go; story repo/handler; helpers/pagination.go; router/server settings; Dockerfile, docker-compose, render blueprint and .env.example.
- [ ] Add failing tests for production config, pool caps, startup without destructive changes, transaction rollback, pagination, image validation/quotas and story persistence/read state.
- [ ] Use max four/min zero DB connections with idle/query/acquisition deadlines, safe connection encoding/protocol; remove startup account/key/schema resets; explicit idempotent migrations with failure rollback and packaged SQL; explicit admin bootstrap.
- [ ] Store bounded media bytes durably with per-account/database quotas atomically; preserve 5 MiB input and supported photo formats with bounded pixels; expiry filters and request-driven cleanup.
- [ ] Configure one free web service, truthful version/readiness, no background DB churn; validate empty/upgraded DBs, concurrent uploads and restart persistence.

## Task 6: consistent complete website and single-instance default

Files: frontend app/page.tsx, lib/api.ts, navigation/state helpers, HomePage.jsx, ThreadPage.jsx, ProfilePage.jsx, InstancePage.jsx, LandingPage.jsx, SignupPage.jsx, component styles and all other page surfaces touched by verified bugs.
- [ ] Add failing real-component/browser tests for logo navigation, URL refresh/back, actual instance requests, private caches, hidden/offline polling, empty/error/cold-start states and mobile overflow.
- [ ] Default to the primary instance without probing/waking the old secondary; retain configured/custom instance selection for federation. Repair links, state mapping, pagination, image fallbacks and accessible controls using existing appearance.
- [ ] Limit visible authenticated unread polling to at most once/60s, cancel on hidden/offline/logout, coalesce reads and apply bounded retry. Preserve all features and avoid fabricated statistics.
- [ ] Browser-check every page at desktop/mobile sizes using disposable accounts and role fixtures. Fix concrete regressions, not only test mocks.

## Task 7: dependencies, CI and release checks

Files: go.mod/go.sum; frontend package.json/package-lock.json, remove redundant lockfile when safe; jest config/setup, new Playwright/eslint config; workflows, next config and tests.
- [ ] Upgrade affected dependencies to compatible supported patches and remove obsolete indirect middleware dependency where possible; validate Go/Node floors. Keep meaningful existing tests.
- [ ] Separate runners, fix stale mocks, install declared lint/browser-test tooling, enable type validation and make complete meaningful tests/builds release gates.
- [ ] Run Go tests/vet/build/race subset, frontend lint/typecheck/Jest/Playwright/build, vulnerability scans and small local resource probes. Record exact commands/results.

## Task 8: deployment and final independent review

Files: README/DEPLOYMENT/OPS and status report; release evidence under ignored audit/artifact folder.
- [ ] Independently review security, feature preservation, appearance and free-tier behavior against the spec; address all actionable findings and rerun affected gates.
- [ ] Establish provider access, actual usage and deployed repo/commit. Prepare tested artifacts; use backup and additive schema rollout before deploying backend/frontend. Never claim a live change when only local source changed.
- [ ] Verify current URL, readiness/version, static/media persistence, login/post/reply/encrypted DM/roles and configured federation; no mass production load or bot seeding.
- [ ] Update docs and report with verified feature behavior, exact resource settings, cold-start limits, deployment outcome and any external access still needed.

Completion requires the implemented scope and evidence, not simply a successful build. No automatic commit/push of the nested frontend or user files; use repository-safe changes and review before integration.
