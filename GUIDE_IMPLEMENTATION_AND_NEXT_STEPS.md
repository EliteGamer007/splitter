# Splitter — Implemented Details and Next Improvements (Guide Notes)

## 1) What is already implemented

### Federation and Interoperability
- ActivityPub + WebFinger baseline is implemented:
  - WebFinger lookup for remote handles
  - Actor resolution and caching (`remote_actors`)
  - Inbox endpoints (`/ap/users/:username/inbox`, `/ap/shared-inbox`)
  - Outbox endpoint (`/ap/users/:username/outbox`)
- Signed federation traffic is implemented:
  - Outgoing HTTP signatures
  - Incoming signature verification
- Delivery reliability is implemented:
  - Outbox persistence + retry metadata
  - Exponential backoff retry logic
  - Circuit breaker controls for unstable domains
- Federation safety/governance is implemented:
  - Blocked domains enforcement (inbox + outbox)
  - Deduplication of incoming activities
- Federation content behavior implemented:
  - Federated likes/reposts
  - Federated profile updates and key updates
  - Federated delete propagation and remote tombstone handling
  - Remote parent-thread context fetch/cache on demand

### Identity, Privacy, and Security
- Password + DID-oriented identity model implemented
- Client-side key generation/storage flow implemented
- Encrypted recovery export/import validation implemented
- E2EE DM foundation implemented with edit/delete windows
- Role-based admin/mod authorization and audit logs implemented

### Admin and Observability
- Moderation queue actions implemented (approve/remove/warn)
- Federation inspector dashboard implemented
- Reputation scoring + failing-domain metrics implemented
- Federation network graph API + UI implemented

---

## 2) Important improvements still needed

### High priority (should do soon)
- **Cross-instance compatibility hardening (especially Mastodon ecosystem):**
  - Normalize more ActivityPub payload variants
  - Improve object type coverage and edge-case parsing
  - Add stricter compatibility tests against non-local instances
- **Federation robustness tests:**
  - Replay, malformed payload, timeout, partial failure, and recovery scenarios
- **Inbox abuse controls:**
  - Per-domain/per-actor request rate limits
  - Better suspicious-activity detection and telemetry

### Medium priority
- **Ephemeral post lifecycle completion:**
  - Worker cleanup + timeline filtering by expiry
- **Privacy model completion:**
  - Persist and enforce all privacy defaults consistently across all flows
- **Federated messaging hardening:**
  - Complete encrypted cross-instance DM protocol path end-to-end

---

## 3) AI bot novelty for populating Trending

## Is it possible soon?
**Yes — feasible in a short implementation cycle**, and it is compatible with current app architecture because bots can use the same existing user/post/interaction pathways.

## Compatibility with this app
- Very compatible if bots are treated as normal users:
  - Create bot accounts in existing users table
  - Use existing post creation + interaction APIs
  - Trending gets populated naturally from bot activity volume
- Optional federation mode:
  - Keep bots local first (simpler)
  - Later allow controlled federated bot posting if needed

## Recommended implementation approach
### Phase 1 (fast)
- Add bot account seeding and scheduler (worker job)
- Generate posts/replies/likes at configurable intervals
- Use topic pools + templates initially
- Mark bot-origin content with metadata for transparency in admin tools

### Phase 2 (AI-powered)
- Add LLM content generation adapter with:
  - Prompt templates per bot persona
  - Output length/style constraints
  - Safety filter and banned-topic checks
- Add interaction policy engine:
  - Which bot interacts with which content
  - Cooldowns to avoid unrealistic bursts

### Phase 3 (quality + controls)
- Add admin controls:
  - Enable/disable bots
  - Intensity level (low/medium/high activity)
  - Topic profiles
- Add analytics:
  - Bot contribution to trending score
  - Ratio of human vs bot interactions

---

## 4) API recommendation for bot content generation

### Best practical choice right now
- **OpenAI API (small model tier)** for reliability, quality, and easy JSON-structured outputs.

### Why this is a good fit
- Strong instruction following (useful for persona-stable bot output)
- Good latency/cost balance on smaller models
- Easy server-side integration and moderation hooks

### Alternatives
- **Local model (Ollama):** lowest running cost, more infra effort, weaker consistency depending on model
- **Other hosted APIs (Together/Groq/Anthropic/etc.):** can work, but expect adapter changes and varying output consistency

---

## 5) Guardrails to include from day 1 (important)
- Clearly tag bot accounts internally (and optionally in UI)
- Apply content moderation filters before publish
- Add rate limits to prevent spam-like flooding
- Keep audit logs of bot-generated actions
- Make bot simulation toggleable for demo vs production mode

---

## 6) Suggested immediate next sprint scope
1. Mastodon-compatibility test matrix + parser hardening
2. Federation replay/failure test suite expansion
3. Bot framework (local bots + scheduler + templates)
4. Optional AI adapter behind feature flag for controlled rollout

---

## 7) Mandatory course requirements to include (H3 compliance)

### Team and process constraints
- Team size must be **4–5** members (6 only with justification + approval)
- Version control must show:
  - Branching + merging discipline
  - Meaningful commit messages and contribution traceability
- CI/CD pipeline is **mandatory** (GitHub Actions/GitLab CI/Jenkins etc.)

### Deployment constraints
- Cloud deployment is **mandatory** (AWS/Azure/GCP/Render/Railway/DigitalOcean or equivalent)
- On-prem deployment is only acceptable with strong technical justification

### Sprint review expectations
- **Sprint 0**: inception artifacts (problem framing, personas, epics/stories, architecture, stack, deployment/workflow decisions)
- **Sprint 1**: ~40% deployed build on cloud + initial test evidence + monitoring visibility
- **Sprint 2**: near-complete features + testing coverage + performance evidence + deployment/documentation readiness + live demo

### Required technical evidence
- UML/architecture diagrams, ERD, API specs, design trade-off justifications
- Testing evidence: unit, integration, usability, accessibility, regression, performance
- Observability: logging, exception handling, uptime/resource visibility
- Documentation: install/deploy guide, release notes, user manual, environment details
- Scalability/performance reasoning and reflection

---

## 8) Correct implementation order from now (recommended)

### Phase A — Governance and delivery foundation (start immediately)
1. **CI/CD now**: lint/test/build workflows + branch protection + PR checks
2. **Cloud deployment baseline**: deploy backend/frontend + managed DB + environment secrets
3. **Observability baseline**: centralized logs, health checks, uptime monitor, error tracking

### Phase B — Compliance closure
4. Complete documentation pack (architecture/UML/ERD/API/release notes/user guide)
5. Expand test matrix (regression + compatibility + perf smoke + accessibility/usability evidence)
6. Validate sprint evidence mapping (Sprint 0/1/2 rubric to repo artifacts)

### Phase C — Feature extensions / novelty
7. Finish remaining high-impact gaps (interop hardening, abuse controls, expiry lifecycle)
8. Implement bot simulation framework (template-based first)
9. Enable AI-content mode behind feature flag + moderation/rate guardrails

---

## 9) Direct answers to guide questions

### Should CI/CD start right now?
**Yes. Start now.**
- It is mandatory and should gate all later work.
- It reduces risk before adding novelty features like AI bots.

### Should deployment happen before AI bots?
**Yes. Deploy first, then bots.**
- Required by course policy.
- Better to validate stability/observability on cloud before load-simulation features.

### Is Dockerized multi-instance deployment accepted?
**Yes, if the containers run on cloud infrastructure.**
- Example accepted pattern:
  - Backend instance-1 + backend instance-2 as separate services/containers
  - Frontend service
  - Managed PostgreSQL
  - Reverse proxy / domain routing per instance
- Not sufficient if only local Docker is used for final submission.
- On-prem container hosting needs explicit technical justification and approval.

---

## 10) Practical cloud deployment pattern for Splitter instances

### Minimal accepted architecture
- `splitter-instance-1` service (own env + federation domain)
- `splitter-instance-2` service (own env + federation domain)
- `splitter-frontend` service
- Managed Postgres (single DB with isolation strategy or separate DBs)
- CI/CD pipeline deploying on merge to main/release

### Operational checks before demo
- Health endpoint checks for all instances
- Federation smoke tests between cloud instances
- Uptime dashboard screenshots
- Logs showing federation deliveries/retries/signature verification

---

## 11) AI bot novelty — feasibility and API choice (updated)

### Can this be implemented soon and remain compatible?
**Yes.**
- Fast path: treat bots as standard users and use existing post/interaction APIs.
- This is fully compatible with current architecture and trending logic.

### Recommended API
- **OpenAI API (small model tier)** as first choice for quality + stable structured output.

### Implementation sequence for bots
1. Template/non-AI bot generator (quick population)
2. Scheduler + rate control + persona definitions
3. Optional AI generation adapter (feature flag)
4. Safety/moderation + audit logging + admin controls

### Important caution
- Keep bots clearly tagged internally.
- Do not let bot traffic hide real-user behavior in analytics; track bot/human ratios separately.



# CI/CD Setup and Runbook (Splitter + Splitter-frontend)

This document explains what has already been implemented and what manual setup is still required.

## 1) Implemented in repository

### Backend repo (`splitter`)
- GitHub Actions CI workflow: `.github/workflows/backend-ci.yml`
  - gofmt check
  - go vet (`cmd` + `internal`)
  - binary build checks (`server`, `worker`, `migrate`)
  - unit-focused tests (`internal/federation`, `tests/unit/...`)
  - optional DB tests (`tests/db`) when DB secrets are configured
  - optional integration tests (manual trigger only)
- Deploy template workflow: `.github/workflows/backend-deploy-template.yml`
  - deploy hook trigger for instance 1
  - deploy hook trigger for instance 2
- Containerization artifacts:
  - `Dockerfile`
  - `docker-compose.instances.yml` (2 backend instances + frontend service)

### Frontend repo (`Splitter-frontend`)
- GitHub Actions CI workflow: `.github/workflows/frontend-ci.yml`
  - `npm ci`
  - best-effort lint (skips if eslint dependency missing)
  - stable gating test (`__tests__/lib/api.test.ts`)
  - full test run as non-blocking diagnostics
  - production build
- Deploy template workflow: `.github/workflows/frontend-deploy-template.yml`
- Containerization artifact: `Dockerfile`

### GHCR image publish workflows
- Backend image publish: `.github/workflows/backend-ghcr-publish.yml`
- Frontend image publish: `.github/workflows/frontend-ghcr-publish.yml`
- Trigger: push to `main` or manual dispatch
- Registry: `ghcr.io/<owner>/splitter-backend` and `ghcr.io/<owner>/splitter-frontend`

---

## 2) Manual setup required (mandatory)

## A. Enable branch protection (GitHub UI)
For both repos:
1. Settings → Branches → Add branch protection rule for `main`
2. Enable:
   - Require pull request before merging
   - Require status checks to pass before merging
   - Require branches to be up to date before merging
3. Select checks:
   - backend repo: `quality`, `build`, `unit-tests`
   - frontend repo: `test-build`

## B. Add GitHub Secrets

### Backend repo secrets (`splitter`)
Required for optional DB tests:
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

Required for deploy workflow:
- `BACKEND_INSTANCE1_DEPLOY_HOOK_URL`
- `BACKEND_INSTANCE2_DEPLOY_HOOK_URL`

### Frontend repo secrets (`Splitter-frontend`)
Required for deploy workflow:
- `FRONTEND_DEPLOY_HOOK_URL`

### GHCR secrets
- No extra secret is required for same-repo GHCR publish; workflows use `GITHUB_TOKEN` with `packages: write`.

---

## 3) Manual secret setup commands (GitHub CLI)

> Run these separately in each repo root after authenticating `gh auth login`.

### Backend (`splitter`)
```bash
gh secret set DB_HOST
gh secret set DB_PORT
gh secret set DB_USER
gh secret set DB_PASSWORD
gh secret set DB_NAME

gh secret set BACKEND_INSTANCE1_DEPLOY_HOOK_URL
gh secret set BACKEND_INSTANCE2_DEPLOY_HOOK_URL
```

### Frontend (`Splitter-frontend`)
```bash
gh secret set FRONTEND_DEPLOY_HOOK_URL
```

---

## 4) How to use the workflows

## Automatic CI
- Triggered on push/PR to `main` and `develop`.

## Manual integration run (backend)
- Actions → `Backend CI` → `Run workflow`
- Set `run_integration=true`
- Requires DB secrets to be configured.

## Deploy trigger
- Backend: push to `main` or run `Backend Deploy (Template)` manually
- Frontend: push to `main` or run `Frontend Deploy (Template)` manually

## GHCR publish trigger
- Backend: push to `main` or run `Backend GHCR Publish` manually
- Frontend: push to `main` or run `Frontend GHCR Publish` manually

After first publish, in GitHub Packages:
1. Set package visibility as needed (private/public)
2. If deploy platform pulls private GHCR images, configure platform credentials/token

---

## 5) Cloud deployment recommendations (accepted approach)

Using Dockerized instance servers is accepted **if deployed on cloud infrastructure**.

Recommended topology:
- `splitter-instance-1` service/container
- `splitter-instance-2` service/container
- `splitter-frontend` service/container
- Managed Postgres (Neon/RDS/etc)

Good platforms:
- Render / Railway / DigitalOcean Apps / AWS ECS / Azure Container Apps / GCP Cloud Run

---

## 6) Local dry-run commands

### Backend local CI-equivalent
```bash
cd splitter
go mod download
gofmt -l .
go vet ./cmd/...
go vet ./internal/...
go build ./cmd/server ./cmd/worker ./cmd/migrate
go test -v ./internal/federation ./tests/unit/...
```

### Frontend local CI-equivalent
```bash
cd Splitter-frontend
npm ci
npm run lint
npm test -- --ci --watchAll=false
npm run build
```

### Local multi-instance container run
```bash
cd splitter
docker compose -f docker-compose.instances.yml up --build
```

---

## 7) Suggested immediate activation order
1. Push workflows to both repos
2. Configure secrets
3. Enable branch protection
4. Verify CI green on test PR
5. Add cloud deploy hooks
6. Verify deployment from `main` merge
7. Capture screenshots/logs for sprint evidence

