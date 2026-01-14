# Contributing to Splitter

Thank you for contributing to Splitter! This guide will help you get started.

## Development Workflow

### 1. Setting Up Your Environment

```bash
# Clone the repository
git clone <repository-url>
cd splitter

# Copy environment file
cp .env.example .env

# Update .env with your PostgreSQL credentials

# Install dependencies
go mod download

# Run migrations
psql -U postgres -d splitter -f migrations/001_initial_schema.sql

# Run the application
make run
```

### 2. Creating a New Feature

```bash
# Create a new branch
git checkout -b feature/your-feature-name

# Make your changes
# ... code ...

# Format code
make fmt

# Run tests
make test

# Commit with clear messages
git commit -m "Add: description of your feature"

# Push to your branch
git push origin feature/your-feature-name
```

### 3. Code Standards

#### Go Code Style
- Follow official Go formatting: `gofmt` and `goimports`
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused
- Handle errors explicitly

#### Project Structure
```
internal/
├── handlers/     # HTTP request handlers
├── models/       # Data structures
├── repository/   # Database access
├── middleware/   # HTTP middleware
└── server/       # Router setup
```

#### Adding New Endpoints

1. Define model in `internal/models/`
2. Create repository in `internal/repository/`
3. Create handler in `internal/handlers/`
4. Register route in `internal/server/router.go`

Example:
```go
// internal/models/example.go
type Example struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// internal/repository/example_repo.go
func (r *ExampleRepository) Create(ctx context.Context, e *Example) error {
    // Implementation
}

// internal/handlers/example_handler.go
func (h *ExampleHandler) Create(c echo.Context) error {
    // Implementation
}

// internal/server/router.go
api.POST("/examples", exampleHandler.Create)
```

### 4. Database Migrations

When adding new tables or modifying schema:

1. Create a new migration file: `migrations/00X_description.sql`
2. Test the migration locally
3. Update `migrations/README.md` if needed
4. Document any schema changes in PR description

### 5. Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Test a specific package
go test ./internal/handlers -v
```

### 6. Commit Message Format

Use clear, descriptive commit messages:

```
Add: New feature description
Fix: Bug description
Update: Changes to existing feature
Remove: Removed feature
Refactor: Code refactoring
Docs: Documentation changes
```

### 7. Pull Request Process

1. Ensure your code follows the style guide
2. Update documentation if needed
3. Add tests for new features
4. Ensure all tests pass
5. Create a pull request with clear description
6. Wait for code review
7. Address review comments
8. Merge after approval

## Common Tasks

### Adding a New Model

```go
// internal/models/your_model.go
package models

type YourModel struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Adding a New Repository

```go
// internal/repository/your_repo.go
package repository

type YourRepository struct{}

func NewYourRepository() *YourRepository {
    return &YourRepository{}
}

func (r *YourRepository) Create(ctx context.Context, item *models.YourModel) error {
    // Implementation using db.GetDB()
}
```

### Adding a New Handler

```go
// internal/handlers/your_handler.go
package handlers

type YourHandler struct {
    repo *repository.YourRepository
}

func NewYourHandler(repo *repository.YourRepository) *YourHandler {
    return &YourHandler{repo: repo}
}

func (h *YourHandler) Create(c echo.Context) error {
    // Implementation
}
```

## Questions or Issues?

- Check existing issues on GitHub
- Ask questions in team chat
- Review documentation in `/docs`
- Refer to [README.md](README.md) for setup

## Code Review Checklist

Before submitting PR, ensure:

- [ ] Code follows Go conventions
- [ ] All tests pass
- [ ] Code is formatted (`make fmt`)
- [ ] No sensitive data in commits
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] Branch is up to date with main

Thank you for contributing! 🚀
