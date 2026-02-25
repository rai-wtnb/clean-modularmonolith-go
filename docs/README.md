# Documentation

This directory contains project documentation organized by purpose.

## Directory Structure

```
docs/
├── README.md              # This file
├── architecture/          # High-level architecture and design decisions
├── rules/                 # Hard constraints (violations = bugs)
├── style-guide/           # Conventions and preferences (violations = inconsistency)
└── patterns/              # Design pattern explanations and rationale
```

## Categories

### [architecture/](architecture/)

High-level architecture documentation describing system structure and design decisions.

| Document | Description |
|----------|-------------|
| [domain-events.md](architecture/domain-events.md) | Domain event flow between modules |

### [rules/](rules/)

**Hard constraints that must be followed.** Violations cause bugs, data corruption, or system failures.

| Document | Description |
|----------|-------------|
| [domain-event-handlers.md](rules/domain-event-handlers.md) | Constraints for event handlers in transactional contexts |
| [module-boundaries.md](rules/module-boundaries.md) | Inter-module communication restrictions |
| [dependency-rule.md](rules/dependency-rule.md) | Layer dependency direction enforcement |

### [style-guide/](style-guide/)

**Conventions and preferences.** Violations cause inconsistency but not bugs.

| Document | Description |
|----------|-------------|
| [naming.md](style-guide/naming.md) | Naming conventions for types, functions, and files |

### [patterns/](patterns/)

Design pattern explanations with rationale for when and why to use them.

| Document | Description |
|----------|-------------|
| [domain-service.md](patterns/domain-service.md) | When to use domain services |
| [lightweight-domain-events.md](patterns/lightweight-domain-events.md) | In-process domain event implementation |
| [outbox.md](patterns/outbox.md) | Outbox pattern for reliable event publishing |
| [unit-of-work.md](patterns/unit-of-work.md) | Unit of Work pattern for transaction management |

## When to Add Documentation

| Type | Add when... |
|------|-------------|
| **rules/** | A constraint exists where violations cause runtime bugs |
| **style-guide/** | A convention should be followed for consistency |
| **patterns/** | A design pattern is used and needs explanation |
| **architecture/** | A high-level design decision affects multiple modules |
