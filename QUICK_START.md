# Quick Start: Running Both Instances

## Prerequisites
- Go installed
- PostgreSQL databases set up (Neon Cloud):
  - neondb for Instance 1
  - neondb_2 for Instance 2
- Both .env files configured

## Start Instance 1 (splitter-1)

```powershell
cd "c:\Users\Sanjeev Srinivas\Desktop\splitter"
go run cmd/server/main.go
```

**Available at**: http://localhost:8000
**Federation Domain**: splitter-1
**Database**: neondb

## Start Instance 2 (splitter-2)

```powershell
cd "c:\Users\Sanjeev Srinivas\Desktop\splitter"
$env:ENV_FILE = ".env.instance2"
go run cmd/server/main.go
```

**Available at**: http://localhost:8001
**Federation Domain**: splitter-2
**Database**: neondb_2

## Verify Both Running

```powershell
# Test Instance 1
curl http://localhost:8000/api/v1/health

# Test Instance 2
curl http://localhost:8001/api/v1/health
```

Both should return: `{"status":"ok"}`

## Test Federation

```powershell
# WebFinger lookup on Instance 2
curl "http://localhost:8001/.well-known/webfinger?resource=acct:admin@splitter-2"

# Get Actor profile on Instance 2
curl -H "Accept: application/activity+json" http://localhost:8001/ap/users/admin
```

## Connect Frontend

Configure your frontend's API base URL to point to the instance you want to test:
- Instance 1: `http://localhost:8000`
- Instance 2: `http://localhost:8001`

Then navigate to http://localhost:3000 (or your frontend port).

## Stopping Instances

Press `Ctrl+C` in each terminal window running the instances.

## Logs Locations

- Instance 1: `instance1.log`
- Instance 2: `instance2.log`
