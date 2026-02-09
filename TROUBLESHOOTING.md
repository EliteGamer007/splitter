# Troubleshooting Guide

## Overview

This guide provides solutions to common issues encountered when setting up, developing, and deploying the Splitter application. Use this as a first reference when encountering problems.

---

## Common Setup Issues

### Issue: Go Version Mismatch

**Symptoms:**
```
go: go.mod requires go >= 1.21, but go version is 1.20
```

**Solution:**
```bash
# Check current Go version
go version

# Download and install Go 1.21+ from https://go.dev/dl/
# On Windows: Use the MSI installer
# On macOS: Use Homebrew
brew install go@1.21

# On Linux: Download and extract
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Verify installation
go version
```

---

### Issue: Missing `.env` File

**Symptoms:**
```
Error: DATABASE_URL environment variable not set
panic: runtime error: invalid memory address
```

**Solution:**
```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your actual credentials
# Required variables:
# - DB_HOST
# - DB_PORT
# - DB_USER
# - DB_PASSWORD
# - DB_NAME
# - JWT_SECRET
```

**Example `.env` configuration:**
```env
DB_HOST=ep-your-endpoint.region.aws.neon.tech
DB_PORT=5432
DB_USER=your_neon_username
DB_PASSWORD=your_neon_password
DB_NAME=neondb

PORT=8000
ENV=development
BASE_URL=http://localhost:8000
JWT_SECRET=your-secret-key-change-this-in-production
```

---

### Issue: Node.js Version Incompatibility

**Symptoms:**
```
error: The engine "node" is incompatible with this module
```

**Solution:**
```bash
# Check Node.js version
node --version

# Install Node.js 18+ using nvm (recommended)
# On Windows: Download from https://nodejs.org/
# On macOS/Linux:
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
nvm use 18

# Verify installation
node --version
npm --version
```

---

## Dependency Errors

### Issue: Go Module Download Failures

**Symptoms:**
```
go: github.com/labstack/echo/v4@v4.11.0: Get "https://proxy.golang.org/...": dial tcp: lookup proxy.golang.org: no such host
```

**Solution:**
```bash
# Option 1: Clear Go module cache
go clean -modcache
go mod download

# Option 2: Set Go proxy (if behind firewall)
go env -w GOPROXY=https://goproxy.io,direct

# Option 3: Use direct mode (bypass proxy)
go env -w GOPROXY=direct

# Retry
go mod tidy
go mod download
```

---

### Issue: Frontend Dependency Installation Fails

**Symptoms:**
```
npm ERR! code ERESOLVE
npm ERR! ERESOLVE unable to resolve dependency tree
```

**Solution:**
```bash
# Option 1: Clear npm cache
npm cache clean --force
rm -rf node_modules package-lock.json
npm install

# Option 2: Use legacy peer deps (if needed)
npm install --legacy-peer-deps

# Option 3: Update npm
npm install -g npm@latest
```

---

### Issue: Missing PostgreSQL Client Tools

**Symptoms:**
```
psql: command not found
```

**Solution:**
```bash
# Use Docker instead (recommended)
docker run --rm postgres:15 psql --version

# Or install PostgreSQL client
# On macOS:
brew install postgresql@15

# On Ubuntu/Debian:
sudo apt-get update
sudo apt-get install postgresql-client-15

# On Windows:
# Download from https://www.postgresql.org/download/windows/
```

---

## Port Issues

### Issue: Port 8000 Already in Use

**Symptoms:**
```
Error: listen tcp :8000: bind: address already in use
```

**Solution:**
```bash
# Option 1: Find and kill the process using port 8000
# On Windows:
netstat -ano | findstr :8000
taskkill /PID <PID> /F

# On macOS/Linux:
lsof -ti:8000 | xargs kill -9

# Option 2: Change the port in .env
# Edit .env and set:
PORT=8080

# Restart the server
go run ./cmd/server
```

---

### Issue: Frontend Port 3000 Already in Use

**Symptoms:**
```
Error: Port 3000 is already in use
```

**Solution:**
```bash
# Option 1: Kill process on port 3000
# On Windows:
netstat -ano | findstr :3000
taskkill /PID <PID> /F

# On macOS/Linux:
lsof -ti:3000 | xargs kill -9

# Option 2: Use a different port
# Next.js will prompt to use 3001 automatically, or:
PORT=3001 npm run dev
```

---

## Environment Variable Issues

### Issue: Environment Variables Not Loading

**Symptoms:**
```
Config value is empty
Database connection failed: missing credentials
```

**Solution:**

**Check `.env` file location:**
```bash
# .env must be in the project root (same directory as go.mod)
ls -la .env

# Verify file contents
cat .env
```

**Ensure proper loading in Go:**
```go
// In config/config.go, verify godotenv is loaded
import "github.com/joho/godotenv"

func init() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
}
```

**For frontend (Next.js):**
```bash
# Environment variables must be prefixed with NEXT_PUBLIC_ for client-side access
# .env.local (create this file)
NEXT_PUBLIC_API_URL=http://localhost:8000
```

---

### Issue: JWT Secret Not Set

**Symptoms:**
```
Error: JWT secret is empty
panic: invalid JWT configuration
```

**Solution:**
```bash
# Generate a secure random secret
# On macOS/Linux:
openssl rand -base64 32

# On Windows (PowerShell):
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))

# Add to .env:
JWT_SECRET=<generated-secret>
```

---

## Database Connection Errors

### Issue: Cannot Connect to Neon Database

**Symptoms:**
```
Error: dial tcp: lookup ep-xxx.neon.tech: no such host
pq: SSL is not enabled on the server
```

**Solution:**

**Verify connection string format:**
```env
# Correct format (must include sslmode=require)
DATABASE_URL=postgresql://user:password@ep-xxx.region.aws.neon.tech/neondb?sslmode=require

# Individual variables:
DB_HOST=ep-xxx.region.aws.neon.tech
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=neondb
DB_SSLMODE=require
```

**Test connection manually:**
```bash
# Using Docker
docker run --rm postgres:15 psql \
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require' \
  -c "SELECT version();"

# Should return PostgreSQL version
```

**Common fixes:**
- ✅ Ensure `sslmode=require` is set
- ✅ Check Neon project is not suspended (free tier)
- ✅ Verify credentials are correct (no extra spaces)
- ✅ Check firewall/network allows outbound connections on port 5432

---

### Issue: Database Schema Not Found

**Symptoms:**
```
Error: relation "users" does not exist
pq: relation "posts" does not exist
```

**Solution:**
```bash
# Run the master migration
docker run --rm -v $(pwd)/migrations:/migrations postgres:15 psql \
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require' \
  -f /migrations/000_master_schema.sql

# On Windows (PowerShell):
docker run --rm -v ${PWD}/migrations:/migrations postgres:15 psql `
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require' `
  -f /migrations/000_master_schema.sql

# Verify migration
docker run --rm -v $(pwd)/migrations:/migrations postgres:15 psql \
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require' \
  -f /migrations/verify_migration.sql
```

---

### Issue: Connection Pool Exhausted

**Symptoms:**
```
Error: pq: sorry, too many clients already
Error: remaining connection slots are reserved
```

**Solution:**

**Adjust connection pool settings in code:**
```go
// In db/postgres.go
config, err := pgxpool.ParseConfig(connString)
if err != nil {
    return nil, err
}

// Set reasonable limits
config.MaxConns = 10              // Max connections
config.MinConns = 2               // Min connections
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute

pool, err := pgxpool.NewWithConfig(ctx, config)
```

**For Neon free tier:**
- Limit: 100 concurrent connections
- Reduce `MaxConns` to 5-10 for development

---

## Runtime Errors

### Issue: Null Pointer Dereference

**Symptoms:**
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Solution:**

**Add nil checks:**
```go
// ❌ Bad
user := getUserByID(id)
fmt.Println(user.Username)  // Panic if user is nil

// ✅ Good
user := getUserByID(id)
if user == nil {
    return errors.New("user not found")
}
fmt.Println(user.Username)
```

**Check database query results:**
```go
// Always check errors
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    if err == pgx.ErrNoRows {
        return nil, ErrUserNotFound
    }
    return nil, fmt.Errorf("database error: %w", err)
}
```

---

### Issue: CORS Errors in Frontend

**Symptoms:**
```
Access to fetch at 'http://localhost:8000/api/v1/users/me' from origin 'http://localhost:3000' has been blocked by CORS policy
```

**Solution:**

**Verify CORS middleware in backend:**
```go
// In server/router.go
import "github.com/labstack/echo/v4/middleware"

e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"http://localhost:3000"},
    AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH},
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
    AllowCredentials: true,
}))
```

**For production, use environment variable:**
```go
allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
if allowedOrigins == "" {
    allowedOrigins = "http://localhost:3000"
}
```

---

### Issue: JWT Token Invalid or Expired

**Symptoms:**
```
Error: invalid or expired JWT
401 Unauthorized
```

**Solution:**

**Check token expiration:**
```go
// In middleware/auth.go
// Set appropriate expiration time
token.Claims = &jwt.StandardClaims{
    ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    IssuedAt:  time.Now().Unix(),
}
```

**Frontend: Store and send token correctly:**
```typescript
// Store token
localStorage.setItem('token', response.data.token);

// Send with requests
const token = localStorage.getItem('token');
const response = await fetch('/api/v1/users/me', {
  headers: {
    'Authorization': `Bearer ${token}`
  }
});
```

**Clear expired tokens:**
```typescript
// Check token expiration before use
const isTokenExpired = (token: string): boolean => {
  const payload = JSON.parse(atob(token.split('.')[1]));
  return payload.exp * 1000 < Date.now();
};
```

---

## Build/Deployment Errors

### Issue: Go Build Fails

**Symptoms:**
```
undefined: echo.Context
package splitter/internal/models is not in GOROOT
```

**Solution:**
```bash
# Ensure you're in the correct directory
cd splitter

# Clean and rebuild
go clean
go mod tidy
go mod download

# Build
go build -o bin/server ./cmd/server

# If still failing, verify module path in go.mod matches import paths
# go.mod should have:
# module splitter
```

---

### Issue: Docker Build Fails

**Symptoms:**
```
ERROR [internal] load metadata for docker.io/library/golang:1.21
failed to solve with frontend dockerfile.v0
```

**Solution:**
```bash
# Update Docker
# Check Docker is running
docker --version

# Pull base images manually
docker pull golang:1.21-alpine
docker pull postgres:15

# Rebuild with no cache
docker-compose build --no-cache

# Or build individually
docker build -f Dockerfile.backend -t splitter-backend .
```

---

### Issue: Frontend Build Fails

**Symptoms:**
```
Error: Module not found: Can't resolve 'components/UserProfile'
Type error: Property 'user' does not exist on type '{}'
```

**Solution:**
```bash
# Check import paths (case-sensitive)
# ❌ import UserProfile from 'Components/UserProfile'
# ✅ import UserProfile from 'components/UserProfile'

# TypeScript errors: Add proper types
interface Props {
  user: User;
}

# Clear Next.js cache
rm -rf .next
npm run build
```

---

## Debugging Tips

### Enable Debug Logging

**Backend (Go):**
```go
// In config/config.go
import "log/slog"

// Set log level based on environment
logLevel := slog.LevelInfo
if os.Getenv("ENV") == "development" {
    logLevel = slog.LevelDebug
}

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: logLevel,
}))
slog.SetDefault(logger)
```

**Frontend (Next.js):**
```typescript
// Add to next.config.js
module.exports = {
  reactStrictMode: true,
  logging: {
    fetches: {
      fullUrl: true,
    },
  },
};
```

---

### Database Query Debugging

**Log SQL queries:**
```go
// In repository functions
slog.Debug("executing query",
    "query", query,
    "args", args,
)

rows, err := pool.Query(ctx, query, args...)
```

**Use EXPLAIN ANALYZE:**
```sql
-- In psql or database client
EXPLAIN ANALYZE
SELECT * FROM posts
WHERE author_did = 'did:example:alice'
ORDER BY created_at DESC
LIMIT 20;
```

---

### Network Request Debugging

**Backend:**
```bash
# Enable Echo debug mode
# In server/router.go
e.Debug = true

# This will log all requests
```

**Frontend:**
```typescript
// Add request interceptor
axios.interceptors.request.use(request => {
  console.log('Starting Request:', request);
  return request;
});

axios.interceptors.response.use(
  response => {
    console.log('Response:', response);
    return response;
  },
  error => {
    console.error('Request Error:', error);
    return Promise.reject(error);
  }
);
```

---

### Check Application Health

**Backend health endpoint:**
```bash
curl http://localhost:8000/api/v1/health

# Should return:
# {"status":"ok"}
```

**Database connectivity:**
```bash
# Test from Go app
go run test_db_check.go

# Or use psql
docker run --rm postgres:15 psql \
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require' \
  -c "SELECT 1;"
```

---

## FAQ

### Q: How do I reset the database?

**A:** Drop and recreate all tables:
```bash
# Connect to database
docker run --rm -it postgres:15 psql \
  'postgresql://user:password@ep-xxx.neon.tech/neondb?sslmode=require'

# Drop all tables
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

# Re-run migration
\i /migrations/000_master_schema.sql
```

---

### Q: How do I create a new admin user?

**A:** The default admin is created automatically on first run. To create another:
```sql
-- Connect to database and run:
INSERT INTO users (username, email, password_hash, instance_domain, role, created_at)
VALUES (
  'newadmin',
  'admin@example.com',
  '$2a$10$...',  -- Generate with bcrypt
  'localhost',
  'admin',
  now()
);
```

Or use the API:
```bash
# Register user
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"newadmin","email":"admin@example.com","password":"password123"}'

# Then update role in database or use admin endpoint
```

---

### Q: How do I change the JWT secret in production?

**A:** 
1. Generate new secret: `openssl rand -base64 32`
2. Update `.env` file with new `JWT_SECRET`
3. Restart the application
4. **Note:** All existing tokens will be invalidated

---

### Q: Frontend can't connect to backend

**A:** Check these common issues:
- ✅ Backend is running on port 8000
- ✅ Frontend API URL is correct (`http://localhost:8000`)
- ✅ CORS is configured correctly
- ✅ No firewall blocking localhost connections
- ✅ Check browser console for errors

---

### Q: How do I enable HTTPS for local development?

**A:** Use a reverse proxy like Caddy:
```bash
# Install Caddy
# Create Caddyfile:
localhost {
    reverse_proxy :8000
}

# Run Caddy
caddy run
```

Or use mkcert for local certificates:
```bash
# Install mkcert
brew install mkcert  # macOS
# or download from https://github.com/FiloSottile/mkcert

# Create local CA
mkcert -install

# Generate certificate
mkcert localhost 127.0.0.1 ::1

# Use in Go server (update server config to use TLS)
```

---

### Q: Database migrations failed halfway, how do I recover?

**A:**
```bash
# Check which tables exist
docker run --rm postgres:15 psql \
  'postgresql://...' \
  -c "\dt"

# Option 1: Drop failed tables and re-run
# Option 2: Create a recovery migration that only creates missing tables
# Option 3: Full reset (see "How do I reset the database?")
```

---

### Q: How do I monitor application performance?

**A:** 
- **Database**: Use Neon dashboard for query performance
- **Backend**: Add middleware for request timing
- **Frontend**: Use React DevTools Profiler
- **Logs**: Implement structured logging and use log aggregation

```go
// Example: Request timing middleware
e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
    LogLatency: true,
    LogStatus:  true,
    LogURI:     true,
}))
```

---

### Q: Application is slow, how do I debug?

**A:**
1. **Check database queries**: Use `EXPLAIN ANALYZE`
2. **Add indexes**: See `DATABASE_SCHEMA.md` for recommended indexes
3. **Enable query logging**: Log slow queries (>100ms)
4. **Profile Go code**: Use `pprof`
5. **Check network latency**: Test Neon connection speed
6. **Monitor resources**: Check CPU/memory usage

```bash
# Profile Go application
go tool pprof http://localhost:8000/debug/pprof/profile
```

---

## Still Having Issues?

If your issue isn't covered here:

1. **Check logs**: Backend logs and browser console
2. **Search existing issues**: GitHub issues page
3. **Ask for help**: Create a new GitHub issue with:
   - Error message (full stack trace)
   - Steps to reproduce
   - Environment details (OS, Go version, Node version)
   - Relevant configuration (sanitized, no secrets)

**Useful debugging commands:**
```bash
# System info
go version
node --version
docker --version

# Check running processes
# Windows:
netstat -ano | findstr "8000 3000"

# macOS/Linux:
lsof -i :8000
lsof -i :3000

# Check environment
printenv | grep DB_
printenv | grep JWT_
```
