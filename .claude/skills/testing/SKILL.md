# Testing Guidelines

## Philosophy

Less is more, but as much as needed. Don't split tests artificially, but don't combine unrelated behaviors either. Write the minimum tests that give you confidence.

## Two Test Types

### Unit Tests

**When to use**: Component supports dependency injection and you can substitute dependencies meaningfully.

**The mock trap**: If you're mocking everything, you're testing nothing. A test with all mocks only verifies mock wiring, not real behavior. This is the "mockist" anti-pattern—avoid it.

**Prefer fakes over mocks**: Use in-memory implementations (InMemoryRepository, FakeFileSystem) that behave like real dependencies. Stubs that return canned data are acceptable. Full mocks that verify call sequences are a last resort.

**Mock ceiling**: If you need more than 2 mocks, stop and write an integration test instead.

### Integration Tests

**When to use**: Default choice. Always prefer unless unit test is clearly better.

**Real everything**: Use real databases (SQLite in-memory, testcontainers), real APIs (test/sandbox environments), real filesystem operations. Only mock when explicitly requested.

**Touch the outer edges**: Integration tests should exercise the farthest boundaries of the application. Start as close to main() or the entry point as practical.

**CLI applications**: Call the compiled binary with real arguments and real (but controlled) test data. Verify exit codes, stdout, stderr, and side effects on filesystem/database.

## Decision Logic

1. Pure function with no dependencies → Unit test, no mocks needed
2. Class with DI and 1-2 simple dependencies → Unit test with fakes/stubs
3. Anything touching database, API, or filesystem → Integration test with real resources
4. CLI command → Integration test calling actual binary
5. Would require >2 mocks → Integration test
6. Unsure → Integration test

## Test Structure

One test can and should verify multiple related behaviors. Assert on status, data transformation, persistence, and side effects in a single test when they're part of the same operation. Don't split into test_status, test_data, test_persistence—that's artificial separation.

## Acceptable Mock Scenarios

- Third-party APIs you don't control (payment gateways, external SaaS)
- Time-dependent behavior (clock/date functions)
- Non-deterministic operations (random, UUIDs)
- Error conditions hard to reproduce (network timeouts, disk full)

For these cases, prefer a thin wrapper you control that can be substituted, rather than deep mocking of library internals.
