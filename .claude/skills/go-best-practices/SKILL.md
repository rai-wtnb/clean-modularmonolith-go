---
name: go-best-practices
description: Idiomatic Go programming patterns and conventions. Use when writing Go code, designing interfaces, handling errors, structuring packages, naming things, or reviewing Go code quality. Covers interface design, error handling, concurrency patterns, and effective Go style.
---

# Go Best Practices

Idiomatic Go patterns for clean, maintainable code.

## Interface Design

### Accept Interfaces, Return Structs

```go
// GOOD: Accept interface (flexible for callers)
func ProcessData(r io.Reader) (*Result, error) {
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("read data: %w", err)
    }
    // Process...
    return &Result{}, nil
}

// Can be called with any Reader
ProcessData(file)
ProcessData(bytes.NewReader(data))
ProcessData(response.Body)
ProcessData(strings.NewReader("test"))

// GOOD: Return concrete type (provides full API)
func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

### Small Interfaces

Follow Go standard library style - interfaces should be small.

```go
// Go standard library examples
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Compose when needed
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}

// Your interfaces should be small too
type UserFinder interface {
    FindByID(ctx context.Context, id string) (*User, error)
}

type UserSaver interface {
    Save(ctx context.Context, user *User) error
}

// NOT: Large interfaces that force clients to implement everything
type UserStore interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindAll(ctx context.Context) ([]User, error)
    Save(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    Count(ctx context.Context) (int, error)
    // ... 10 more methods
}
```

### Define Interfaces Where Used

Define interfaces in the **consumer** package, not the provider.

```go
// package order (consumer)
// Define the interface you need
type UserGetter interface {
    GetUser(ctx context.Context, id string) (*user.User, error)
}

type OrderService struct {
    users UserGetter  // Only needs GetUser
}

// package user (provider)
// No interface needed - just the implementation
type Service struct {
    repo Repository
}

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) { ... }
func (s *Service) CreateUser(ctx context.Context, u *User) error { ... }
func (s *Service) DeleteUser(ctx context.Context, id string) error { ... }

// user.Service implicitly satisfies order.UserGetter
```

## Error Handling

### Return Errors, Don't Panic

```go
// GOOD: Return error
func GetUser(ctx context.Context, id string) (*User, error) {
    user, err := repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("find user %s: %w", id, err)
    }
    return user, nil
}

// BAD: Panic for recoverable errors
func GetUser(ctx context.Context, id string) *User {
    user, err := repo.FindByID(ctx, id)
    if err != nil {
        panic(err)  // Don't do this!
    }
    return user
}

// Panic is OK for:
// - Programmer errors (nil pointer that should never be nil)
// - Initialization failures (can't start the app anyway)
// - "Must" functions with compile-time known inputs

func MustCompile(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

var emailPattern = MustCompile(`^[a-z]+@[a-z]+\.[a-z]+$`)  // OK at init time
```

### Error Wrapping

```go
// Wrap errors with context using %w
func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
    user, err := s.users.GetUser(ctx, cmd.UserID)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)  // Wrap with context
    }

    order, err := NewOrder(user, cmd.Items)
    if err != nil {
        return nil, fmt.Errorf("create order: %w", err)
    }

    if err := s.repo.Save(ctx, order); err != nil {
        return nil, fmt.Errorf("save order: %w", err)
    }

    return order, nil
}

// Check wrapped errors
if errors.Is(err, ErrUserNotFound) {
    // Handle not found specifically
}

// Get underlying error type
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // Access validation error fields
    fmt.Println(validationErr.Field, validationErr.Message)
}
```

### Sentinel Errors

```go
// Define package-level sentinel errors
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
)

// Use in functions
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    row := r.db.QueryRowContext(ctx, "SELECT ... WHERE id = $1", id)
    var user User
    if err := row.Scan(&user.ID, &user.Name); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound  // Translate to domain error
        }
        return nil, fmt.Errorf("scan user: %w", err)
    }
    return &user, nil
}

// Callers check
user, err := repo.FindByID(ctx, id)
if errors.Is(err, ErrNotFound) {
    // Handle missing user
}
```

### Custom Error Types

```go
// Custom error with structured information
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// Usage
func ValidateUser(u User) error {
    if u.Email == "" {
        return &ValidationError{Field: "email", Message: "required"}
    }
    if u.Age < 0 {
        return &ValidationError{Field: "age", Message: "must be non-negative"}
    }
    return nil
}

// Handling
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // Return 400 with field-specific error
    return c.JSON(400, map[string]string{
        "field":   validationErr.Field,
        "message": validationErr.Message,
    })
}
```

### Error Handling at Boundaries

```go
// Don't log and return - choose one
func (s *Service) DoSomething(ctx context.Context) error {
    err := s.repo.Save(ctx, data)
    if err != nil {
        // BAD: Logging AND returning causes duplicate logs
        log.Error("failed to save", "error", err)
        return err
    }
    return nil
}

// GOOD: Return error, let caller decide
func (s *Service) DoSomething(ctx context.Context) error {
    if err := s.repo.Save(ctx, data); err != nil {
        return fmt.Errorf("save data: %w", err)  // Just wrap and return
    }
    return nil
}

// Log at boundaries (HTTP handler, main, etc.)
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    err := h.service.DoSomething(r.Context())
    if err != nil {
        slog.Error("request failed", "error", err, "path", r.URL.Path)
        http.Error(w, "internal error", http.StatusInternalServerError)
    }
}
```

## Package Design

### Package Naming

```go
// Package names: short, lowercase, no underscores or camelCase
package user       // GOOD
package httputil   // GOOD
package strconv    // GOOD (standard library)

package user_service  // BAD: underscore
package userService   // BAD: camelCase
package utils         // BAD: too generic
package common        // BAD: too generic
package base          // BAD: too generic
```

### Package Structure

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go           # Entry point only
├── internal/                  # Private to this module
│   ├── user/
│   │   ├── user.go           # Types and core logic
│   │   ├── repository.go     # Data access interface
│   │   ├── service.go        # Business logic
│   │   └── handler.go        # HTTP handlers
│   └── order/
│       └── ...
├── pkg/                       # Public, importable by others
│   └── httputil/
│       └── response.go
└── go.mod
```

### Avoid Circular Imports

```go
// PROBLEM: Circular dependency
// package user imports package order
// package order imports package user

// SOLUTION 1: Extract shared types
// package types (shared by both)
type UserID string

// package user
import "myapp/types"
func GetUser(id types.UserID) (*User, error)

// package order
import "myapp/types"
func CreateOrder(userID types.UserID) (*Order, error)

// SOLUTION 2: Use interfaces (dependency inversion)
// package order defines what it needs
type UserGetter interface {
    GetUser(ctx context.Context, id string) (*User, error)
}

// package user implements it (without importing order)
func (s *Service) GetUser(ctx context.Context, id string) (*User, error)

// Wire together at composition root (main or app package)
orderService := order.NewService(userService)
```

## Naming Conventions

### Variables and Functions

```go
// Short names for short scope
for i, v := range items {
    process(v)
}

// Descriptive names for larger scope
userRepository := NewUserRepository(db)
orderProcessingService := NewOrderProcessingService(deps)

// Avoid stuttering (package.Type repetition)
user.User        // Acceptable (constructor often returns this)
user.UserService // BAD: stutter
user.Service     // GOOD

http.HTTPClient  // BAD: stutter
http.Client      // GOOD (actual standard library)

// Boolean naming - use positive form
isActive := true      // GOOD
isNotDisabled := true // BAD: double negative

hasPermission := checkPermission(user)
canEdit := user.Role == Admin
shouldRetry := attempts < maxAttempts
```

### Function Names

```go
// verb + noun pattern
func GetUser(id string) (*User, error)
func CreateOrder(cmd CreateOrderCommand) (*Order, error)
func ValidateEmail(email string) error
func ParseConfig(data []byte) (*Config, error)
func IsValidEmail(email string) bool

// No Get prefix for simple getters (Go convention)
type User struct {
    name string
}

func (u *User) Name() string { return u.name }  // GOOD
func (u *User) GetName() string { return u.name }  // Less idiomatic

// Set prefix is fine for setters
func (u *User) SetName(name string) { u.name = name }
```

### Acronyms

```go
// All caps for acronyms in identifiers
var userID string     // GOOD
var httpClient *http.Client  // GOOD
var xmlParser *XMLParser     // GOOD

var userId string     // BAD
var xmlparser *Xmlparser  // BAD

// Exported identifiers start with capital
type HTTPHandler struct{}  // Exported
type httpHandler struct{}  // Unexported
```

## Context Usage

### Always First Parameter

```go
// Context is always the first parameter
func GetUser(ctx context.Context, id string) (*User, error) {
    // ...
}

func (s *Service) ProcessOrder(ctx context.Context, cmd Command) error {
    // ...
}

// Never put context in struct
type Service struct {
    ctx context.Context  // BAD!
}
```

### Propagate Context

```go
// Always pass context through the call chain
func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
    // Pass ctx to all downstream calls
    user, err := s.users.GetUser(ctx, cmd.UserID)
    if err != nil {
        return nil, err
    }

    order := NewOrder(user, cmd.Items)

    // Pass ctx to repository
    if err := s.repo.Save(ctx, order); err != nil {
        return nil, err
    }

    // Pass ctx to event publishing
    s.events.Publish(ctx, OrderCreated{OrderID: order.ID})

    return order, nil
}
```

### Context Values (Use Sparingly)

```go
// Use context values for request-scoped data that crosses API boundaries
type contextKey string

const (
    requestIDKey contextKey = "requestID"
    userIDKey    contextKey = "userID"
)

// Set value
ctx = context.WithValue(ctx, requestIDKey, "req-123")

// Get value (always check type assertion)
if reqID, ok := ctx.Value(requestIDKey).(string); ok {
    // Use reqID
}

// Better: Use typed helpers
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) string {
    if id, ok := ctx.Value(requestIDKey).(string); ok {
        return id
    }
    return ""
}
```

## Struct Design

### Constructor Functions

```go
// Use New prefix for constructors
func NewUserService(repo UserRepository, logger *slog.Logger) *UserService {
    return &UserService{
        repo:   repo,
        logger: logger,
    }
}

// Validate in constructor
func NewEmail(value string) (Email, error) {
    normalized := strings.ToLower(strings.TrimSpace(value))
    if !isValidEmail(normalized) {
        return Email{}, ErrInvalidEmail
    }
    return Email{value: normalized}, nil
}

// Must variant for known-good values
func MustNewEmail(value string) Email {
    email, err := NewEmail(value)
    if err != nil {
        panic(fmt.Sprintf("invalid email: %s", value))
    }
    return email
}
```

### Functional Options Pattern

```go
// For optional configuration
type Server struct {
    addr         string
    readTimeout  time.Duration
    writeTimeout time.Duration
    logger       *slog.Logger
    middleware   []Middleware
}

// Option is a function that configures Server
type Option func(*Server)

func WithReadTimeout(d time.Duration) Option {
    return func(s *Server) {
        s.readTimeout = d
    }
}

func WithLogger(l *slog.Logger) Option {
    return func(s *Server) {
        s.logger = l
    }
}

func WithMiddleware(mw ...Middleware) Option {
    return func(s *Server) {
        s.middleware = append(s.middleware, mw...)
    }
}

func NewServer(addr string, opts ...Option) *Server {
    // Start with sensible defaults
    s := &Server{
        addr:         addr,
        readTimeout:  30 * time.Second,
        writeTimeout: 30 * time.Second,
        logger:       slog.Default(),
    }

    // Apply options
    for _, opt := range opts {
        opt(s)
    }

    return s
}

// Usage - clean and readable
server := NewServer(":8080",
    WithReadTimeout(60*time.Second),
    WithLogger(customLogger),
    WithMiddleware(authMiddleware, loggingMiddleware),
)
```

### Zero Value Usefulness

```go
// Design structs so zero value is useful
type Buffer struct {
    data []byte
}

func (b *Buffer) Write(p []byte) (int, error) {
    b.data = append(b.data, p...)  // Works with nil slice
    return len(p), nil
}

// Can be used without explicit initialization
var buf Buffer
buf.Write([]byte("hello"))  // Works!

// sync.Mutex is a great example
type SafeCounter struct {
    mu    sync.Mutex  // Zero value is ready to use
    count int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

## Concurrency

### Don't Start Goroutines in Libraries

```go
// BAD: Library starts goroutine - caller can't control lifecycle
func NewWatcher(path string) *Watcher {
    w := &Watcher{path: path}
    go w.watch()  // Who stops this?
    return w
}

// GOOD: Let caller control goroutine lifecycle
func NewWatcher(path string) *Watcher {
    return &Watcher{path: path}
}

func (w *Watcher) Start(ctx context.Context) error {
    // Caller decides when to start
    // Context controls cancellation
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            w.checkForChanges()
        }
    }
}

// Caller controls lifecycle
ctx, cancel := context.WithCancel(context.Background())
go watcher.Start(ctx)
// Later...
cancel()  // Stop the watcher
```

### sync.Mutex Patterns

```go
// Embed mutex, keep it unexported
type SafeMap struct {
    mu   sync.RWMutex  // Unexported
    data map[string]any
}

func (m *SafeMap) Get(key string) (any, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    v, ok := m.data[key]
    return v, ok
}

func (m *SafeMap) Set(key string, value any) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.data == nil {
        m.data = make(map[string]any)
    }
    m.data[key] = value
}

// sync.Once for one-time initialization
type Client struct {
    conn     *grpc.ClientConn
    connOnce sync.Once
    connErr  error
}

func (c *Client) getConn() (*grpc.ClientConn, error) {
    c.connOnce.Do(func() {
        c.conn, c.connErr = grpc.Dial(c.addr)
    })
    return c.conn, c.connErr
}
```

### Channel Patterns

```go
// Signal completion with empty struct channel
done := make(chan struct{})
go func() {
    defer close(done)
    // Do work...
}()
<-done  // Wait for completion

// Fan-out pattern
func process(jobs <-chan Job) <-chan Result {
    results := make(chan Result)
    go func() {
        defer close(results)
        for job := range jobs {
            results <- doWork(job)
        }
    }()
    return results
}

// Bounded concurrency with semaphore
sem := make(chan struct{}, maxConcurrency)
for _, item := range items {
    sem <- struct{}{}  // Acquire
    go func(item Item) {
        defer func() { <-sem }()  // Release
        process(item)
    }(item)
}
```

## Testing

### Table-Driven Tests

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"empty string", "", true},
        {"no domain", "user@", true},
        {"no at sign", "userexample.com", true},
        {"multiple at signs", "user@@example.com", true},
        {"valid with subdomain", "user@mail.example.com", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v",
                    tt.input, err, tt.wantErr)
            }
        })
    }
}
```

### Test Helpers

```go
func TestUserService(t *testing.T) {
    svc, cleanup := setupTestService(t)
    defer cleanup()

    // Test...
}

func setupTestService(t *testing.T) (*UserService, func()) {
    t.Helper()  // Mark as helper - errors report caller's line

    db := setupTestDB(t)
    repo := NewUserRepository(db)
    svc := NewUserService(repo)

    cleanup := func() {
        db.Close()
    }

    return svc, cleanup
}

// With Go 1.14+ use t.Cleanup
func setupTestService(t *testing.T) *UserService {
    t.Helper()

    db := setupTestDB(t)
    t.Cleanup(func() { db.Close() })  // Automatic cleanup

    return NewUserService(NewUserRepository(db))
}
```

## Logging with slog

```go
import "log/slog"

// Structured logging
slog.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
)

// With error
slog.Error("failed to process order",
    "error", err,
    "order_id", orderID,
)

// Create logger with context
logger := slog.With(
    "service", "orders",
    "version", "1.0.0",
)
logger.Info("service started")

// Use in structs
type OrderService struct {
    repo   OrderRepository
    logger *slog.Logger
}

func NewOrderService(repo OrderRepository, logger *slog.Logger) *OrderService {
    return &OrderService{
        repo:   repo,
        logger: logger.With("component", "order_service"),
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
    s.logger.Info("creating order",
        "user_id", cmd.UserID,
        "items", len(cmd.Items),
    )
    // ...
}
```

## Related Skills

- `/clean-architecture` - Where these patterns apply
- `/go-workspace` - Multi-module setup
- `/modular-monolith` - Module structure
- `/cohesion-coupling` - Design principles
