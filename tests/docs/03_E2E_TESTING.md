# End-to-End (E2E) Testing Guide

## Overview
End-to-End testing performs cross-domain functional tests simulating real client usage. E2E tests are located in `tests/e2e_test/`. They simulate the interactions of a full-fledged client sending requests to our endpoints.

## Setup Requirements
Unlike unit or integration tests, E2E tests require a fully functional environment:
1. Running Application Server.
2. An active, seeded Database.
3. Accessible dependent services (like caching mechanisms or message brokers).

## E2E Flow Example: Public Feed Validation

**Test Scenario:** `TestE2EPublicFeed`
**Steps:**
1. A user establishes a session (Registers and logs in).
2. The user executes a `POST /api/v1/posts` to create a new generic text post.
3. The user executes a `GET /api/v1/posts/public` and confirms the feed contains their previously created post.

**Expected Output:** HTTP 200 OK with the array of posts containing the target message ID.

**Actual Output Snippet:**
```text
=== RUN   TestE2EPublicFeed
--- PASS: TestE2EPublicFeed (0.19s)
PASS
ok  	splitter/tests/e2e_test	2.229s
```
