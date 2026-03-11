# Database Testing Guide

## Overview
Database operations are central to Splitter functionality. Database tests ensure schema integrity, migration correctness, and proper execution of queries or stored procedures. These tests, along with various DB utilities, are located in the `tests/db`, `tests/check_schema`, and `tests/apply_migration` directories.

## Setup
When running Database-specific tests, make sure to use a test database separate from production or staging.
You can configure the test database connection string by setting a test-oriented `.env` file before executing tests.

## Running Database Tests
To execute all database-related tests safely:
```bash
go test -v ./tests/db/...
```

## Schema and Migration Validation
Before pushing database alters, ensure migrations and schemas are solid:
- **`tests/check_schema/`**: Contains schema integrity checks to verify constraints and associations automatically.
- **`tests/apply_migration/`**: Validates whether migration up/down scripts complete without errors. Run these tests on a fresh database environment.

## Database Utilities
- Use scripts within `tests/fix_db` to handle known desynchronizations in the local development environment.
- Use `tests/verify_db` scripts periodically to check against drift and potential anomalies.
