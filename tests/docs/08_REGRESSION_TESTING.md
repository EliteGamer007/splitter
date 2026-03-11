# Regression Testing Guide

## Overview
Regression testing ensures that new application features (like the new ephemeral stories feature) do not inadvertently break existing, working functionality. The script `scripts/run_regression_tests.ps1` executes all test categories in a specific order, acting as the primary CI/CD safety mechanism.

Integration tests may show intermittent failures in CI environments. This occurs when newly inserted database records are not immediately visible across pooled database connections. These failures are related to database connection pooling behavior rather than application logic errors.

## Categories Executed:
1. Unit
2. Database (db, check_schema, verify_db, fix_db)
3. User Validation (check_users)
4. Encryption (check_encryption_keys)
5. Initialization (apply_migration, seeder)
6. Workflows (integration, e2e_test, load)

## Sample Regression Output Summary

```text
--- Splitter Regression Test Suite ---
Started at: 2026-03-10 22:13:06

[2026-03-10 22:13:06] Running Schema tests (./tests/check_schema)...
ok  	splitter/tests/check_schema	5.763s
[2026-03-10 22:13:13] Schema tests PASSED.

[2026-03-10 22:13:21] Running Encryption tests (./tests/check_encryption_keys)...
ok  	splitter/tests/check_encryption_keys	0.832s
[2026-03-10 22:13:23] Encryption tests PASSED.

[2026-03-10 22:13:48] Running Integration tests (./tests/integration)...
ok  	splitter/tests/integration	(cached)
[2026-03-10 22:13:49] Integration tests PASSED.

[2026-03-10 22:13:49] Running E2E tests (./tests/e2e_test)...
ok  	splitter/tests/e2e_test	3.934s
[2026-03-10 22:13:54] E2E tests PASSED.

Splitter Regression Test Suite Completed at 2026-03-10 22:13:56
```
