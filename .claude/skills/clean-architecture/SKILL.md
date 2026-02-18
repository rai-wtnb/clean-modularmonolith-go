---
name: clean-architecture
description: Robert C. Martin's Clean Architecture principles for Go. Use when designing layer structure, defining dependencies between layers, creating use cases, implementing ports and adapters, or understanding dependency inversion. Covers domain, application, infrastructure layers and the dependency rule.
---

# Clean Architecture for Go

Clean Architecture separates concerns into layers with strict dependency rules, ensuring business logic remains independent of frameworks, databases, and UI.

## The Dependency Rule

**Dependencies point inward. Inner layers know nothing about outer layers.**

```
┌─────────────────────────────────────────────────────────────┐
│                      Infrastructure                          │
│    (DB, Web Framework, External APIs, File System)          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Interface                         │    │
│  │     (HTTP Handlers, gRPC, CLI, Presenters)          │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │                Application                   │    │    │
│  │  │         (Use Cases, App Services)           │    │    │
│  │  │  ┌─────────────────────────────────────┐    │    │    │
│  │  │  │             Domain                   │    │    │    │
│  │  │  │   (Entities, Value Objects, Rules)   │    │    │    │
│  │  │  └─────────────────────────────────────┘    │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
module/
├── domain/                 # Innermost layer - Pure business rules
│   ├── entity/            # Business entities with behavior
│   │   ├── user.go
│   │   └── order.go
│   ├── valueobject/       # Immutable value objects
│   │   ├── email.go
│   │   └── money.go
│   ├── repository.go      # Repository interfaces (ports)
│   ├── service.go         # Domain services
│   └── errors.go          # Domain-specific errors
├── application/           # Use cases layer - Application logic
│   ├── command/           # Commands (write operations)
│   │   ├── create_user.go
│   │   └── update_user.go
│   ├── query/             # Queries (read operations)
│   │   ├── get_user.go
│   │   └── list_users.go
│   └── port/              # Output ports (secondary adapters)
│       ├── repository.go  # May extend domain repository
│       └── notifier.go    # External service ports
├── infrastructure/        # Outermost layer - Frameworks & drivers
│   ├── persistence/       # Database implementations
│   │   └── postgres/
│   │       ├── user_repository.go
│   │       └── migrations/
│   ├── messaging/         # Message queue implementations
│   │   └── rabbitmq/
│   └── external/          # External service clients
│       └── stripe/
└── ports/                 # Input ports - Interface layer
    ├── http/              # HTTP handlers
    │   ├── handler.go
    │   ├── router.go
    │   └── dto/
    └── grpc/              # gRPC handlers
        └── server.go
```

## Layer 1: Domain (Innermost)

The domain layer contains enterprise business rules. It has **no dependencies** on other layers or external libraries (except Go standard library).

### Entities

Entities have identity and lifecycle. They encapsulate business rules.

```go
// domain/entity/user.go
package entity

import (
    "errors"
    "time"

    "myapp/modules/users/domain/valueobject"
)

// UserID is a strongly-typed identifier.
type UserID string

func NewUserID() UserID {
    return UserID(uuid.New().String())
}

func ParseUserID(s string) (UserID, error) {
    if _, err := uuid.Parse(s); err != nil {
        return "", errors.New("invalid user id format")
    }
    return UserID(s), nil
}

func (id UserID) String() string { return string(id) }

// UserStatus represents the user's account status.
type UserStatus string

const (
    StatusPending  UserStatus = "pending"
    StatusActive   UserStatus = "active"
    StatusInactive UserStatus = "inactive"
)

// User is a domain entity representing a user in the system.
type User struct {
    id        UserID
    email     valueobject.Email
    name      string
    status    UserStatus
    createdAt time.Time
    updatedAt time.Time
}

// NewUser creates a new User with validation.
// This is the only way to create a valid User.
func NewUser(email valueobject.Email, name string) (*User, error) {
    if name == "" {
        return nil, ErrEmptyName
    }
    if len(name) > 100 {
        return nil, ErrNameTooLong
    }

    now := time.Now()
    return &User{
        id:        NewUserID(),
        email:     email,
        name:      name,
        status:    StatusPending,
        createdAt: now,
        updatedAt: now,
    }, nil
}

// ReconstructUser recreates a User from persistence.
// Used by repositories - bypasses validation for trusted data.
func ReconstructUser(id, email, name, status string, createdAt, updatedAt time.Time) (*User, error) {
    uid, err := ParseUserID(id)
    if err != nil {
        return nil, err
    }
    em, err := valueobject.NewEmail(email)
    if err != nil {
        return nil, err
    }
    return &User{
        id:        uid,
        email:     em,
        name:      name,
        status:    UserStatus(status),
        createdAt: createdAt,
        updatedAt: updatedAt,
    }, nil
}

// Business behavior methods - rules live here

// Activate transitions user from pending to active.
func (u *User) Activate() error {
    if u.status != StatusPending {
        return ErrInvalidStatusTransition
    }
    u.status = StatusActive
    u.updatedAt = time.Now()
    return nil
}

// Deactivate transitions user to inactive.
func (u *User) Deactivate() error {
    if u.status == StatusInactive {
        return ErrAlreadyInactive
    }
    u.status = StatusInactive
    u.updatedAt = time.Now()
    return nil
}

// CanPlaceOrder checks if user can perform orders.
func (u *User) CanPlaceOrder() bool {
    return u.status == StatusActive
}

// Getters - expose state without allowing direct modification
func (u *User) ID() UserID              { return u.id }
func (u *User) Email() valueobject.Email { return u.email }
func (u *User) Name() string            { return u.name }
func (u *User) Status() UserStatus      { return u.status }
func (u *User) CreatedAt() time.Time    { return u.createdAt }
func (u *User) UpdatedAt() time.Time    { return u.updatedAt }
```

### Value Objects

Value objects are immutable and compared by value, not identity.

```go
// domain/valueobject/email.go
package valueobject

import (
    "errors"
    "regexp"
    "strings"
)

var (
    ErrInvalidEmail = errors.New("invalid email format")
    emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Email is a value object representing a validated email address.
type Email struct {
    value string
}

// NewEmail creates a validated Email value object.
func NewEmail(value string) (Email, error) {
    normalized := strings.ToLower(strings.TrimSpace(value))
    if normalized == "" {
        return Email{}, ErrInvalidEmail
    }
    if !emailRegex.MatchString(normalized) {
        return Email{}, ErrInvalidEmail
    }
    return Email{value: normalized}, nil
}

func (e Email) String() string { return e.value }

// Equals compares two emails by value.
func (e Email) Equals(other Email) bool {
    return e.value == other.value
}

// Domain returns the domain part of the email.
func (e Email) Domain() string {
    parts := strings.Split(e.value, "@")
    if len(parts) != 2 {
        return ""
    }
    return parts[1]
}
```

```go
// domain/valueobject/money.go
package valueobject

import "errors"

var ErrCurrencyMismatch = errors.New("currency mismatch")

// Money represents a monetary amount with currency.
type Money struct {
    amount   int64  // In smallest unit (cents)
    currency string // ISO 4217
}

func NewMoney(amount int64, currency string) Money {
    return Money{amount: amount, currency: currency}
}

func (m Money) Amount() int64   { return m.amount }
func (m Money) Currency() string { return m.currency }

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, ErrCurrencyMismatch
    }
    return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m Money) Multiply(factor int64) Money {
    return Money{amount: m.amount * factor, currency: m.currency}
}

func (m Money) IsZero() bool {
    return m.amount == 0
}
```

### Domain Errors

```go
// domain/errors.go
package domain

import "errors"

var (
    ErrUserNotFound            = errors.New("user not found")
    ErrEmailAlreadyExists      = errors.New("email already exists")
    ErrInvalidStatusTransition = errors.New("invalid status transition")
    ErrAlreadyInactive         = errors.New("user already inactive")
    ErrEmptyName               = errors.New("name cannot be empty")
    ErrNameTooLong             = errors.New("name exceeds maximum length")
)
```

### Repository Interface (Port)

Repository interfaces are defined in the domain layer - they are **ports**.

```go
// domain/repository.go
package domain

import (
    "context"

    "myapp/modules/users/domain/entity"
    "myapp/modules/users/domain/valueobject"
)

// UserRepository defines the contract for user persistence.
// This is a PORT - the interface lives in the domain.
// Implementation (adapter) lives in infrastructure.
type UserRepository interface {
    Save(ctx context.Context, user *entity.User) error
    FindByID(ctx context.Context, id entity.UserID) (*entity.User, error)
    FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)
    Delete(ctx context.Context, id entity.UserID) error
    ExistsByEmail(ctx context.Context, email valueobject.Email) (bool, error)
}
```

## Layer 2: Application (Use Cases)

The application layer orchestrates domain entities to fulfill use cases. It contains application-specific business rules.

### Commands (Write Operations)

```go
// application/command/create_user.go
package command

import (
    "context"
    "fmt"

    "myapp/modules/users/application/port"
    "myapp/modules/users/domain"
    "myapp/modules/users/domain/entity"
    "myapp/modules/users/domain/valueobject"
)

// CreateUserCommand is the input for user creation.
type CreateUserCommand struct {
    Email string
    Name  string
}

// CreateUserHandler handles user creation use case.
type CreateUserHandler struct {
    repo     domain.UserRepository
    notifier port.UserNotifier
}

func NewCreateUserHandler(repo domain.UserRepository, notifier port.UserNotifier) *CreateUserHandler {
    return &CreateUserHandler{repo: repo, notifier: notifier}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (entity.UserID, error) {
    // 1. Create and validate value objects
    email, err := valueobject.NewEmail(cmd.Email)
    if err != nil {
        return "", fmt.Errorf("invalid email: %w", err)
    }

    // 2. Check business rules (uniqueness)
    exists, err := h.repo.ExistsByEmail(ctx, email)
    if err != nil {
        return "", fmt.Errorf("check email exists: %w", err)
    }
    if exists {
        return "", domain.ErrEmailAlreadyExists
    }

    // 3. Create domain entity
    user, err := entity.NewUser(email, cmd.Name)
    if err != nil {
        return "", fmt.Errorf("create user: %w", err)
    }

    // 4. Persist via repository
    if err := h.repo.Save(ctx, user); err != nil {
        return "", fmt.Errorf("save user: %w", err)
    }

    // 5. Side effects (notifications, events) - non-critical
    if err := h.notifier.NotifyUserCreated(ctx, user); err != nil {
        // Log but don't fail the operation
        slog.Warn("failed to send notification", "error", err, "user_id", user.ID())
    }

    return user.ID(), nil
}
```

```go
// application/command/activate_user.go
package command

import (
    "context"
    "fmt"

    "myapp/modules/users/domain"
    "myapp/modules/users/domain/entity"
)

type ActivateUserCommand struct {
    UserID string
}

type ActivateUserHandler struct {
    repo domain.UserRepository
}

func NewActivateUserHandler(repo domain.UserRepository) *ActivateUserHandler {
    return &ActivateUserHandler{repo: repo}
}

func (h *ActivateUserHandler) Handle(ctx context.Context, cmd ActivateUserCommand) error {
    // 1. Parse and validate ID
    id, err := entity.ParseUserID(cmd.UserID)
    if err != nil {
        return fmt.Errorf("invalid user id: %w", err)
    }

    // 2. Fetch entity
    user, err := h.repo.FindByID(ctx, id)
    if err != nil {
        return fmt.Errorf("find user: %w", err)
    }

    // 3. Execute domain behavior
    if err := user.Activate(); err != nil {
        return err // Domain error (e.g., invalid transition)
    }

    // 4. Persist changes
    if err := h.repo.Save(ctx, user); err != nil {
        return fmt.Errorf("save user: %w", err)
    }

    return nil
}
```

### Queries (Read Operations)

```go
// application/query/get_user.go
package query

import (
    "context"
    "fmt"
    "time"

    "myapp/modules/users/domain"
    "myapp/modules/users/domain/entity"
)

type GetUserQuery struct {
    UserID string
}

// UserDTO is a Data Transfer Object for read operations.
// DTOs are flat structures optimized for the consumer.
type UserDTO struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

type GetUserHandler struct {
    repo domain.UserRepository
}

func NewGetUserHandler(repo domain.UserRepository) *GetUserHandler {
    return &GetUserHandler{repo: repo}
}

func (h *GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (*UserDTO, error) {
    id, err := entity.ParseUserID(q.UserID)
    if err != nil {
        return nil, fmt.Errorf("invalid user id: %w", err)
    }

    user, err := h.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Map domain entity to DTO
    return &UserDTO{
        ID:        user.ID().String(),
        Email:     user.Email().String(),
        Name:      user.Name(),
        Status:    string(user.Status()),
        CreatedAt: user.CreatedAt(),
    }, nil
}
```

### Application Ports (Output)

```go
// application/port/notifier.go
package port

import (
    "context"

    "myapp/modules/users/domain/entity"
)

// UserNotifier is an output port for user notifications.
// Implementation will be in infrastructure layer.
type UserNotifier interface {
    NotifyUserCreated(ctx context.Context, user *entity.User) error
    NotifyPasswordReset(ctx context.Context, user *entity.User, resetToken string) error
}
```

## Layer 3: Infrastructure (Adapters)

The infrastructure layer implements interfaces defined in inner layers.

### Repository Implementation (Adapter)

```go
// infrastructure/persistence/postgres/user_repository.go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "time"

    "myapp/modules/users/domain"
    "myapp/modules/users/domain/entity"
    "myapp/modules/users/domain/valueobject"
)

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

// Save implements domain.UserRepository
func (r *UserRepository) Save(ctx context.Context, user *entity.User) error {
    query := `
        INSERT INTO users (id, email, name, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            email = EXCLUDED.email,
            name = EXCLUDED.name,
            status = EXCLUDED.status,
            updated_at = EXCLUDED.updated_at
    `
    _, err := r.db.ExecContext(ctx, query,
        user.ID().String(),
        user.Email().String(),
        user.Name(),
        string(user.Status()),
        user.CreatedAt(),
        user.UpdatedAt(),
    )
    return err
}

// FindByID implements domain.UserRepository
func (r *UserRepository) FindByID(ctx context.Context, id entity.UserID) (*entity.User, error) {
    query := `SELECT id, email, name, status, created_at, updated_at FROM users WHERE id = $1`

    var (
        idStr, email, name, status string
        createdAt, updatedAt       time.Time
    )

    err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
        &idStr, &email, &name, &status, &createdAt, &updatedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrUserNotFound
        }
        return nil, err
    }

    return entity.ReconstructUser(idStr, email, name, status, createdAt, updatedAt)
}

// FindByEmail implements domain.UserRepository
func (r *UserRepository) FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error) {
    query := `SELECT id, email, name, status, created_at, updated_at FROM users WHERE email = $1`

    var (
        idStr, emailStr, name, status string
        createdAt, updatedAt          time.Time
    )

    err := r.db.QueryRowContext(ctx, query, email.String()).Scan(
        &idStr, &emailStr, &name, &status, &createdAt, &updatedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrUserNotFound
        }
        return nil, err
    }

    return entity.ReconstructUser(idStr, emailStr, name, status, createdAt, updatedAt)
}

// ExistsByEmail implements domain.UserRepository
func (r *UserRepository) ExistsByEmail(ctx context.Context, email valueobject.Email) (bool, error) {
    query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
    var exists bool
    err := r.db.QueryRowContext(ctx, query, email.String()).Scan(&exists)
    return exists, err
}

// Delete implements domain.UserRepository
func (r *UserRepository) Delete(ctx context.Context, id entity.UserID) error {
    query := `DELETE FROM users WHERE id = $1`
    result, err := r.db.ExecContext(ctx, query, id.String())
    if err != nil {
        return err
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return domain.ErrUserNotFound
    }
    return nil
}
```

### External Service Adapter

```go
// infrastructure/external/sendgrid/notifier.go
package sendgrid

import (
    "context"
    "fmt"

    "github.com/sendgrid/sendgrid-go"
    "github.com/sendgrid/sendgrid-go/helpers/mail"

    "myapp/modules/users/application/port"
    "myapp/modules/users/domain/entity"
)

type EmailNotifier struct {
    client *sendgrid.Client
    from   string
}

func NewEmailNotifier(apiKey, fromEmail string) *EmailNotifier {
    return &EmailNotifier{
        client: sendgrid.NewSendClient(apiKey),
        from:   fromEmail,
    }
}

// NotifyUserCreated implements port.UserNotifier
func (n *EmailNotifier) NotifyUserCreated(ctx context.Context, user *entity.User) error {
    from := mail.NewEmail("MyApp", n.from)
    to := mail.NewEmail(user.Name(), user.Email().String())
    subject := "Welcome to MyApp!"
    content := fmt.Sprintf("Hello %s, welcome to our platform!", user.Name())

    message := mail.NewSingleEmail(from, subject, to, content, content)
    _, err := n.client.Send(message)
    return err
}
```

## Layer 4: Interface (Input Ports)

The interface layer handles input/output with the external world.

### HTTP Handler

```go
// ports/http/handler.go
package http

import (
    "encoding/json"
    "errors"
    "net/http"

    "myapp/modules/users/application/command"
    "myapp/modules/users/application/query"
    "myapp/modules/users/domain"
)

type UserHandler struct {
    createUser   *command.CreateUserHandler
    activateUser *command.ActivateUserHandler
    getUser      *query.GetUserHandler
}

func NewUserHandler(
    createUser *command.CreateUserHandler,
    activateUser *command.ActivateUserHandler,
    getUser *query.GetUserHandler,
) *UserHandler {
    return &UserHandler{
        createUser:   createUser,
        activateUser: activateUser,
        getUser:      getUser,
    }
}

// DTOs for HTTP layer
type CreateUserRequest struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

type CreateUserResponse struct {
    ID string `json:"id"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    id, err := h.createUser.Handle(r.Context(), command.CreateUserCommand{
        Email: req.Email,
        Name:  req.Name,
    })
    if err != nil {
        handleDomainError(w, err)
        return
    }

    writeJSON(w, CreateUserResponse{ID: id.String()}, http.StatusCreated)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := r.PathValue("id") // Go 1.22+

    user, err := h.getUser.Handle(r.Context(), query.GetUserQuery{UserID: userID})
    if err != nil {
        handleDomainError(w, err)
        return
    }

    writeJSON(w, user, http.StatusOK)
}

func handleDomainError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, domain.ErrUserNotFound):
        writeError(w, "user not found", http.StatusNotFound)
    case errors.Is(err, domain.ErrEmailAlreadyExists):
        writeError(w, "email already exists", http.StatusConflict)
    case errors.Is(err, domain.ErrInvalidStatusTransition):
        writeError(w, "invalid operation", http.StatusBadRequest)
    default:
        writeError(w, "internal error", http.StatusInternalServerError)
    }
}

func writeJSON(w http.ResponseWriter, data any, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, message string, status int) {
    writeJSON(w, ErrorResponse{Error: message}, status)
}
```

## Dependency Injection & Wiring

```go
// module.go
package users

import (
    "database/sql"
    "log/slog"

    "myapp/modules/users/application/command"
    "myapp/modules/users/application/query"
    "myapp/modules/users/infrastructure/external/sendgrid"
    "myapp/modules/users/infrastructure/persistence/postgres"
    httpport "myapp/modules/users/ports/http"
)

type Dependencies struct {
    DB           *sql.DB
    SendGridKey  string
    FromEmail    string
    Logger       *slog.Logger
}

func New(deps Dependencies) (*httpport.UserHandler, error) {
    // Infrastructure layer
    userRepo := postgres.NewUserRepository(deps.DB)
    notifier := sendgrid.NewEmailNotifier(deps.SendGridKey, deps.FromEmail)

    // Application layer
    createUserHandler := command.NewCreateUserHandler(userRepo, notifier)
    activateUserHandler := command.NewActivateUserHandler(userRepo)
    getUserHandler := query.NewGetUserHandler(userRepo)

    // Interface layer
    httpHandler := httpport.NewUserHandler(
        createUserHandler,
        activateUserHandler,
        getUserHandler,
    )

    return httpHandler, nil
}
```

## Testing Strategy

### Domain Layer: Unit Tests (No Mocks)

```go
func TestUser_Activate(t *testing.T) {
    email, _ := valueobject.NewEmail("test@example.com")
    user, _ := entity.NewUser(email, "John Doe")

    err := user.Activate()

    assert.NoError(t, err)
    assert.Equal(t, entity.StatusActive, user.Status())
}

func TestUser_Activate_WhenAlreadyActive_Fails(t *testing.T) {
    email, _ := valueobject.NewEmail("test@example.com")
    user, _ := entity.NewUser(email, "John Doe")
    user.Activate() // First activation

    err := user.Activate() // Second activation

    assert.ErrorIs(t, err, entity.ErrInvalidStatusTransition)
}
```

### Application Layer: Unit Tests with Mocks

```go
func TestCreateUserHandler(t *testing.T) {
    repo := mocks.NewUserRepository(t)
    notifier := mocks.NewUserNotifier(t)
    handler := command.NewCreateUserHandler(repo, notifier)

    repo.EXPECT().ExistsByEmail(mock.Anything, mock.Anything).Return(false, nil)
    repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
    notifier.EXPECT().NotifyUserCreated(mock.Anything, mock.Anything).Return(nil)

    id, err := handler.Handle(context.Background(), command.CreateUserCommand{
        Email: "test@example.com",
        Name:  "Test User",
    })

    assert.NoError(t, err)
    assert.NotEmpty(t, id)
}
```

### Infrastructure Layer: Integration Tests

```go
func TestPostgresUserRepository_Save(t *testing.T) {
    db := testutil.NewTestDB(t) // Real database
    repo := postgres.NewUserRepository(db)

    email, _ := valueobject.NewEmail("test@example.com")
    user, _ := entity.NewUser(email, "John Doe")

    err := repo.Save(context.Background(), user)
    require.NoError(t, err)

    found, err := repo.FindByID(context.Background(), user.ID())
    require.NoError(t, err)
    assert.Equal(t, user.Email(), found.Email())
}
```

## Common Violations to Avoid

### 1. Framework in Domain

```go
// WRONG: Domain depends on HTTP framework
package entity

import "github.com/gin-gonic/gin"

func (u *User) Respond(c *gin.Context) {
    c.JSON(200, u)
}

// CORRECT: Domain is pure - no framework dependencies
package entity

func (u *User) Name() string { return u.name }
```

### 2. Database in Application Layer

```go
// WRONG: Application knows about SQL
func (h *Handler) Handle(ctx context.Context, cmd Cmd) error {
    _, err := h.db.Exec("INSERT INTO users...")
    return err
}

// CORRECT: Application uses repository interface
func (h *Handler) Handle(ctx context.Context, cmd Cmd) error {
    return h.repo.Save(ctx, user)
}
```

### 3. Domain Entities in HTTP Response

```go
// WRONG: Exposing domain entity directly
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, _ := h.repo.FindByID(ctx, id)
    json.NewEncoder(w).Encode(user) // Leaks internal structure!
}

// CORRECT: Map to DTO
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, _ := h.repo.FindByID(ctx, id)
    dto := mapToDTO(user)
    json.NewEncoder(w).Encode(dto)
}
```

## Related Skills

- `/modular-monolith` - Module structure containing these layers
- `/cohesion-coupling` - Why this separation matters
- `/philosophy-software-design` - Deep modules concept
- `/go-best-practices` - Go implementation patterns
