# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library that provides configurable exponential backoff functionality using Go 1.23+ iterators. The library offers a functional options pattern for composable retry logic with support for exponential and constant backoff strategies, jitter, context cancellation, and early termination.

## Development Commands

### Testing
```bash
# Run all tests
go test -v

# Run a specific test
go test -v -run TestDefaults

# Run tests with coverage
go test -cover

# Run example tests (includes output verification)
go test -v -run Example
```

### Code Quality
```bash
# Check for issues
go vet ./...

# Format code
go fmt ./...

# Generate documentation
go doc -all
```

### Building
```bash
# Build (library packages don't produce binaries)
go build

# Install locally for testing with other projects
go install
```

## Architecture

### Core Design Patterns

**Functional Options Pattern**: The library uses functional options (`Option func(*config)`) instead of builder pattern for configuration. All options can be combined and override defaults applied by `Exponential()`.

**Iterator-Based API**: Leverages Go 1.23+ `iter.Seq[time.Duration]` for clean iteration over backoff delays. The `Iter()` function returns an iterator that yields delay durations.

**Default Behavior**: When no options are provided, `Iter()` defaults to exponential backoff with sensible defaults (100ms initial, 30s max, 2.0 multiplier, 10% jitter, infinite retries).

### Key Components

**Configuration System**:
- `config` struct holds all backoff parameters
- `Option` functions modify configuration
- `Exponential()` and `Constant()` provide preset configurations
- Individual options like `InitialDelay()`, `MaxRetries()` allow fine-tuning

**Error Handling**:
- `CancelError` type for early termination of retry loops
- `Cancel(err)` function wraps errors to signal immediate stop
- Implements `Unwrap()` for compatibility with Go error handling

**Retry Functions**:
- `Retry[T]()` - simple retry wrapper using context.Background()
- `RetryWithContext[T]()` - context-aware retry with cancellation support
- Generic functions work with any return type

### Testing Strategy

**Unit Tests** (`backoff_test.go`):
- Tests for default behavior, custom options, jitter, edge cases
- Context cancellation scenarios
- Cancel error behavior
- Separate tests for constant vs exponential backoff

**Example Tests** (`example_test.go`):
- Demonstrates API usage patterns
- Includes output verification for deterministic examples
- Shows real-world usage scenarios (HTTP requests, network retries)

## Important Implementation Details

**Jitter Implementation**: Uses `math/rand/v2` for jitter calculation. Exponential backoff includes 10% jitter by default, constant backoff has no jitter.

**Context Handling**: Context cancellation is only supported in `RetryWithContext()`, not in the raw iterator functions. Iterator functions remain context-free for flexibility.

**Option Application Order**: Options are applied after `Exponential()` defaults, allowing users to override any default parameter.

**Validation**: All option functions include validation (e.g., negative values default to sensible minimums).

## Go Version Requirements

This library requires Go 1.24.3+ due to its use of range-over-func iterators introduced in Go 1.23. The `iter.Seq[time.Duration]` type is central to the API design.