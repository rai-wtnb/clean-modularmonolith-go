---
name: philosophy-of-software-design
description: John Ousterhout's software design philosophy for managing complexity. Use when making interface design decisions, deciding what to hide in modules, evaluating deep vs shallow designs, writing comments, or applying strategic vs tactical programming. Covers complexity reduction, deep modules, and information hiding.
---

# A Philosophy of Software Design

Based on John Ousterhout's principles for managing complexity in software systems. The central goal is to reduce complexity to make software easier to understand and modify.

## Core Insight: Complexity is the Root Problem

Complexity is anything that makes software hard to understand or modify. It manifests in three ways:

### 1. Change Amplification

A small change requires modifications in many places.

```go
// HIGH COMPLEXITY: Changing date format requires edits everywhere
func FormatUserCreatedAt(u User) string {
    return u.CreatedAt.Format("2006-01-02")
}
func FormatOrderCreatedAt(o Order) string {
    return o.CreatedAt.Format("2006-01-02")
}
func FormatPaymentCreatedAt(p Payment) string {
    return p.CreatedAt.Format("2006-01-02")
}

// LOW COMPLEXITY: Single place to change
const DateFormat = "2006-01-02"

func FormatDate(t time.Time) string {
    return t.Format(DateFormat)
}
```

### 2. Cognitive Load

Too much information needed to make a change safely.

```go
// HIGH COGNITIVE LOAD: Must understand all these to make changes
type OrderService struct {
    db              *sql.DB
    cache           *redis.Client
    queue           *amqp.Channel
    paymentGateway  *stripe.Client
    inventoryClient *grpc.ClientConn
    emailService    *smtp.Client
    config          *Config
    metrics         *prometheus.Registry
    tracer          *opentelemetry.Tracer
}

// LOWER COGNITIVE LOAD: Dependencies abstracted behind interfaces
type OrderService struct {
    orders   OrderRepository
    payments PaymentProcessor
    notifier OrderNotifier
}
```

### 3. Unknown Unknowns

Not obvious what code needs to change, or what might break.

```go
// UNKNOWN UNKNOWNS: Side effects not obvious from signature
func ProcessOrder(order *Order) error {
    // Secretly modifies global state
    globalOrderCount++
    // Secretly sends email
    sendConfirmationEmail(order)
    // Secretly writes to cache
    orderCache[order.ID] = order
    return nil
}

// EXPLICIT: Effects clear from signature
func (s *OrderProcessor) Process(ctx context.Context, order *Order) (Receipt, error) {
    // All effects visible through injected dependencies
}
```

## The Deep Module Concept

### What Makes a Module "Deep"?

A deep module provides **powerful functionality behind a simple interface**.

```
Simple Interface (few methods, easy to understand)
    │
    ▼
┌─────────────────────────────────────────────────┐
│                                                 │
│                                                 │
│           Complex Implementation                │
│        (lots of code, many details)            │
│                                                 │
│                                                 │
└─────────────────────────────────────────────────┘
```

The benefit: Users of the module get powerful capabilities without needing to understand the implementation.

### Deep Module Example: Go's `io.Reader`

```go
// DEEP: Incredibly simple interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Hides enormous complexity:
// - Buffering strategies
// - File system operations
// - Network protocols
// - Compression/decompression
// - Encryption/decryption
// - Error recovery
// - Platform differences
```

### Deep Module Example in Application Code

```go
// DEEP: Simple interface, complex implementation
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
}

// Implementation hides:
// - Connection pooling
// - Serialization format
// - Eviction policy (LRU, LFU, TTL)
// - Cluster topology
// - Failover handling
// - Retry logic
// - Metrics collection
```

### Shallow Module Anti-Pattern

A shallow module has a **complex interface relative to its functionality**.

```
Complex Interface (many methods, lots to learn)
    │
    ▼
┌─────────────────────────────────────────────────┐
│         Small Implementation                    │
└─────────────────────────────────────────────────┘
```

```go
// SHALLOW: Interface complexity matches implementation
type UserValidator interface {
    ValidateEmail(email string) error
    ValidateName(name string) error
    ValidateAge(age int) error
    ValidatePhone(phone string) error
    ValidateAddress(addr Address) error
    ValidatePassword(password string) error
    ValidateUsername(username string) error
}

// Each method is trivial - no abstraction benefit
func (v *validator) ValidateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return ErrInvalidEmail
    }
    return nil
}

// DEEP: Single method hides validation complexity
type UserValidator interface {
    Validate(user User) ValidationResult
}

// Implementation handles all validations, returns comprehensive result
func (v *validator) Validate(user User) ValidationResult {
    var errs []ValidationError
    // All validations in one place
    // Can add new validations without changing interface
    return ValidationResult{Errors: errs}
}
```

## Information Hiding

### Principle: Hide Implementation Decisions

Modules should hide internal decisions so callers don't depend on them.

```go
// BAD: Leaks storage decision
type UserStore interface {
    GetFromRedis(key string) (*User, error)
    SaveToRedis(user *User) error
}

// BAD: Leaks caching policy
type OrderCache interface {
    GetWithLRUEviction(key string) (*Order, error)
}

// GOOD: Hides all implementation decisions
type UserRepository interface {
    FindByID(ctx context.Context, id UserID) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

### What to Hide

1. **Data structures and formats** - JSON, protobuf, internal structs
2. **Algorithms** - Sorting, searching, optimization strategies
3. **External dependencies** - Redis, Postgres, third-party APIs
4. **Error handling strategies** - Retries, fallbacks, circuit breakers
5. **Configuration details** - Timeouts, pool sizes, thresholds

### Information Leakage Signs

```go
// LEAKY: Parameter names reveal implementation
func FetchUser(redisKey string, postgresID int64) (*User, error)

// LEAKY: Return type reveals storage
func GetOrders() *sql.Rows

// LEAKY: Error reveals implementation
if err == redis.Nil {
    // Caller knows we use Redis
}

// BETTER: Abstract errors
var ErrNotFound = errors.New("not found")
if errors.Is(err, ErrNotFound) {
    // Caller doesn't know storage type
}
```

## Pull Complexity Downward

When you have complexity, push it into the implementation, not the interface.

### Before: Complexity Pushed to Caller

```go
// SHALLOW: Caller handles all complexity
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Caller must handle:
func fetchUser(client HTTPClient, url string) (*User, error) {
    req, _ := http.NewRequest("GET", url, nil)

    var user *User
    for attempt := 0; attempt < 3; attempt++ {  // Retries
        resp, err := client.Do(req)
        if err != nil {
            time.Sleep(time.Second * time.Duration(attempt+1))  // Backoff
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == 429 {  // Rate limiting
            time.Sleep(time.Second * 5)
            continue
        }

        if resp.StatusCode != 200 {
            return nil, fmt.Errorf("status: %d", resp.StatusCode)
        }

        body, _ := io.ReadAll(resp.Body)
        json.Unmarshal(body, &user)
        break
    }
    return user, nil
}
```

### After: Complexity Pulled Into Implementation

```go
// DEEP: Implementation handles complexity
type UserFetcher interface {
    Fetch(ctx context.Context, userID string) (*User, error)
}

type userFetcher struct {
    baseURL    string
    httpClient *http.Client
    retrier    *Retrier
}

func (f *userFetcher) Fetch(ctx context.Context, userID string) (*User, error) {
    // All complexity hidden inside:
    // - URL construction
    // - Retries with backoff
    // - Rate limit handling
    // - Response parsing
    // - Error wrapping
}

// Caller code is simple:
user, err := fetcher.Fetch(ctx, "123")
```

### Configuration Example

```go
// SHALLOW: Many config options exposed
func NewServer(
    addr string,
    readTimeout time.Duration,
    writeTimeout time.Duration,
    idleTimeout time.Duration,
    maxHeaderBytes int,
    maxConns int,
    tlsConfig *tls.Config,
    logger *slog.Logger,
) *Server

// DEEP: Sensible defaults, few options
func NewServer(addr string, opts ...Option) *Server {
    s := &Server{
        addr:         addr,
        readTimeout:  30 * time.Second,   // Good default
        writeTimeout: 30 * time.Second,
        maxConns:     10000,
        logger:       slog.Default(),
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Only override when needed
server := NewServer(":8080", WithLogger(myLogger))
```

## Define Errors Out of Existence

Reduce exception handling by designing APIs where errors can't occur.

### Before: Errors Possible

```go
func Substring(s string, start, end int) (string, error) {
    if start < 0 {
        return "", errors.New("start cannot be negative")
    }
    if end > len(s) {
        return "", errors.New("end exceeds string length")
    }
    if start > end {
        return "", errors.New("start cannot exceed end")
    }
    return s[start:end], nil
}

// Every caller must handle errors
sub, err := Substring(s, 0, 10)
if err != nil {
    // Handle...
}
```

### After: Errors Defined Away

```go
func Substring(s string, start, end int) string {
    // Automatically handle edge cases
    if start < 0 {
        start = 0
    }
    if end > len(s) {
        end = len(s)
    }
    if start > end {
        return ""
    }
    return s[start:end]
}

// Caller code is simple
sub := Substring(s, 0, 10)
```

### Go's Built-in Examples

```go
// Map access returns zero value - no error
value := myMap[key]  // Returns "" if key missing

// Slice within bounds automatically via min/max
end := min(requestedEnd, len(slice))

// Close is idempotent - safe to call multiple times
file.Close()
file.Close()  // No error
```

### When to Apply This

Apply "define errors out of existence" when:
- Edge cases have sensible default behaviors
- The "error" represents normal variation, not true failure
- Callers would all handle the error the same way

Don't apply when:
- The error represents genuine failure that needs handling
- Different callers need different error responses
- Silent failure would mask bugs

## General-Purpose vs Special-Purpose

### The Sweet Spot: Somewhat General-Purpose

Design interfaces general enough to serve multiple uses, but specific enough to be easy to use.

```go
// TOO SPECIAL: Only one use case
type UserConfigReader interface {
    ReadUserConfig() (*UserConfig, error)
}

// TOO GENERAL: Hard to use for anything
type DataProcessor interface {
    Process(format string, data []byte, options map[string]any) (any, error)
}

// JUST RIGHT: General yet usable
type ConfigReader interface {
    Read(path string) (Config, error)
}

// Can read user config, app config, feature flags, etc.
```

### The Rule of Three

Before building something reusable:
1. Build it for the first use case
2. When second use case appears, note the pattern
3. On third use case, extract the general solution

```go
// First use: inline
func processUserOrder(order UserOrder) { ... }

// Second use: still inline, note similarity
func processGuestOrder(order GuestOrder) { ... }

// Third use: now generalize
type Order interface {
    Items() []Item
    Total() Money
}

func processOrder(order Order) { ... }
```

## Different Layer, Different Abstraction

Each layer in your system should provide **meaningful abstraction**, not just pass-through.

### Anti-Pattern: Pass-Through Layers

```go
// BAD: Service just calls repository
type UserService struct {
    repo UserRepository
}

func (s *UserService) GetUser(id string) (*User, error) {
    return s.repo.GetUser(id)  // No added value!
}

func (s *UserService) SaveUser(u *User) error {
    return s.repo.SaveUser(u)  // Just forwarding!
}
```

### Good: Each Layer Adds Value

```go
// GOOD: Service adds business logic
type UserService struct {
    repo    UserRepository
    cache   UserCache
    events  EventPublisher
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // Layer adds: caching
    if cached := s.cache.Get(id); cached != nil {
        return cached, nil
    }

    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    s.cache.Set(id, user)
    return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, cmd CreateUserCommand) (*User, error) {
    // Layer adds: validation, business rules, events
    if exists, _ := s.repo.ExistsByEmail(ctx, cmd.Email); exists {
        return nil, ErrEmailTaken
    }

    user := NewUser(cmd.Email, cmd.Name)
    if err := s.repo.Save(ctx, user); err != nil {
        return nil, err
    }

    s.events.Publish(UserCreated{UserID: user.ID})
    return user, nil
}
```

## Strategic vs Tactical Programming

### Tactical Programming (Avoid)

- **Focus**: Get feature working as quickly as possible
- **Mindset**: "I'll clean it up later" (you won't)
- **Result**: Technical debt accumulates

```go
// TACTICAL: Quick but creates debt
func handleOrder(w http.ResponseWriter, r *http.Request) {
    // 300 lines of mixed concerns:
    // - Request parsing
    // - Validation
    // - Database queries (raw SQL inline)
    // - Business logic
    // - External API calls
    // - Response formatting
    // - Error handling (inconsistent)
}
```

### Strategic Programming (Prefer)

- **Focus**: Good design that enables future changes
- **Mindset**: "A little extra time now saves a lot later"
- **Investment**: ~10-20% extra time upfront

```go
// STRATEGIC: Clean, maintainable
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
    cmd, err := h.parseCreateCommand(r)
    if err != nil {
        h.writeError(w, err)
        return
    }

    order, err := h.orderService.Create(r.Context(), cmd)
    if err != nil {
        h.writeError(w, err)
        return
    }

    h.writeJSON(w, mapToResponse(order), http.StatusCreated)
}
```

### When Tactical is Acceptable

- Prototypes that will be thrown away
- One-time scripts
- Urgent production fixes (but schedule cleanup!)

## Comments: Describe What's Not Obvious

### Bad Comments: Repeat the Code

```go
// BAD: Adds no information
// GetUser gets a user by ID
func GetUser(id string) (*User, error)

// BAD: States the obvious
// Increment counter by 1
counter++

// BAD: Describes what, not why
// Set timeout to 30 seconds
timeout := 30 * time.Second
```

### Good Comments: Add Information

```go
// GOOD: Explains non-obvious behavior
// GetUser retrieves a user from cache if available, falling back
// to database. Returns ErrUserNotFound if the user doesn't exist
// or has been soft-deleted more than 30 days ago.
func GetUser(ctx context.Context, id string) (*User, error)

// GOOD: Explains why
// 30 second timeout chosen based on p99 latency of downstream
// service during peak load (measured Q4 2024).
timeout := 30 * time.Second

// GOOD: Documents edge cases
// ProcessBatch handles empty input gracefully, returning nil
// without error. Partial failures are collected and returned
// as a MultiError while successful items are still processed.
func ProcessBatch(items []Item) error
```

### Write Interface Comments First

Document interfaces before implementing them. This clarifies your design.

```go
// Step 1: Write interface with comprehensive comments
// OrderService handles the complete order lifecycle from creation
// through fulfillment. All methods are idempotent - calling them
// multiple times with the same input produces the same result.
type OrderService interface {
    // Create validates and persists a new order.
    // Returns ErrInsufficientStock if any item cannot be fulfilled.
    // The order starts in PENDING status until payment confirms.
    Create(ctx context.Context, cmd CreateOrderCommand) (*Order, error)

    // Confirm transitions an order from PENDING to CONFIRMED.
    // Triggers inventory reservation and fulfillment notification.
    // Returns ErrInvalidStatus if order is not PENDING.
    Confirm(ctx context.Context, orderID OrderID, paymentRef string) error
}

// Step 2: Implementation follows the documented contract
```

## Red Flags

Watch for these signs of complexity:

| Red Flag | Indication |
|----------|------------|
| Shallow module | Interface nearly as complex as implementation |
| Information leakage | Implementation details visible in interface |
| Temporal decomposition | Operations split by time rather than information |
| Pass-through method | Method does nothing except call another method |
| Repetition | Same code appears in multiple places |
| Special-general mixture | General-purpose code intertwined with special cases |
| Conjoined methods | Can't understand one method without reading another |
| Hard to describe | If explaining what code does is difficult, redesign |

## Applying to Go

### Go's Design Aligns Well

- **Small interfaces**: `io.Reader`, `io.Writer` are deep
- **Zero values**: Reduce errors out of existence
- **Error returns**: Explicit, but use wisely
- **Package privacy**: Lowercase = information hiding

### Interface Design Checklist

- [ ] Is the interface smaller than the implementation? (Deep)
- [ ] Does it hide implementation decisions? (Information hiding)
- [ ] Are most errors handled internally? (Pull complexity down)
- [ ] Does it have sensible defaults? (Define errors away)
- [ ] Does it serve multiple use cases? (Somewhat general)

## Related Skills

- `/clean-architecture` - Layer design where these principles apply
- `/cohesion-coupling` - Module design metrics
- `/go-best-practices` - Go-specific implementation
- `/modular-monolith` - Module boundary design
