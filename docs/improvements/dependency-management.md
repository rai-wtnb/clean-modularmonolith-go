# Dependency Management Improvements

## Current Limitations with `depguard`

Currently, we use `depguard` in `.golangci.yml` to prevent cross-module dependencies (e.g., preventing the `orders` module from importing the `users` domain or infrastructure).

However, `depguard` v2 has a limitation: it **does not support dynamic rules** such as regular expressions for module names or variables like `$CURRENT_MODULE`. Because of this, we cannot define a generic rule stating "any module inside `modules/*` cannot import another module inside `modules/*`".

As a result, every time a new module is added to the project, its `depguard` rules must be **manually added** to `.golangci.yml`. If a developer forgets to update the configuration, cross-module dependency violations might go unnoticed.

## Proposed Approaches to Prevent Oversight

To ensure that new modules are automatically checked for dependency rules and to prevent developers from forgetting to update the linter configurations, we propose the following three approaches:

### Approach 1: Architecture Testing in Go (Recommended)

Instead of relying solely on static linter configurations, we can introduce architecture tests written in Go. By using libraries like [`matthewmcnew/archtest`](https://github.com/matthewmcnew/archtest) or writing custom tests using `golang.org/x/tools/go/packages`, we can dynamically scan all directories under `modules/` during the test phase.

- **Pros**: Any new directory added under `modules/` is automatically picked up by the test. No manual configuration updates are needed when a new module is created.
- **Cons**: Requires maintaining custom test code, and dependency violations are caught during `go test` rather than a standard linter run (though both can run in CI).

### Approach 2: Linter Configuration Validation Test

Keep the current `depguard` configuration, but add a simple Go test (or CI script) to enforce that the configuration stays up to date.
The test would:

1. Scan the `modules/` directory to get a list of all current modules.
2. Read and parse `.golangci.yml`.
3. Assert that every existing module has corresponding `depguard` rules defined.

- **Pros**: Keeps the dependency logic within the standard linter. If a developer adds a module but forgets to update `.golangci.yml`, this test will fail and prompt them to do so.
- **Cons**: `.golangci.yml` still grows continuously and remains tedious to maintain.

### Approach 3: Adopt an Architecture-Specific Linter

Replace or complement `depguard` with a tool specifically designed for architectural boundaries, such as [`go-arch-lint`](https://github.com/fe3dback/go-arch-lint).

- **Pros**: These tools understand the concept of "components" dynamically and allow for generic rules like "components in `modules/*` cannot depend on other components in `modules/*`".
- **Cons**: Introduces a new tool to the stack, requiring team members to learn its configuration syntax and install it locally/in CI.

## Next Steps

We should evaluate these approaches and decide on the best fit for our modular monolith architecture to automate dependency boundary enforcement.
