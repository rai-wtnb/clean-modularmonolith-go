# Domain Service Pattern

## Overview

A Domain Service encapsulates domain logic that doesn't naturally belong to a single Entity or Value Object. It operates on domain objects without knowing about persistence.

## Key Characteristics

- **Stateless**: No internal state, pure functions
- **Domain-focused**: Contains business rules, not infrastructure concerns
- **Persistence-agnostic**: Works with in-memory objects only

## Implementation

### Domain Service (Domain Layer)

```go
// domain/services/order_pricing_service.go
type OrderPricingService struct{}

// Pure domain logic - no persistence awareness
func (s *OrderPricingService) ApplyDiscount(order *Order, coupon *Coupon) error {
    if !coupon.IsValidFor(order) {
        return ErrInvalidCoupon
    }
    order.ApplyDiscount(coupon.DiscountRate())
    return nil
}
```

### Command Handler (Application Layer)

```go
// application/commands/apply_discount_handler.go
type ApplyDiscountHandler struct {
    orderRepo      domain.OrderRepository
    pricingService *domain.OrderPricingService
}

func (h *ApplyDiscountHandler) Handle(ctx context.Context, cmd ApplyDiscountCommand) error {
    // 1. Load aggregate from repository
    order, err := h.orderRepo.FindByID(ctx, cmd.OrderID)
    if err != nil {
        return err
    }

    // 2. Domain Service applies business logic (in-memory only)
    coupon := domain.NewCoupon(cmd.CouponCode)
    if err := h.pricingService.ApplyDiscount(order, coupon); err != nil {
        return err
    }

    // 3. Repository saves with transaction encapsulated inside
    return h.orderRepo.Save(ctx, order)
}
```

## Flow

```
┌─────────────────────────────────────────────────────────┐
│                   Command Handler                       │
│                                                         │
│   ┌─────────────┐                                       │
│   │ FindByID()  │  ← Read (no transaction needed)       │
│   └──────┬──────┘                                       │
│          ▼                                              │
│   ┌─────────────────────┐                               │
│   │  Domain Service     │  ← Pure logic (no persistence)│
│   │  (in-memory ops)    │                               │
│   └──────┬──────────────┘                               │
│          ▼                                              │
│   ┌─────────────┐                                       │
│   │   Save()    │  ← Write (transaction inside repo)    │
│   └─────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

## Responsibilities

| Layer | Responsibility | Transaction |
|-------|----------------|-------------|
| Domain Service | Business logic (what to do) | Not involved |
| Command Handler | Orchestration | Not involved |
| Repository | Persistence (how to save) | Encapsulated inside |

## Why This Works

1. **Single Aggregate**: Domain Service operates on one aggregate at a time
2. **In-Memory Operations**: All logic happens on loaded objects
3. **Repository Handles Persistence**: Transaction is repository's concern

## When Domain Service Needs Multiple Aggregates

If a Domain Service seems to need multiple aggregates for consistency:

| Scenario | Solution |
|----------|----------|
| "Update A and B together" | They should be the same aggregate |
| "Update A, then B" | Use domain events (eventual consistency) |
| "Validate A against B" | Pass B's data as a read-only parameter |

### Example: Validation Against Another Aggregate

```go
// Don't load User aggregate - just pass the needed data
func (s *OrderService) ValidateOrder(order *Order, userStatus UserStatus) error {
    if userStatus != UserStatusActive {
        return ErrInactiveUser
    }
    return order.Validate()
}

// Command handler
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
    order, _ := h.orderRepo.FindByID(ctx, cmd.OrderID)
    user, _ := h.userRepo.FindByID(ctx, cmd.UserID)  // Read-only

    if err := h.orderService.ValidateOrder(order, user.Status()); err != nil {
        return err
    }

    return h.orderRepo.Save(ctx, order)  // Only order is saved
}
```

## Comparison with Unit of Work

| Aspect | Domain Service + Repository TX | Unit of Work |
|--------|-------------------------------|--------------|
| Transaction scope | Single aggregate | Multiple aggregates |
| Domain purity | High (no infra in domain) | Lower (UoW abstraction leaks) |
| Complexity | Simple | Moderate |
| DDD alignment | Strong | Weaker (violates aggregate boundary) |

## References

- [Domain-Driven Design](https://www.domainlanguage.com/ddd/) - Eric Evans
- [Implementing Domain-Driven Design](https://www.oreilly.com/library/view/implementing-domain-driven-design/9780133039900/) - Vaughn Vernon
