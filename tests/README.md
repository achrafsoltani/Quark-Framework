# Integration Tests

This directory contains integration and end-to-end tests for the Quark framework.

## Test Organization

The Quark project follows a hybrid testing approach:

### Unit Tests
Located alongside source files (e.g., `router_test.go` next to `router.go`)
- Test individual components in isolation
- Can test unexported functions and types
- Use `package quark`
- Run with: `go test ./...` (includes all tests)

### Integration Tests
Located in this `tests/` directory
- Test multiple components working together
- Test real HTTP requests/responses
- Test middleware chains
- Test with actual dependencies (databases, etc.)
- Use `package tests` or `package quark_test`
- Run with: `go test ./tests/...`

## Running Tests

```bash
# Run all tests (unit + integration)
go test ./...

# Run only unit tests
go test -short ./...

# Run only integration tests
go test ./tests/...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...
```

## Adding New Tests

- **Unit tests**: Add `*_test.go` files next to the source code
- **Integration tests**: Add `*_test.go` files in this directory
- Use `testing.Short()` to skip slow tests: `if testing.Short() { t.Skip() }`

## Test Naming Conventions

- Unit tests: `TestFunctionName`
- Integration tests: `TestIntegration_FeatureName`
- Benchmarks: `BenchmarkFunctionName`
