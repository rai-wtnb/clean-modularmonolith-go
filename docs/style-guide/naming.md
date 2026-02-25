# Naming Conventions

This document describes naming conventions for consistency across the codebase.

## General Principles

1. **Clarity over brevity** - Names should be self-documenting
2. **Context-aware** - Shorter names are acceptable in limited scope
3. **Consistent** - Follow established patterns in the codebase

## Types

### Typed IDs

Format: `{Aggregate}ID`

```go
// GOOD
type UserID string
type OrderID string

// BAD
type UserId string   // Inconsistent casing
type User_ID string  // Underscores
type ID string       // Too generic
```

### Value Objects

Format: `{Concept}` (noun, no suffix)

```go
// GOOD
type Email string
type Money struct { amount int64; currency string }
type Address struct { ... }

// BAD
type EmailValue string    // Unnecessary suffix
type EmailVO string       // Abbreviation
```

### Aggregates/Entities

Format: `{Noun}`

```go
// GOOD
type User struct { ... }
type Order struct { ... }

// BAD
type UserEntity struct { ... }    // Unnecessary suffix
type UserAggregate struct { ... } // Unnecessary suffix
```

### Domain Events

Format: `{Aggregate}{PastTenseVerb}Event`

```go
// GOOD
type UserCreatedEvent struct { ... }
type OrderSubmittedEvent struct { ... }
type OrderCancelledEvent struct { ... }

// BAD
type UserCreateEvent struct { ... }    // Not past tense
type CreateUserEvent struct { ... }    // Verb first
type UserCreated struct { ... }        // Missing Event suffix
```

### Event Type Constants

Format: `{Aggregate}{PastTenseVerb}EventType`

Value format: `{module}.{Aggregate}{PastTenseVerb}`

```go
// GOOD
const UserCreatedEventType events.EventType = "users.UserCreated"
const OrderSubmittedEventType events.EventType = "orders.OrderSubmitted"

// BAD
const UserCreated = "UserCreated"              // Missing module prefix
const USER_CREATED_EVENT = "users.UserCreated" // SCREAMING_CASE
```

### Repository Interfaces

Format: `{Aggregate}Repository`

```go
// GOOD
type UserRepository interface { ... }
type OrderRepository interface { ... }

// BAD
type UserRepo interface { ... }       // Abbreviation
type IUserRepository interface { ... } // Hungarian notation
type Users interface { ... }          // Not clear it's a repository
```

### Command/Query Handlers

Format: `{Verb}{Noun}Handler`

```go
// GOOD
type CreateUserHandler struct { ... }
type GetOrderHandler struct { ... }
type ListUserOrdersHandler struct { ... }

// BAD
type UserCreator struct { ... }       // Inconsistent pattern
type CreateUserCommand struct { ... } // Command, not handler
```

### Event Handlers

Format: `{EventName}Handler` (without "Event" suffix)

```go
// GOOD
type UserDeletedHandler struct { ... }
type OrderSubmittedHandler struct { ... }

// BAD
type UserDeletedEventHandler struct { ... }  // Redundant "Event"
type HandleUserDeleted struct { ... }        // Verb first
```

## Functions

### Factory Functions

Format: `New{Type}`

```go
// GOOD
func NewUser(email, firstName, lastName string) (*User, error)
func NewEmail(value string) (Email, error)
func NewMoney(amount int64, currency string) Money

// BAD
func CreateUser(...) (*User, error)  // Create implies persistence
func MakeUser(...) (*User, error)    // Inconsistent with Go conventions
```

### Reconstitution Functions

Format: `Reconstitute{Type}` or `Reconstitute`

Used for hydrating aggregates from persistence without validation.

```go
// GOOD
func Reconstitute(id UserID, email Email, name Name, createdAt time.Time) *User

// BAD
func FromDB(...) *User           // Implementation-specific
func Hydrate(...) *User          // Less clear intent
func NewUserFromRow(...) *User   // Mixing concerns
```

### Validation Methods

Format: `Validate() error` or constructor validation

```go
// GOOD: Validation in constructor
func NewEmail(value string) (Email, error) {
    if !isValidEmail(value) {
        return "", ErrInvalidEmail
    }
    return Email(value), nil
}

// GOOD: Explicit validation method
func (t EventType) Validate() error { ... }
```

### Business Methods

Format: `{Verb}` or `{Verb}{Object}`

```go
// GOOD
func (o *Order) Submit() error
func (o *Order) AddItem(item OrderItem) error
func (o *Order) Cancel() error

// BAD
func (o *Order) DoSubmit() error    // Unnecessary "Do"
func (o *Order) SubmitOrder() error // Redundant "Order"
```

## Files

### Domain Files

| File | Contents |
|------|----------|
| `{aggregate}.go` | Aggregate root entity |
| `{value_object}.go` | Value object (if complex) |
| `repository.go` | Repository interface |
| `events.go` | Domain event definitions |
| `errors.go` | Domain errors |

### Application Files

| File | Contents |
|------|----------|
| `{verb}_{noun}.go` | Single command/query handler |

Example: `create_user.go`, `get_order.go`, `list_user_orders.go`

### Infrastructure Files

| File | Contents |
|------|----------|
| `spanner_repository.go` | Spanner implementation of repository |
| `handler.go` | HTTP handlers |

## Packages

### Module Package Names

Use lowercase, single-word names matching the bounded context.

```go
// GOOD
package users
package orders
package notifications

// BAD
package user           // Singular (less idiomatic for Go)
package userManagement // CamelCase
package user_module    // Underscores
```

### Layer Package Names

```go
// GOOD
package domain
package commands
package queries
package eventhandlers
package http
package persistence

// BAD
package cmd        // Abbreviation (conflicts with cmd/)
package handlers   // Ambiguous (HTTP? Event?)
```

## Quick Reference

| Concept | Pattern | Example |
|---------|---------|---------|
| Typed ID | `{Aggregate}ID` | `UserID`, `OrderID` |
| Value Object | `{Noun}` | `Email`, `Money` |
| Aggregate | `{Noun}` | `User`, `Order` |
| Domain Event | `{Aggregate}{Verb}Event` | `UserCreatedEvent` |
| Event Type | `{module}.{Aggregate}{Verb}` | `users.UserCreated` |
| Repository | `{Aggregate}Repository` | `UserRepository` |
| Handler | `{Verb}{Noun}Handler` | `CreateUserHandler` |
| Factory | `New{Type}` | `NewUser()` |
| Reconstitution | `Reconstitute` | `Reconstitute()` |
