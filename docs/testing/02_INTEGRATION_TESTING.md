# Integration Testing Guide

## Overview
Integration tests ensure that different parts of the Splitter application work together correctly, particularly focusing on interactions with the database, cache, and external APIs. These tests are located in `tests/integration/`.

## Prerequisites
Before running integration tests, ensure that your local testing database (like PostgreSQL) is running or Docker is available to spin up test containers automatically.

## Running Integration Tests
To execute all integration tests:
```bash
go test -v ./tests/integration/...
```

## Writing Integration Tests
1. **Test Database Setup**: Use a dedicated clean test database. Run migrations before tests or use libraries like `testcontainers-go` to spin up isolated databases.
2. **Setup and Teardown**: Always clean up state after a test runs to prevent pollution of subsequent tests.
3. **Environment Variables**: Configure test-specific `.env` or use inline environment variables so production credentials are not used.

## Focus Areas
- API endpoint handlers combined with real database queries.
- Authentication and Authorization flows.
- Complex transactions that touch multiple tables.
