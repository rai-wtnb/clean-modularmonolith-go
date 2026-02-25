# Dependency Rule

This document defines the layer dependency rules within each module, following Clean Architecture principles.

## Layer Structure

Each module has three layers with strict dependency direction:

```
┌─────────────────────────────────────────────────────────┐
│                    infrastructure/                       │
│         (HTTP handlers, repositories, external)          │
├─────────────────────────────────────────────────────────┤
│                      application/                        │
│              (commands, queries, event handlers)         │
├─────────────────────────────────────────────────────────┤
│                        domain/                           │
│        (entities, value objects, repository interfaces)  │
└─────────────────────────────────────────────────────────┘
```

## The Dependency Rule

**Dependencies point inward only.**

```
infrastructure → application → domain
       ↓              ↓           ↓
   outer layer    middle      inner layer
```

- `infrastructure` may import `application` and `domain`
- `application` may import `domain` only
- `domain` imports nothing from the module (except shared kernel)

## Rules by Layer

### Domain Layer (`domain/`)

The domain layer has **zero dependencies** on other layers or external packages.

```go
// domain/user.go
package domain

// GOOD: No imports from application or infrastructure
type User struct {
    id    UserID
    email Email
    name  Name
}

// GOOD: Repository interface defined in domain
type UserRepository interface {
    Save(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id UserID) (*User, error)
}
```

**Allowed imports:**
- Standard library (`context`, `errors`, `time`, `fmt`)
- `modules/shared` (events, shared value objects)

**Prohibited imports:**
- `application/*`
- `infrastructure/*`
- External packages (database drivers, HTTP frameworks, etc.)

### Application Layer (`application/`)

The application layer depends only on the domain layer.

```go
// application/commands/create_user.go
package commands

import (
    "github.com/example/app/modules/users/domain"  // GOOD: domain import
)

type CreateUserHandler struct {
    repo domain.UserRepository  // Uses domain interface
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (string, error) {
    user, err := domain.NewUser(cmd.Email, cmd.FirstName, cmd.LastName)
    if err != nil {
        return "", err
    }
    return user.ID().String(), h.repo.Save(ctx, user)
}
```

**Allowed imports:**
- `domain/*`
- `modules/shared/*`
- Standard library

**Prohibited imports:**
- `infrastructure/*`
- Database drivers, HTTP packages, etc.

### Infrastructure Layer (`infrastructure/`)

The infrastructure layer implements domain interfaces and depends on both layers.

```go
// infrastructure/persistence/spanner_repository.go
package persistence

import (
    "cloud.google.com/go/spanner"  // GOOD: External package OK here
    "github.com/example/app/modules/users/domain"
)

type SpannerUserRepository struct {
    client *spanner.Client
}

// Implements domain.UserRepository
func (r *SpannerUserRepository) Save(ctx context.Context, user *domain.User) error {
    // Spanner-specific implementation
}
```

**Allowed imports:**
- `domain/*`
- `application/*`
- `modules/shared/*`
- External packages (database drivers, frameworks, etc.)

## Why This Matters

### 1. Testability

Domain and application layers can be tested without infrastructure:

```go
// Test with mock repository
func TestCreateUser(t *testing.T) {
    repo := &MockUserRepository{}
    handler := NewCreateUserHandler(repo)
    // No database needed
}
```

### 2. Flexibility

Infrastructure can be swapped without changing business logic:

```go
// Switch from Spanner to PostgreSQL
// Only infrastructure/ changes, domain/ and application/ untouched
```

### 3. Focus

Each layer has a single responsibility:
- **domain**: Business rules
- **application**: Use case orchestration
- **infrastructure**: Technical concerns

## Dependency Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                         module                                │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ infrastructure/                                       │    │
│  │  ├── http/           → handlers call application     │    │
│  │  └── persistence/    → implements domain interfaces  │    │
│  └──────────────────────────────────────────────────────┘    │
│                           │                                   │
│                           ▼                                   │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ application/                                          │    │
│  │  ├── commands/       → orchestrate domain objects    │    │
│  │  ├── queries/        → read operations               │    │
│  │  └── eventhandlers/  → react to domain events        │    │
│  └──────────────────────────────────────────────────────┘    │
│                           │                                   │
│                           ▼                                   │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ domain/                                               │    │
│  │  ├── entities        → business objects              │    │
│  │  ├── value objects   → immutable values              │    │
│  │  ├── events          → domain event definitions      │    │
│  │  └── repository.go   → repository interfaces         │    │
│  └──────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

## Common Violations

### Violation: Domain imports infrastructure

```go
// domain/user.go
import "cloud.google.com/go/spanner"  // BAD!

type User struct {
    key spanner.Key  // Domain tied to Spanner
}
```

**Fix:** Use primitive types or domain-defined types.

### Violation: Application imports HTTP packages

```go
// application/commands/create_user.go
import "net/http"  // BAD!

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    // Application layer shouldn't know about HTTP
}
```

**Fix:** HTTP handling belongs in `infrastructure/http/`.

### Violation: Domain calls external services

```go
// domain/user.go
func (u *User) SendWelcomeEmail() {
    smtp.Send(u.email, "Welcome!")  // BAD!
}
```

**Fix:** External calls belong in infrastructure, orchestrated by application.

## See Also

- [Module Boundaries](module-boundaries.md)
- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
