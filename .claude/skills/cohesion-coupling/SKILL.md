---
name: cohesion-coupling
description: Software design principles for high cohesion and loose coupling. Use when evaluating module boundaries, deciding what belongs together, reducing dependencies between components, or refactoring tightly coupled code. Covers types of cohesion, types of coupling, and practical refactoring strategies.
---

# Cohesion and Coupling in Software Design

Two fundamental metrics for evaluating module quality:

- **Cohesion**: How strongly related elements within a module are
- **Coupling**: How dependent modules are on each other

**Goal: High Cohesion + Low Coupling**

## Understanding Cohesion

Cohesion measures how well the elements within a module belong together.

### Types of Cohesion (Best to Worst)

#### 1. Functional Cohesion (Best)

Every element contributes to a **single, well-defined task**.

```go
// HIGH FUNCTIONAL COHESION: All methods relate to order processing
type OrderProcessor struct {
    repo   OrderRepository
    pricer Pricer
    events EventPublisher
}

func (p *OrderProcessor) Create(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
    // Validates, creates, and persists order
}

func (p *OrderProcessor) Confirm(ctx context.Context, orderID OrderID) error {
    // Confirms order and triggers fulfillment
}

func (p *OrderProcessor) Cancel(ctx context.Context, orderID OrderID) error {
    // Cancels order and reverses any reservations
}

// Everything relates to order processing - nothing else
```

#### 2. Sequential Cohesion

Output of one element is input to another (pipeline).

```go
// SEQUENTIAL COHESION: Each step feeds the next
type ImagePipeline struct{}

func (p *ImagePipeline) Process(img image.Image) (image.Image, error) {
    img = p.resize(img, 800, 600)     // Output feeds next
    img = p.applyFilters(img)          // Output feeds next
    img = p.compress(img, 80)          // Output feeds next
    img = p.addWatermark(img)          // Final output
    return img, nil
}
```

#### 3. Communicational Cohesion

Elements operate on the **same data**.

```go
// COMMUNICATIONAL COHESION: All operate on User data
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) Create(ctx context.Context, u *User) error    { ... }
func (r *UserRepository) Update(ctx context.Context, u *User) error    { ... }
func (r *UserRepository) Delete(ctx context.Context, id UserID) error  { ... }
func (r *UserRepository) FindByID(ctx context.Context, id UserID) (*User, error) { ... }
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) { ... }
```

#### 4. Procedural Cohesion

Elements are grouped because they follow a procedure.

```go
// PROCEDURAL COHESION: Grouped by procedure, not by concept
type CheckoutProcedure struct {
    cart     CartService
    payment  PaymentService
    shipping ShippingService
    email    EmailService
}

func (c *CheckoutProcedure) Execute(ctx context.Context, userID string) error {
    // These steps must happen in order but are conceptually different
    cart, _ := c.cart.Get(ctx, userID)
    payment, _ := c.payment.Charge(ctx, cart.Total())
    shipping, _ := c.shipping.Schedule(ctx, cart.Items())
    c.email.SendConfirmation(ctx, userID, payment, shipping)
    return nil
}

// Better: Each step could be its own cohesive module
```

#### 5. Logical Cohesion (Weak)

Elements related by **category** but do different things.

```go
// LOGICAL COHESION: Related by "utility" category, not by function
package utils

func ParseJSON(data []byte) (any, error)        { ... }
func SendEmail(to, subject, body string) error  { ... }
func GenerateUUID() string                      { ... }
func HashPassword(password string) string       { ... }
func FormatCurrency(cents int64) string         { ... }

// These don't belong together!
// Each should be in its own domain-specific package
```

#### 6. Coincidental Cohesion (Worst)

Elements have **no meaningful relationship**.

```go
// COINCIDENTAL COHESION: Random collection
package helpers

func ConnectToDatabase() *sql.DB              { ... }
func ValidateEmail(email string) bool         { ... }
func StartHTTPServer(addr string)             { ... }
func MarshalProtobuf(m proto.Message) []byte  { ... }
func CalculateTax(amount float64) float64     { ... }

// Nothing in common - this is a code smell
```

### Improving Cohesion

```go
// BEFORE: Low cohesion - mixed responsibilities
type UserManager struct {
    db    *sql.DB
    cache *redis.Client
    email *EmailClient
}

func (m *UserManager) GetUser(id string) (*User, error)    { ... }
func (m *UserManager) SendWelcomeEmail(u *User) error      { ... }
func (m *UserManager) ClearUserCache(id string) error      { ... }
func (m *UserManager) GenerateReport() ([]byte, error)     { ... }
func (m *UserManager) ExportToCSV() ([]byte, error)        { ... }
func (m *UserManager) BackupUserData() error               { ... }

// AFTER: High cohesion - single responsibility each
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) { ... }
func (r *UserRepository) Save(ctx context.Context, u *User) error                { ... }
func (r *UserRepository) Delete(ctx context.Context, id string) error            { ... }

type UserCache struct {
    cache *redis.Client
}

func (c *UserCache) Get(id string) (*User, error)   { ... }
func (c *UserCache) Set(u *User) error              { ... }
func (c *UserCache) Invalidate(id string) error     { ... }

type UserNotifier struct {
    email *EmailClient
}

func (n *UserNotifier) SendWelcome(u *User) error   { ... }
func (n *UserNotifier) SendPasswordReset(u *User, token string) error { ... }

type UserExporter struct {
    repo *UserRepository
}

func (e *UserExporter) ToCSV(ctx context.Context) ([]byte, error)    { ... }
func (e *UserExporter) ToJSON(ctx context.Context) ([]byte, error)   { ... }
```

## Understanding Coupling

Coupling measures how much modules depend on each other.

### Types of Coupling (Best to Worst)

#### 1. Message Coupling (Best)

Modules communicate via messages/events with **no shared state**.

```go
// MESSAGE COUPLING: No direct dependencies
type OrderCreatedEvent struct {
    OrderID   string    `json:"order_id"`
    UserID    string    `json:"user_id"`
    Total     int64     `json:"total"`
    CreatedAt time.Time `json:"created_at"`
}

// Publisher doesn't know about subscribers
func (s *OrderService) Create(ctx context.Context, cmd CreateOrderCommand) error {
    order := createOrder(cmd)
    s.repo.Save(ctx, order)
    s.events.Publish(OrderCreatedEvent{...})  // Fire and forget
    return nil
}

// Subscriber doesn't know about publisher
func (s *InventoryService) OnOrderCreated(evt OrderCreatedEvent) {
    s.reserveItems(evt.OrderID)
}

func (s *NotificationService) OnOrderCreated(evt OrderCreatedEvent) {
    s.sendConfirmation(evt.UserID)
}
```

#### 2. Data Coupling (Good)

Modules share **only necessary primitive data** via parameters.

```go
// DATA COUPLING: Only pass what's needed
func CalculateShipping(weight float64, distance int, expedited bool) Money {
    // Only receives specific data needed for calculation
}

// NOT stamp coupling:
// func CalculateShipping(order Order) Money
// (passes whole object when only 3 fields needed)
```

#### 3. Stamp Coupling

Modules share **data structures** but don't use all fields.

```go
// STAMP COUPLING: Receives User but only needs email and name
func SendWelcomeEmail(user User) error {
    return sendEmail(user.Email, fmt.Sprintf("Welcome %s!", user.Name))
    // User has 20 fields but we only use 2
}

// BETTER: Data coupling
func SendWelcomeEmail(email, name string) error {
    return sendEmail(email, fmt.Sprintf("Welcome %s!", name))
}

// OR: Define minimal interface
type EmailRecipient interface {
    Email() string
    Name() string
}

func SendWelcomeEmail(recipient EmailRecipient) error {
    return sendEmail(recipient.Email(), fmt.Sprintf("Welcome %s!", recipient.Name()))
}
```

#### 4. Control Coupling

One module **controls behavior** of another via flags.

```go
// CONTROL COUPLING: Flags control behavior
func ProcessUser(user User, sendEmail bool, createAuditLog bool, notifyAdmin bool) error {
    // Process user...

    if sendEmail {
        sendWelcomeEmail(user)
    }
    if createAuditLog {
        logUserCreation(user)
    }
    if notifyAdmin {
        notifyAdminOfNewUser(user)
    }
    return nil
}

// BETTER: Separate concerns
type UserProcessor struct {
    notifier   UserNotifier
    auditor    Auditor
    adminAlert AdminAlerter
}

func (p *UserProcessor) Process(ctx context.Context, user User) error {
    // Core processing only
    return p.repo.Save(ctx, user)
}

// Callers compose behavior:
func (s *SignupService) SignUp(ctx context.Context, cmd SignUpCommand) error {
    user := createUser(cmd)
    s.processor.Process(ctx, user)
    s.notifier.SendWelcome(user)
    s.auditor.LogCreation(user)
    return nil
}
```

#### 5. Common Coupling

Modules share **global variables**.

```go
// COMMON COUPLING: Shared global state
var (
    currentUser *User  // Global!
    dbPool      *sql.DB
    config      *Config
)

func ProcessOrder() error {
    // Uses global currentUser - any module can modify it
    order := &Order{UserID: currentUser.ID}
    return dbPool.Save(order)
}

// BETTER: Explicit dependencies
type OrderService struct {
    db *sql.DB
}

func (s *OrderService) Process(ctx context.Context, userID string, cmd CreateOrderCommand) error {
    // Dependencies explicit, no hidden state
}
```

#### 6. Content Coupling (Worst)

One module **directly accesses another's internals**.

```go
// CONTENT COUPLING: Reaching into internal fields
type OrderService struct {
    userService *UserService
}

func (s *OrderService) CreateOrder(cmd CreateOrderCommand) error {
    // Directly accessing internal fields!
    s.userService.cache["key"] = value
    s.userService.db.Exec("UPDATE users...")
    s.userService.mutex.Lock()

    // This breaks encapsulation completely
}

// CORRECT: Use public interface
func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
    user, err := s.userService.GetUser(ctx, cmd.UserID)
    if err != nil {
        return err
    }
    // Use user through defined interface only
}
```

## Reducing Coupling in Practice

### 1. Depend on Interfaces, Not Implementations

```go
// TIGHTLY COUPLED: Depends on concrete type
type OrderService struct {
    repo *PostgresOrderRepository  // Concrete!
}

func NewOrderService(db *sql.DB) *OrderService {
    return &OrderService{
        repo: NewPostgresOrderRepository(db),  // Creates its own dependency
    }
}

// LOOSELY COUPLED: Depends on interface
type OrderService struct {
    repo OrderRepository  // Interface
}

type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id OrderID) (*Order, error)
}

func NewOrderService(repo OrderRepository) *OrderService {
    return &OrderService{repo: repo}  // Injected
}
```

### 2. Apply Interface Segregation

```go
// COUPLED: Large interface forces unnecessary dependencies
type UserStore interface {
    Create(u User) error
    Update(u User) error
    Delete(id UserID) error
    FindByID(id UserID) (*User, error)
    FindByEmail(email string) (*User, error)
    FindAll() ([]User, error)
    Search(query string) ([]User, error)
    Export() ([]byte, error)
    Import(data []byte) error
    Backup() error
}

// Client that only reads must still depend on all methods

// DECOUPLED: Segregated interfaces
type UserReader interface {
    FindByID(ctx context.Context, id UserID) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type UserWriter interface {
    Create(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id UserID) error
}

type UserSearcher interface {
    Search(ctx context.Context, query string) ([]User, error)
    FindAll(ctx context.Context) ([]User, error)
}

// Clients depend only on what they need
type AuthService struct {
    users UserReader  // Only needs read
}

type AdminService struct {
    users UserWriter  // Only needs write
}
```

### 3. Use Events for Cross-Module Communication

```go
// COUPLED: Direct calls create dependency graph
type OrderModule struct {
    inventory *InventoryModule
    payment   *PaymentModule
    shipping  *ShippingModule
    email     *EmailModule
    analytics *AnalyticsModule
}

func (m *OrderModule) Complete(ctx context.Context, orderID OrderID) error {
    m.inventory.Reserve(orderID)     // Direct dependency
    m.payment.Capture(orderID)       // Direct dependency
    m.shipping.Schedule(orderID)     // Direct dependency
    m.email.SendConfirmation(orderID)  // Direct dependency
    m.analytics.Track(orderID)       // Direct dependency
    // Order module knows about everyone!
}

// DECOUPLED: Event-driven communication
type OrderModule struct {
    repo   OrderRepository
    events EventPublisher
}

func (m *OrderModule) Complete(ctx context.Context, orderID OrderID) error {
    order, _ := m.repo.FindByID(ctx, orderID)
    order.Complete()
    m.repo.Save(ctx, order)

    // Just publish event - don't know who listens
    m.events.Publish(OrderCompleted{OrderID: orderID})
    return nil
}

// Other modules subscribe independently
func (m *InventoryModule) init() {
    m.events.Subscribe("order.completed", m.onOrderCompleted)
}
```

### 4. Introduce Abstraction Layers

```go
// COUPLED: Handler directly uses database
type UserHandler struct {
    db *sql.DB
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    row := h.db.QueryRow("SELECT * FROM users WHERE id = ?", id)
    // Parse SQL result directly in handler
}

// DECOUPLED: Layers of abstraction
type UserHandler struct {
    service UserService  // Depends on service interface
}

type UserService interface {
    GetUser(ctx context.Context, id string) (*UserDTO, error)
}

type userService struct {
    repo UserRepository  // Depends on repository interface
}

type UserRepository interface {
    FindByID(ctx context.Context, id UserID) (*User, error)
}

// Each layer can change independently
```

## Metrics and Heuristics

### Signs of Low Cohesion

| Sign | Example |
|------|---------|
| Hard to name | "UserManager", "DataHandler", "Utils" |
| Many unrelated methods | Mix of CRUD, validation, notification |
| Changes for different reasons | Payment logic + email templates |
| Methods don't use same fields | Half use `db`, half use `cache` |
| Large interfaces | 20+ methods in one interface |

### Signs of High Coupling

| Sign | Example |
|------|---------|
| Ripple effects | Change in A breaks B, C, D |
| Circular dependencies | A → B → C → A |
| God objects | One class knows about everything |
| Many parameters | Functions with 10+ parameters |
| Excessive mocking | Tests need 20 mock objects |

### Measuring Coupling

```go
// Count incoming dependencies (Afferent Coupling - Ca)
// Count outgoing dependencies (Efferent Coupling - Ce)

// Instability = Ce / (Ca + Ce)
// 0 = Stable (many depend on it, depends on few)
// 1 = Unstable (few depend on it, depends on many)

// Guideline: Depend in direction of stability
// Unstable modules should depend on stable modules
```

## The Stability Principle

**Depend in the direction of stability.**

```
┌──────────────────────────────────────────────────┐
│                    STABLE                         │
│     (Don't change often, many depend on it)      │
│  ┌──────────────────────────────────────────┐    │
│  │  - Interfaces                            │    │
│  │  - Domain entities                       │    │
│  │  - Core abstractions                     │    │
│  └──────────────────────────────────────────┘    │
│                       ▲                          │
│                       │ depends on               │
│                       │                          │
│  ┌──────────────────────────────────────────┐    │
│  │  - Use cases                             │    │
│  │  - Application services                  │    │
│  └──────────────────────────────────────────┘    │
│                       ▲                          │
│                       │ depends on               │
│                       │                          │
│  ┌──────────────────────────────────────────┐    │
│  │  - HTTP handlers                         │    │
│  │  - Repository implementations            │    │
│  │  - External adapters                     │    │
│  └──────────────────────────────────────────┘    │
│                   UNSTABLE                        │
│    (Changes often, few depend on it)             │
└──────────────────────────────────────────────────┘
```

```go
// CORRECT: Unstable depends on stable
type PostgresUserRepo struct{}  // Unstable (implementation)

func (r *PostgresUserRepo) Save(ctx context.Context, user *User) error {
    // Depends on User entity (stable)
}

// WRONG: Stable depends on unstable
type User struct {
    repo *PostgresUserRepo  // Entity depends on implementation!
}
```

## Go-Specific Guidance

### Package Cohesion

```go
// GOOD: Package has clear, focused purpose
package order

type Order struct { ... }
type OrderID string
type OrderStatus string
type OrderItem struct { ... }
type OrderRepository interface { ... }

func NewOrder(items []OrderItem) (*Order, error) { ... }

// BAD: Package is unfocused
package models

type User struct { ... }
type Order struct { ... }
type Product struct { ... }
type Category struct { ... }
type Config struct { ... }
type APIResponse struct { ... }
```

### Interface Size = Coupling

Go idiom: **Small interfaces = Lower coupling**

```go
// Go standard library - minimal interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Compose when needed
type ReadWriter interface {
    Reader
    Writer
}

// Your code should follow this pattern
type Saver interface {
    Save(ctx context.Context, data []byte) error
}

// NOT: 20-method interfaces that force coupling
```

### Accept Interfaces, Return Structs

```go
// GOOD: Minimizes coupling
func ProcessData(r io.Reader) (*Result, error) {
    // Accepts interface (flexible)
    // Returns concrete type (provides full API)
}

// Usage: Any reader works
ProcessData(file)
ProcessData(bytes.NewReader(data))
ProcessData(response.Body)
```

## Refactoring Checklist

When evaluating module design:

- [ ] Can you name the module in 1-2 words without "Manager", "Handler", "Utils"?
- [ ] Do all methods operate on related data or serve related purpose?
- [ ] Does changing one method rarely require changing others?
- [ ] Can you test the module with minimal mocking?
- [ ] Is the interface small (< 5 methods)?
- [ ] Are dependencies injected, not created internally?
- [ ] Is there no circular dependency with other modules?
- [ ] Would extraction to a separate service be straightforward?

## Related Skills

- `/modular-monolith` - Module boundaries where these apply
- `/clean-architecture` - Layer coupling rules
- `/philosophy-software-design` - Deep modules concept
- `/go-best-practices` - Go interface patterns
