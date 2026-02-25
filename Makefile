.PHONY: workspace build run test test-coverage lint clean tidy deps-check deps-update sync vulncheck help

# Module paths
MODULES := cmd/server modules/shared modules/users modules/orders modules/notifications internal/platform

# Default target
.DEFAULT_GOAL := help

## workspace: Initialize go.work with all modules
workspace:
	@go work init 2>/dev/null || true
	@for mod in $(MODULES); do \
		go work use ./$$mod 2>/dev/null || true; \
	done
	@echo "Workspace initialized with modules: $(MODULES)"

## build: Build the server binary
build: workspace
	go build -o bin/server ./cmd/server

## run: Run the server
run: build
	./bin/server

## test: Run all tests
test: workspace
	@for mod in $(MODULES); do \
		echo "Testing $$mod..."; \
		go test -race ./$$mod/...; \
	done

## test-coverage: Run tests with coverage report
test-coverage: workspace
	@for mod in $(MODULES); do \
		go test -race -coverprofile=$$mod/coverage.out ./$$mod/...; \
	done
	@echo "Coverage reports generated in each module directory"

## lint: Run golangci-lint
lint:
	@for mod in $(MODULES); do \
		echo "Linting $$mod..."; \
		golangci-lint run ./$$mod/...; \
	done

## tidy: Run go mod tidy on all modules
tidy:
	@for mod in $(MODULES); do \
		echo "Tidying $$mod..."; \
		(cd $$mod && go mod tidy); \
	done

## deps-check: Check for available dependency updates
deps-check:
	@for mod in $(MODULES); do \
		echo "=== $$mod ==="; \
		(cd $$mod && go list -m -u all 2>/dev/null | grep '\[' || echo "No updates available"); \
	done

## deps-update: Update all dependencies across all modules
deps-update:
	@for mod in $(MODULES); do \
		echo "Updating $$mod..."; \
		(cd $$mod && go get -u ./... && go mod tidy); \
	done
	@$(MAKE) sync

## sync: Synchronize go.work with all modules
sync:
	go work sync

## vulncheck: Check for security vulnerabilities
vulncheck:
	@command -v govulncheck >/dev/null 2>&1 || { echo "Installing govulncheck..."; go install golang.org/x/vuln/cmd/govulncheck@latest; }
	govulncheck ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
