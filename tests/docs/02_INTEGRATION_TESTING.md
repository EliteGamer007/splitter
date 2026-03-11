# Integration Testing Guide

## Overview
Integration testing in Splitter validates the complete request lifecycle: from the HTTP API boundary through the router, into the service layer, writing to the PostgreSQL database, and triggering the asynchronous Federation mock server.

**Implemented Components Interacted With:**
- `TestServer` (HTTP API)
- `db.DB` (Neon PostgreSQL via pgBouncer)
- `Federation Simulator` (Mocks ActivityPub HTTP deliveries)

## Environment Stability Note
Some integration tests depend on asynchronous behavior and database connection pooling. When using serverless PostgreSQL providers such as Neon, tests may show occasional non-deterministic results due to connection pooling, transaction propagation delays, and async federation events. This behavior does not affect application correctness.

## Example: User Registration Flow

**Test Case:** Register User
**Description:** Verify that a new user without an existing email or username can successfully register and receive a JWT token.
**Expected Result:** HTTP 201 Created

**Real Test Output Snippet:**
```json
=== RUN   TestAuthFlow/Register_User
{"time":"2026-03-10T22:28:35.2718607+05:30","id":"","remote_ip":"127.0.0.1","host":"127.0.0.1:53051","method":"POST","uri":"/api/v1/auth/register","user_agent":"Go-http-client/1.1","status":201,"error":"","latency":904767800,"latency_human":"904.7678ms","bytes_in":101,"bytes_out":807}
```
