# Splitter

A federated social media application built with Go, Echo framework, and PostgreSQL.

## Prerequisites

- **Go**: 1.21 or higher - [Download Go](https://go.dev/dl/)
- **PostgreSQL**: 14 or higher - [Download PostgreSQL](https://www.postgresql.org/download/)
- **Git**: For version control

## Project Structure

```
splitter/
├── cmd/server/          # Application entrypoint
├── internal/            # Internal packages
│   ├── config/         # Configuration management
│   ├── db/             # Database connection
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # Authentication middleware
│   ├── models/         # Data models
│   ├── repository/     # Data access layer
│   └── server/         # Router setup
├── migrations/         # Database migrations
├── .env.example        # Environment variables template
└── Makefile           # Build commands
```

## Getting Started

### 1. Clone the Repository

```bash
git clone <repository-url>
cd splitter
```

### 2. Start PostgreSQL Server

**Windows (with PostgreSQL installed):**
```powershell
# Check if PostgreSQL service is running
Get-Service -Name postgresql*

# If not running, start it (run as Administrator)
Start-Service -Name postgresql-x64-14  # Adjust version number
```

**Linux/Mac:**
```bash
# Start PostgreSQL service
sudo systemctl start postgresql   # Linux
brew services start postgresql    # Mac
```

**Verify PostgreSQL is running:**
```bash
psql --version
psql -U postgres -c "SELECT version();"
```

### 3. Create Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE splitter;

# Exit
\q
```

### 4. Run Database Migrations

```bash
psql -U postgres -d splitter -f migrations/001_initial_schema.sql
```

### 5. Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your database credentials
# Update these values:
#   DB_HOST=localhost
#   DB_PORT=5432
#   DB_USER=postgres
#   DB_PASSWORD=your_password
#   DB_NAME=splitter
#   JWT_SECRET=change-this-to-a-secure-random-string
```

### 6. Install Dependencies

```bash
go mod download
```

### 7. Run the Application

**Option 1: Using Go Run**
```bash
go run cmd/server/main.go
```

**Option 2: Using Makefile**
```bash
make run
```

**Option 3: Build and Run Binary**
```bash
make build
./bin/server          # Linux/Mac
.\bin\server.exe      # Windows
```

The server will start on `http://localhost:3000`

You should see:
```
Database connection established successfully
Starting server on port 3000
⇨ http server started on [::]:3000
```

## API Endpoints

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | User login |
| GET | `/api/v1/users/:id` | Get user profile |
| GET | `/api/v1/posts/:id` | Get post by ID |

### Protected Endpoints (Require Authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/me` | Get current user |
| PUT | `/api/v1/users/me` | Update profile |
| DELETE | `/api/v1/users/me` | Delete account |
| POST | `/api/v1/posts` | Create new post |
| GET | `/api/v1/posts/feed` | Get personalized feed |
| PUT | `/api/v1/posts/:id` | Update post |
| DELETE | `/api/v1/posts/:id` | Delete post |

### Authentication

Protected endpoints require a JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" http://localhost:3000/api/v1/users/me
```

## Development

### Available Make Commands

```bash
make help          # Show all available commands
make run           # Run the application
make build         # Build the binary
make test          # Run tests
make clean         # Clean build artifacts
make fmt           # Format code
make lint          # Run linter
```

### Testing the API

**Health Check:**
```bash
curl http://localhost:3000/api/v1/health
```

**Register a User:**
```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "full_name": "Test User"
  }'
```

**Login:**
```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

## Database Schema

The application uses a comprehensive schema for federated social media including:

- **Users & Authentication**: Local and federated user accounts
- **Posts & Media**: Content creation with attachments
- **Social Features**: Follows, likes, reposts, bookmarks
- **Messaging**: End-to-end encrypted direct messages
- **Federation**: ActivityPub-compatible inbox/outbox
- **Moderation**: Reports, blocks, and admin actions

See [migrations/001_initial_schema.sql](migrations/001_initial_schema.sql) for the complete schema.

## Troubleshooting

### Database Connection Issues

**Error: "Failed to initialize database"**
- Verify PostgreSQL is running: `Get-Service postgresql*` (Windows) or `systemctl status postgresql` (Linux)
- Check database credentials in `.env` file
- Ensure database exists: `psql -U postgres -l | grep splitter`

### Port Already in Use

**Error: "bind: address already in use"**
- Change the `PORT` value in `.env` file
- Check what's using the port: `netstat -ano | findstr :3000` (Windows) or `lsof -i :3000` (Linux/Mac)

### Migration Errors

**Error: "relation already exists"**
- Database tables already exist. Either drop the database and recreate, or use migration rollback tools

## Contributing

1. Create a new branch for your feature
2. Make your changes
3. Run tests: `make test`
4. Format code: `make fmt`
5. Commit your changes
6. Push to your branch
7. Create a Pull Request

## License

See [LICENSE](LICENSE) file for details.

## Team

Software Engineering Project - Team 5
