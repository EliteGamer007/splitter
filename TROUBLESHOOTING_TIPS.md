# 🔧 Backend Troubleshooting Guide

Common issues and solutions for the Splitter Go backend.

## 🔴 Database Connection Issues

### "dial tcp: lookup ep-xyz...: no such host"
**Cause:** Incorrect `DB_HOST` in `.env`.
**Fix:**
- Ensure you copied the "Endpoint" correctly from Neon.
- It should look like `ep-random-name-123.region.neon.tech`.
- Do **not** include `https://` or port numbers in `DB_HOST`.

### "password authentication failed"
**Cause:** Wrong `DB_PASSWORD` or `DB_USER`.
**Fix:**
- Verify credentials against the Neon dashboard.
- Ensure there are no surrounding quotes (e.g. `DB_PASSWORD=secret`, NOT `DB_PASSWORD="secret"`).

### "App fails to connect immediately on startup"
**Check:**
- Are you behind a corporate firewall blocking port 5432?
- Is your IP allowed if you set up IP restrictions in Neon?

---

## 🟡 Environment & Runtime

### "go: go.mod requires go >= 1.21"
**Fix:** Update Go.
- **Windows**: Download MSI from [go.dev/dl](https://go.dev/dl/).
- **Mac**: `brew install go`
- **Linux**: Use your package manager or download tarball.

### "panic: runtime error: invalid memory address..."
**Cause:** Often due to missing `.env` variables causing configuration validaton to fail.
**Fix:**
- Ensure `.env` exists in the SAME directory where you run `go run`.
- Check all required fields (DB_*, JWT_SECRET, PORT).

---

## 🔵 Migration Issues

### "relation 'users' already exists"
**Cause:** You are trying to run the setup migration on a DB that is already set up.
**Fix:**
- Ignore if existing data is correct.
- OR Drop and recreate tables if you want a fresh start (WARNING: Data Loss):
  ```bash
  docker run --rm postgres:15 psql 'CONN_STRING' -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
  # Then re-run migration command
  ```
