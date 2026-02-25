.PHONY: workspace build run test test-coverage lint clean tidy deps-check deps-update sync vulncheck deps-graph deps-svg help

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

## deps-graph: Show cross-module dependency graph
deps-graph:
	@echo "=== Module Dependencies ==="
	@echo ""
	@echo "Legend: ✅ clean | ❌ forbidden import | ✓ allowed (shared)"
	@echo ""
	@for module in orders users notifications; do \
		echo "📦 modules/$$module:"; \
		forbidden=$$(go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' ./modules/$$module/... 2>/dev/null \
			| grep "github.com/rai/clean-modularmonolith-go/modules" \
			| grep -v "modules/$$module" \
			| grep -v "modules/shared" \
			| sort -u); \
		if [ -z "$$forbidden" ]; then \
			echo "   ✅ No cross-module imports"; \
		else \
			echo "$$forbidden" | sed 's/^/   ❌ /'; \
		fi; \
		go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' ./modules/$$module/... 2>/dev/null \
			| grep "modules/shared" \
			| sort -u \
			| sed 's/^/   ✓ /'; \
		echo ""; \
	done

## deps-svg: Generate dependency graph SVG in docs/
deps-svg:
	@command -v goda >/dev/null 2>&1 || { echo "Installing goda..."; go install github.com/loov/goda@latest; }
	@command -v dot >/dev/null 2>&1 || { echo "Error: graphviz not installed. Run: brew install graphviz"; exit 1; }
	goda graph "github.com/rai/clean-modularmonolith-go/..." | dot -Tsvg -o docs/deps.svg
	@echo "Generated docs/deps.svg"

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
