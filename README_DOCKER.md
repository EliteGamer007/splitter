# Docker Setup for Splitter

## Quick Start

### 1. Clone and Setup
```bash
git clone <repository-url>
cd splitter
cp .env.example .env
```

### 2. Edit `.env` file
Update the following values:
- `POSTGRES_PASSWORD` - Set a strong password
- `JWT_SECRET` - Generate a secure random string
- `PGADMIN_PASSWORD` - Set pgAdmin admin password

### 3. Start Services
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Check status
docker-compose ps
```

### 4. Access Services
- **Backend API**: http://localhost:8000
- **pgAdmin**: http://localhost:5050
  - Email: Value from `PGADMIN_EMAIL` in `.env`
  - Password: Value from `PGADMIN_PASSWORD` in `.env`
- **PostgreSQL**: localhost:5432
  - Database: Value from `POSTGRES_DB`
  - User: Value from `POSTGRES_USER`
  - Password: Value from `POSTGRES_PASSWORD`

### 5. Database Migrations
Migrations in `migrations/` folder will run automatically on first startup.

## Managing Services

### Stop Services
```bash
docker-compose down
```

### Stop and Remove Data (DESTRUCTIVE)
```bash
docker-compose down -v
```

### Restart Single Service
```bash
docker-compose restart postgres
docker-compose restart backend
```

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f postgres
docker-compose logs -f backend
```

## Connecting to Database

### From Host Machine
```bash
psql -h localhost -p 5432 -U splitter_user -d splitter_db
```

### From Backend Container
```bash
docker-compose exec backend psql -h postgres -U splitter_user -d splitter_db
```

### Using pgAdmin
1. Open http://localhost:5050
2. Login with credentials from `.env`
3. Server "Splitter PostgreSQL" should be auto-configured
4. Right-click > Connect (enter password from `.env`)

## Troubleshooting

### Backend can't connect to database
```bash
# Check if postgres is healthy
docker-compose ps

# Check postgres logs
docker-compose logs postgres

# Verify environment variables
docker-compose exec backend env | grep DB_
```

### Reset Everything
```bash
docker-compose down -v
docker-compose up -d
```

### Access PostgreSQL Shell
```bash
docker-compose exec postgres psql -U splitter_user -d splitter_db
```

## Production Considerations

1. **Change all default passwords** in `.env`
2. **Never commit `.env`** file to git
3. **Use Docker secrets** for sensitive data in production
4. **Setup automated backups**:
   ```bash
   docker-compose exec postgres pg_dump -U splitter_user splitter_db > backup.sql
   ```
5. **Configure firewall** to restrict database access
6. **Use SSL/TLS** for database connections in production
