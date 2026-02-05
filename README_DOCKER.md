# Docker Setup for Splitter

> **Note:** The primary database is now **Neon Cloud PostgreSQL**. Docker is used only for pgAdmin and running psql commands. See [NEON_SETUP_GUIDE.md](NEON_SETUP_GUIDE.md) for the main setup guide.

## Docker Usage

Docker Compose provides:
- **pgAdmin** — Database management UI at http://localhost:5050
- **psql** — Run migrations and queries against Neon

### Start pgAdmin
```bash
docker-compose up -d pgadmin
```

### Run Migrations via Docker
```bash
docker run --rm postgres:15 psql \
  'postgresql://user:password@host.neon.tech/dbname?sslmode=require' \
  -f migrations/000_master_schema.sql
```

### Query Database via Docker
```bash
docker run --rm postgres:15 psql \
  'YOUR_NEON_CONNECTION_STRING' \
  -c "SELECT username, role FROM users;"
```

### pgAdmin Access
- **URL:** http://localhost:5050
- **Email:** Value from `PGADMIN_DEFAULT_EMAIL` in `.env`
- **Password:** Value from `PGADMIN_DEFAULT_PASSWORD` in `.env`

## Running the Application (No Docker Required)

```bash
# Backend
cd splitter
go run ./cmd/server       # http://localhost:8000

# Frontend
cd Splitter-frontend
npm run dev               # http://localhost:3000
```

## Environment Setup

```bash
cp .env.example .env
# Edit .env with your Neon credentials — see .env.example for all required variables
```

## Production Notes

1. **Change all default passwords** — admin account, JWT secret
2. **Never commit `.env`** to git
3. **Neon handles backups** automatically
4. **SSL is required** for all Neon connections (`sslmode=require`)
