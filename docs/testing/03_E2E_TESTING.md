# End-to-End (E2E) Testing Guide

## Overview
End-to-End tests validate the Splitter application as a complete system, mirroring real user behavior. E2E tests are located in `tests/e2e_test/`. They simulate the interactions of a full-fledged client sending requests to our endpoints.

## Setup Requirements
Unlike unit or integration tests, E2E tests require a fully functional environment:
1. Running Application Server.
2. An active, seeded Database.
3. Accessible dependent services (like caching mechanisms or message brokers).

## Running E2E Tests
To execute End-to-End tests, use the following:
```bash
go test -v ./tests/e2e_test/...
```

## Creating Scenarios
- **Real Life Workflows**: Write E2E tests based on common user flows (e.g., User registers -> verify email -> login -> create a split group).
- **Setup Scripts**: Before triggering the tests, you may need to use Seeder scripts or specialized migration data to ensure predictable test results.
- **Teardown**: E2E tests should assume control of cleaning up any state they create or run in independent ephemeral environments (like GitHub Actions workflows).
