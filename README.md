# Splitter - Federated Social Media App

Software Engineering project of Team 5. A federated social media application built with Go and PostgreSQL.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher

## Quick Start

### 1. Database Setup

Create the PostgreSQL database and load the schema:

```bash
psql -U postgres
CREATE DATABASE splitter;
\c splitter
\i Database/schema.sql
```

### 2. Environment Configuration

The `.env` file is already configured with default values:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=splitter
```

**Update these values to match your PostgreSQL credentials.**

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run the Application

```bash
go run main.go
```

You should see:
```
Database connection established successfully
Splitter application started successfully
```

## Project Structure

```
splitter/
├── Database/
│   └── schema.sql       # PostgreSQL database schema
├── database/
│   └── db.go            # Database connection handler with pooling
├── main.go              # Application entry point
├── .env                 # Environment variables (not in git)
├── .env.example         # Environment template
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
└── README.md            # This file
```

## Database Connection

The application uses `pgx/v5` PostgreSQL driver with connection pooling:
- **Max connections:** 25
- **Min connections:** 5
- **Auto-reconnect:** Yes
- **Connection timeout:** Configurable

Connection details are automatically loaded from the `.env` file.

## Development

### Adding New Features

1. Create packages for your features (e.g., `handlers/`, `services/`, `models/`)
2. Use the global `database.DB` connection pool for queries
3. Import and initialize in `main.go`

### Example Query

```go
import "splitter/database"

func GetUser(ctx context.Context, userID string) error {
    var username string
    err := database.DB.QueryRow(ctx, 
        "SELECT username FROM users WHERE id = $1", 
        userID).Scan(&username)
    return err
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| DB_HOST | PostgreSQL host | localhost |
| DB_PORT | PostgreSQL port | 5432 |
| DB_USER | Database user | postgres |
| DB_PASSWORD | Database password | postgres |
| DB_NAME | Database name | splitter |

## License

See LICENSE file for details.
