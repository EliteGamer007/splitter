# Coding Standards

## Overview

This document defines the coding standards and conventions for the Splitter project. Following these standards ensures code consistency, maintainability, and collaboration efficiency across the team.

**Tech Stack:**
- **Backend**: Go 1.21+, Echo v4 framework
- **Database**: PostgreSQL 15 (Neon Cloud)
- **Frontend**: Next.js, React
- **Authentication**: JWT, bcrypt, Ed25519

---

## Folder Structure

### Backend (Go)

```
splitter/
├── cmd/
│   └── server/              # Application entry point (main.go)
├── internal/                # Private application code
│   ├── config/              # Configuration management
│   ├── db/                  # Database connection and migrations
│   ├── handlers/            # HTTP request handlers (controllers)
│   ├── middleware/          # HTTP middleware (auth, logging, CORS)
│   ├── models/              # Data models (structs)
│   ├── repository/          # Data access layer (database queries)
│   ├── server/              # Router setup and server initialization
│   └── service/             # Business logic and external services
├── migrations/              # SQL migration scripts
├── .env.example             # Environment variable template
├── go.mod                   # Go module dependencies
├── go.sum                   # Dependency checksums
├── Makefile                 # Build and deployment commands
└── Dockerfile.backend       # Backend containerization
```

**Rules:**
- **`cmd/`** - Only contains `main.go` files, minimal logic
- **`internal/`** - All application code (not importable by external projects)
- **`migrations/`** - SQL files only, numbered sequentially (e.g., `001_initial_schema.sql`)
- **No business logic in handlers** - Handlers should delegate to services/repositories
- **No database queries in handlers** - Use repository layer

### Frontend (Next.js)

```
Splitter-frontend/
├── app/                     # Next.js 13+ app directory
│   ├── (auth)/              # Auth-related pages
│   ├── (main)/              # Main app pages
│   └── api/                 # API routes (if needed)
├── components/              # React components
│   ├── ui/                  # Reusable UI components
│   └── features/            # Feature-specific components
├── lib/                     # Utility functions and helpers
├── hooks/                   # Custom React hooks
├── types/                   # TypeScript type definitions
├── public/                  # Static assets
└── styles/                  # Global styles
```

**Rules:**
- **Component organization**: Group by feature, not by type
- **One component per file**: Except for tightly coupled sub-components
- **Use TypeScript**: All new code must be TypeScript

---

## Naming Conventions

### Go (Backend)

#### Files
- **Lowercase with underscores**: `user_handler.go`, `post_repo.go`
- **Test files**: `user_handler_test.go`
- **Suffix by type**:
  - Handlers: `*_handler.go`
  - Repositories: `*_repo.go`
  - Models: `*.go` (e.g., `user.go`)
  - Services: `*_service.go`

#### Variables & Functions
- **camelCase for private**: `userID`, `getUserByID()`
- **PascalCase for public**: `UserID`, `GetUserByID()`
- **Acronyms capitalized**: `ID`, `URL`, `HTTP`, `JSON`, `DID`
  - Correct: `userID`, `UserID`, `parseHTTPRequest`
  - Incorrect: `userId`, `UserId`, `parseHttpRequest`

#### Structs & Interfaces
- **PascalCase**: `User`, `PostRepository`
- **Interface naming**: 
  - Prefer descriptive names: `UserRepository`, `PostService`
  - Single-method interfaces: Use `-er` suffix: `Reader`, `Writer`, `Validator`

#### Constants
- **PascalCase or UPPER_SNAKE_CASE**:
  - Exported: `DefaultTimeout`, `MaxRetries`
  - Grouped: Use `const` blocks with `iota`

```go
const (
    RoleUser      = "user"
    RoleModerator = "moderator"
    RoleAdmin     = "admin"
)
```

### Database

#### Tables
- **Lowercase with underscores**: `users`, `message_threads`, `admin_actions`
- **Plural names**: `users` (not `user`)
- **No prefixes**: Avoid `tbl_users`

#### Columns
- **Lowercase with underscores**: `user_id`, `created_at`, `is_suspended`
- **Boolean columns**: Prefix with `is_` or `has_`: `is_locked`, `has_verified`
- **Timestamps**: Suffix with `_at`: `created_at`, `updated_at`, `deleted_at`
- **Foreign keys**: `<table>_id`: `user_id`, `post_id`

### Frontend (TypeScript/React)

#### Files
- **PascalCase for components**: `UserProfile.tsx`, `PostCard.tsx`
- **camelCase for utilities**: `formatDate.ts`, `apiClient.ts`
- **Hooks**: `use` prefix: `useAuth.ts`, `usePosts.ts`

#### Components
- **PascalCase**: `UserProfile`, `PostCard`, `NavBar`
- **Props interfaces**: `<Component>Props`: `UserProfileProps`

#### Variables & Functions
- **camelCase**: `userName`, `fetchPosts()`, `handleSubmit()`
- **Boolean variables**: Prefix with `is`, `has`, `should`: `isLoading`, `hasError`, `shouldRedirect`
- **Event handlers**: Prefix with `handle`: `handleClick`, `handleSubmit`

---

## Code Formatting

### Go

**Use `gofmt` and `goimports`:**
```bash
# Format all Go files
gofmt -w .

# Organize imports
goimports -w .
```

**Standards:**
- **Indentation**: Tabs (enforced by `gofmt`)
- **Line length**: Soft limit of 100 characters
- **Imports**: Group in order:
  1. Standard library
  2. External packages
  3. Internal packages

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/google/uuid"

    "splitter/internal/models"
    "splitter/internal/repository"
)
```

**Error handling:**
```go
// ✅ Good: Check errors immediately
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user: %w", err)
}

// ❌ Bad: Ignoring errors
user, _ := repo.GetUserByID(ctx, id)
```

**Function structure:**
```go
// ✅ Good: Clear, single responsibility
func (h *UserHandler) GetUser(c echo.Context) error {
    id := c.Param("id")
    
    user, err := h.userRepo.GetByID(c.Request().Context(), id)
    if err != nil {
        return c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
    }
    
    return c.JSON(http.StatusOK, user)
}
```

### TypeScript/React

**Use Prettier and ESLint:**
```bash
# Format code
npm run format

# Lint code
npm run lint
```

**Standards:**
- **Indentation**: 2 spaces
- **Quotes**: Single quotes for strings
- **Semicolons**: Required
- **Line length**: 80-100 characters

**Component structure:**
```tsx
// ✅ Good: Functional component with TypeScript
interface UserProfileProps {
  userId: string;
  onUpdate?: () => void;
}

export function UserProfile({ userId, onUpdate }: UserProfileProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchUser(userId);
  }, [userId]);

  const fetchUser = async (id: string) => {
    // Implementation
  };

  return (
    <div className="user-profile">
      {/* JSX */}
    </div>
  );
}
```

---

## Commit Message Conventions

Follow **Conventional Commits** specification:

### Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, no logic change)
- **refactor**: Code refactoring (no feature/fix)
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, build config)
- **perf**: Performance improvements

### Examples
```bash
# Feature
feat(auth): add DID authentication support

# Bug fix
fix(posts): resolve null pointer in feed endpoint

# Documentation
docs(readme): update installation instructions

# Refactoring
refactor(handlers): extract validation logic to middleware

# Multiple changes
feat(messaging): implement E2EE for direct messages

- Add encryption/decryption functions
- Update message schema
- Add key exchange endpoint
```

### Rules
- **Subject line**: 
  - Max 72 characters
  - Lowercase
  - No period at the end
  - Imperative mood ("add" not "added")
- **Body**: 
  - Wrap at 72 characters
  - Explain *what* and *why*, not *how*
- **Scope**: Optional, use feature/module name

---

## Branch Naming

### Format
```
<type>/<issue-number>-<short-description>
```

### Types
- **feature/** - New features
- **fix/** - Bug fixes
- **hotfix/** - Urgent production fixes
- **refactor/** - Code refactoring
- **docs/** - Documentation updates
- **test/** - Test additions/updates

### Examples
```bash
feature/123-did-authentication
fix/456-null-pointer-feed
hotfix/789-security-patch
refactor/101-extract-validation
docs/202-api-documentation
test/303-integration-tests
```

### Rules
- **Lowercase with hyphens**: `feature/add-messaging` (not `feature/Add_Messaging`)
- **Descriptive**: Should explain the purpose
- **Issue tracking**: Include issue/ticket number if applicable
- **Short-lived**: Merge within 1-2 weeks

### Workflow
```bash
# Create feature branch from main
git checkout -b feature/123-user-search main

# Work on feature
git add .
git commit -m "feat(search): add user search endpoint"

# Push and create PR
git push origin feature/123-user-search
```

---

## Error Handling Standards

### Go

**Always wrap errors with context:**
```go
// ✅ Good
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user %s: %w", id, err)
}

// ❌ Bad
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    return nil, err
}
```

**Use custom error types for domain errors:**
```go
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}
```

**HTTP error responses:**
```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
    Code    string `json:"code,omitempty"`
}

// Usage
return c.JSON(http.StatusBadRequest, ErrorResponse{
    Error:   "Invalid input",
    Message: "Username must be at least 3 characters",
    Code:    "INVALID_USERNAME",
})
```

### TypeScript

**Use try-catch for async operations:**
```typescript
// ✅ Good
async function fetchUser(id: string): Promise<User> {
  try {
    const response = await api.get(`/users/${id}`);
    return response.data;
  } catch (error) {
    if (error.response?.status === 404) {
      throw new NotFoundError(`User ${id} not found`);
    }
    throw new Error('Failed to fetch user');
  }
}
```

**Custom error classes:**
```typescript
class NotFoundError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'NotFoundError';
  }
}
```

---

## Logging Guidelines

### Go

**Use structured logging (consider `zerolog` or `slog`):**
```go
import "log/slog"

// ✅ Good: Structured logging
slog.Info("user created",
    "user_id", user.ID,
    "username", user.Username,
    "role", user.Role,
)

slog.Error("failed to create user",
    "error", err,
    "username", username,
)

// ❌ Bad: Unstructured logging
log.Printf("User created: %s", user.ID)
```

**Log levels:**
- **DEBUG**: Detailed diagnostic information
- **INFO**: General informational messages (user actions, state changes)
- **WARN**: Warning messages (deprecated usage, recoverable errors)
- **ERROR**: Error messages (failed operations, exceptions)
- **FATAL**: Critical errors (application cannot continue)

**What to log:**
- ✅ User actions (login, post creation, follows)
- ✅ External API calls (federation activities)
- ✅ Database errors
- ✅ Authentication failures
- ❌ Passwords or sensitive data
- ❌ Excessive debug info in production

### TypeScript

**Use console methods appropriately:**
```typescript
// Development
console.log('User data:', user);
console.error('API error:', error);

// Production: Use logging service
logger.info('User logged in', { userId: user.id });
logger.error('API request failed', { error, endpoint });
```

---

## API Response Standards

### Success Responses

**Single resource:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "alice",
  "display_name": "Alice",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Collection:**
```json
{
  "data": [
    { "id": "...", "username": "alice" },
    { "id": "...", "username": "bob" }
  ],
  "pagination": {
    "total": 42,
    "page": 1,
    "per_page": 20,
    "total_pages": 3
  }
}
```

**Action confirmation:**
```json
{
  "success": true,
  "message": "User followed successfully"
}
```

### Error Responses

**Standard error format:**
```json
{
  "error": "Validation failed",
  "message": "Username must be at least 3 characters",
  "code": "VALIDATION_ERROR",
  "details": {
    "field": "username",
    "constraint": "min_length"
  }
}
```

**HTTP status codes:**
- **200 OK**: Successful GET, PUT, PATCH
- **201 Created**: Successful POST (resource created)
- **204 No Content**: Successful DELETE
- **400 Bad Request**: Invalid input
- **401 Unauthorized**: Missing/invalid authentication
- **403 Forbidden**: Authenticated but not authorized
- **404 Not Found**: Resource doesn't exist
- **409 Conflict**: Resource conflict (duplicate username)
- **422 Unprocessable Entity**: Validation errors
- **500 Internal Server Error**: Server error

---

## Notes / Best Practices

### General Principles
- **DRY (Don't Repeat Yourself)**: Extract common logic into functions/utilities
- **KISS (Keep It Simple)**: Prefer simple solutions over complex ones
- **YAGNI (You Aren't Gonna Need It)**: Don't add functionality until needed
- **Single Responsibility**: Each function/class should do one thing well

### Code Reviews
- **Review your own code first**: Before requesting review
- **Small PRs**: Keep pull requests under 400 lines when possible
- **Descriptive PR titles**: Use conventional commit format
- **Test coverage**: Include tests for new features
- **Documentation**: Update docs for API changes

### Testing
- **Unit tests**: Test individual functions/methods
- **Integration tests**: Test API endpoints
- **Test file naming**: `*_test.go` (Go), `*.test.ts` (TypeScript)
- **Coverage goal**: Aim for 70%+ coverage on critical paths

### Security
- **Never commit secrets**: Use `.env` files (gitignored)
- **Validate all inputs**: Never trust user input
- **Use parameterized queries**: Prevent SQL injection
- **Hash passwords**: Always use bcrypt (cost factor 10+)
- **Sanitize output**: Prevent XSS attacks

### Performance
- **Database indexes**: Index foreign keys and frequently queried columns
- **Pagination**: Always paginate large collections
- **Caching**: Cache expensive queries (use Redis if needed)
- **N+1 queries**: Avoid in loops, use joins or batch loading

### Documentation
- **Code comments**: Explain *why*, not *what*
- **Function documentation**: Document public APIs
- **README updates**: Keep README current with project state
- **API documentation**: Document all endpoints (consider OpenAPI/Swagger)

### Git Workflow
- **Pull before push**: Always pull latest changes first
- **Atomic commits**: One logical change per commit
- **Rebase vs merge**: Use rebase for feature branches, merge for main
- **Delete merged branches**: Clean up after PR merge
