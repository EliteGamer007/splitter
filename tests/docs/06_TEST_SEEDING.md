# Test Seeding Guide

## Overview
Properly seeded data ensures consistency when running End-to-End or specific Integration scenarios. For Splitter, test seeders are stored and run from the `tests/seeder/` directory.

## Best Practices for Seeding
- **Idempotency**: Seed scripts should be perfectly reproducible and, ideally, idempotent. Repeating the script execution should reset the state to what it needs to be, preventing dirty states between test runs.
- **Environment Targeting**: Only seed data into testing databases (`TEST_DB_NAME`) or local non-sensitive dev databases. Never run seeders linked to production endpoints or credentials.
- **Reference Integrity**: Seed scripts must populate base data (like standardized Currencies, minimal User profiles, Mock Groups) before establishing relationships (like User expenses in a specific Group).

## Example Seeding Workflow
To manually apply seeders:
```bash
go run tests/seeder/main.go
```
Or you can import the helper packages directly within `e2e_test` Setup functions to seed right before hitting endpoints.

## Automating Seeding
Most End-to-End Github Actions jobs will:
1. Initialize the PostgreSQL test container.
2. Apply schemas from `migrations/`.
3. Invoke the seeder.
4. Run integration and E2E endpoints.
