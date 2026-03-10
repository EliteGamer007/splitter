# Unit Testing Guide

## Overview
Unit tests ensure that individual components and functions of the Splitter application work correctly in isolation from the rest of the system. In Go, these tests are primarily kept in the dedicated `tests/unit` directory.

## Structure
- `tests/unit/`: Contains isolated tests for internal services, utilities, and helper functions.
- Test files should follow the Go testing convention: `*_test.go`.

## Running Unit Tests
You can run all tests across the project using the Makefile:
```bash
make test
```

To run only the unit tests explicitly located in the unit directory:
```bash
go test -v ./tests/unit/...
```

To run with coverage, use:
```bash
make test-cover
```

## Best Practices
1. **Mock Dependencies**: Use interfaces and mocks to bypass actual database calls and third-party APIs.
2. **Table-Driven Tests**: Group similar test cases in slices (table-driven testing) to keep code clean and maintainable.
3. **Coverage Requirement**: Ensure business logic, controllers, and services maintain high coverage before committing.
