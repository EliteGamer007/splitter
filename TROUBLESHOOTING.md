# 🔧 Troubleshooting & FAQ

This guide provides solutions for common issues and answers frequently asked questions about Splitter.

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
**Fix:** Update Go to the latest version from [go.dev/dl](https://go.dev/dl/).

### "panic: runtime error: invalid memory address..."
**Cause:** Often due to missing `.env` variables causing configuration validation to fail.
**Fix:**
- Ensure `.env` exists in the SAME directory where you run `go run`.
- Check all required fields (DB_*, JWT_SECRET, PORT).

---

## 🔵 Federation & ActivityPub FAQ

### Q: Why can't I see posts from a remote instance?
**A:** Check the following:
1. **Connectivity**: Ensure the remote server is reachable from your instance.
2. **Signature Validation**: If signatures fail, activities are dropped. Sync clocks (NTP) to ensure `Date` headers are valid.
3. **Domain Block**: Check if the instance domain is in the `blocked_domains` list in your Admin dashboard.

### Q: Does Splitter support private instances?
**A:** Yes, by disabling open registration and setting up `authorized_fetch` mode (planned feature), you can limit access.

---

## 🔐 Authentication & DID FAQ

### Q: I lost my private key. Can I recover my account?
**A:** Only if you have the **encrypted recovery file** downloaded during setup. Without it, decentralized identities cannot be restored as the server does not store your private keys.

### Q: Why does DID login fail on a new device?
**A:** You must first **Authorize** the new device using your primary device's "Device Management" settings. This links the new device's encryption keys to your identity.

---

## 🔵 Migration Issues

### "relation 'users' already exists"
**Cause:** You are trying to run the setup migration on a DB that is already set up.
**Fix:** Ignore if data is correct, or drop the public schema to start fresh (Data will be lost).
