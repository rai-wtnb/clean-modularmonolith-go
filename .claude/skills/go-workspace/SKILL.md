---
name: go-workspace
description: Go multi-module development with go.work. Use when setting up monorepos, developing multiple interdependent modules, managing local development workflow, or configuring IDE support for multi-module projects. Covers go.work setup, module dependencies, and workspace best practices.
---

# Go Workspaces for Multi-Module Development

Go workspaces (`go.work`) enable developing multiple modules together without modifying individual `go.mod` files. Perfect for modular monolith architecture.

## When to Use Workspaces

- **Monorepo with multiple modules** - Each module has its own `go.mod`
- **Developing library alongside consumers** - Test changes immediately
- **Local development** - No need for `replace` directives in `go.mod`
- **Cross-module refactoring** - IDE understands all modules together

## Basic Setup

### Creating a Workspace

```bash
# Navigate to project root
cd /path/to/project

# Initialize workspace
go work init

# Add modules to workspace
go work use ./cmd/server
go work use ./modules/users
go work use ./modules/orders
go work use ./modules/shared

# Or add multiple at once
go work use ./cmd/server ./modules/users ./modules/orders ./modules/shared
```

### Resulting go.work File

```
go 1.22

use (
    ./cmd/server
    ./modules/orders
    ./modules/shared
    ./modules/users
)
```

## Recommended Directory Structure

### Modular Monolith Layout

```
project/
├── go.work                    # Workspace file (gitignored!)
├── go.work.sum                # Workspace checksums (gitignored!)
├── .gitignore                 # Contains go.work*
├── cmd/
│   └── server/
│       ├── go.mod             # module myapp/cmd/server
│       ├── go.sum
│       └── main.go
├── modules/
│   ├── users/
│   │   ├── go.mod             # module myapp/modules/users
│   │   ├── go.sum
│   │   ├── module.go
│   │   ├── domain/
│   │   ├── application/
│   │   └── infrastructure/
│   ├── orders/
│   │   ├── go.mod             # module myapp/modules/orders
│   │   ├── go.sum
│   │   └── ...
│   └── shared/
│       ├── go.mod             # module myapp/modules/shared
│       ├── go.sum
│       ├── events/
│       └── types/
└── pkg/
    └── httputil/
        ├── go.mod             # module myapp/pkg/httputil
        └── response.go
```

### Individual go.mod Files

```go
// cmd/server/go.mod
module myapp/cmd/server

go 1.22

require (
    myapp/modules/users v0.0.0
    myapp/modules/orders v0.0.0
    myapp/modules/shared v0.0.0
)
```

```go
// modules/users/go.mod
module myapp/modules/users

go 1.22

require (
    myapp/modules/shared v0.0.0
)
```

```go
// modules/orders/go.mod
module myapp/modules/orders

go 1.22

require (
    myapp/modules/users v0.0.0
    myapp/modules/shared v0.0.0
)
```

```go
// modules/shared/go.mod
module myapp/modules/shared

go 1.22

// No internal dependencies - this is the base
```

**Key Point**: When using `go.work`, the workspace resolves local paths automatically. No `replace` directives needed during development!

## Working with Workspaces

### Adding a New Module

```bash
# 1. Create directory structure
mkdir -p modules/payments

# 2. Initialize the module
cd modules/payments
go mod init myapp/modules/payments

# 3. Return to root and add to workspace
cd ../..
go work use ./modules/payments

# 4. Verify workspace
cat go.work
```

### Running Commands

```bash
# Commands work across all workspace modules

# Run application
go run ./cmd/server

# Build all modules
go build ./...

# Test all modules
go test ./...

# Test specific module
go test ./modules/users/...

# Get dependencies for all modules
go mod tidy  # Run in each module directory
```

### Keeping Dependencies in Sync

```bash
# Sync workspace with module go.mod files
go work sync

# Run after:
# - Adding new external dependencies
# - Updating dependency versions
# - Changing inter-module dependencies
```

## IDE Configuration

### VS Code

VS Code with gopls handles workspaces automatically when `go.work` exists.

```json
// .vscode/settings.json (usually not needed)
{
    "go.useLanguageServer": true,
    "gopls": {
        "ui.semanticTokens": true
    }
}
```

**If IDE doesn't recognize cross-module imports:**

1. Open Command Palette (Cmd+Shift+P)
2. Run "Go: Restart Language Server"
3. Wait for gopls to reindex

### GoLand / IntelliJ

GoLand detects `go.work` automatically. If not working:

1. Right-click `go.work` in Project view
2. Select "Go" → "Sync Go Workspace"

## Best Practices

### 1. Don't Commit go.work

```gitignore
# .gitignore
go.work
go.work.sum
```

**Rationale**:
- `go.work` is for local development only
- CI/CD should use published versions or explicit replace directives
- Each developer might have different local paths

### 2. CI/CD Setup

**Option A: Create workspace in CI**

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Create workspace
        run: |
          go work init
          go work use ./cmd/server ./modules/users ./modules/orders ./modules/shared

      - name: Test
        run: go test ./...

      - name: Build
        run: go build ./cmd/server
```

**Option B: Use Makefile**

```makefile
# Makefile

.PHONY: workspace
workspace:
	@go work init 2>/dev/null || true
	@go work use ./cmd/server ./modules/users ./modules/orders ./modules/shared

.PHONY: test
test: workspace
	go test ./...

.PHONY: build
build: workspace
	go build -o bin/server ./cmd/server

.PHONY: tidy
tidy:
	@for dir in cmd/server modules/users modules/orders modules/shared; do \
		echo "Tidying $$dir..."; \
		cd $$dir && go mod tidy && cd -; \
	done
```

### 3. Testing Without Workspace

```bash
# Temporarily disable workspace to test as if modules were published
GOWORK=off go test ./modules/users/...

# This helps catch:
# - Missing dependencies in go.mod
# - Incorrect import paths
# - Modules that won't work standalone
```

## Common Issues and Solutions

### "module not found" Error

**Cause**: Module not added to workspace

**Fix**:
```bash
go work use ./modules/missing
```

### IDE Not Recognizing Imports

**Fixes**:

1. **Restart language server**
   ```
   VS Code: Cmd+Shift+P → "Go: Restart Language Server"
   ```

2. **Regenerate workspace**
   ```bash
   rm go.work go.work.sum
   go work init
   go work use ./cmd/server ./modules/users ./modules/orders ./modules/shared
   ```

## Workspace Commands Reference

```bash
# Initialize empty workspace
go work init

# Initialize with modules
go work init ./cmd/server ./modules/users

# Add module to workspace
go work use ./path/to/module

# Add multiple modules
go work use ./mod1 ./mod2 ./mod3

# Remove module (Go 1.22+)
go work edit -dropuse ./old/module

# Sync dependencies across modules
go work sync

# Run with workspace disabled
GOWORK=off go build ./...
```

## Related Skills

- `/modular-monolith` - Module structure for the project
- `/go-best-practices` - Go conventions
- `/clean-architecture` - Layer structure within modules
