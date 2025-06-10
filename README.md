# backoff

A Go library for configurable exponential backoff using Go 1.23+ iterators.

[![Go Reference](https://pkg.go.dev/badge/github.com/scnewma/backoff.svg)](https://pkg.go.dev/github.com/scnewma/backoff)
[![Go Version](https://img.shields.io/badge/go-1.24.3+-blue.svg)](https://golang.org/)

## Features

- **Iterator-based API** using Go 1.23+ range-over-func iterators
- **Functional options pattern** for flexible configuration
- **Exponential and constant backoff strategies** with sensible defaults
- **Configurable jitter** to prevent thundering herd problems
- **Context cancellation** support for timeout and cancellation
- **Early termination** with cancel errors for permanent failures
- **Zero dependencies** beyond the Go standard library

## Installation

```bash
go get github.com/scnewma/backoff
```

**Requirements**: Go 1.24.3+ (due to iterator support)

## Quick Start

### Basic Exponential Backoff

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/scnewma/backoff"
)

func main() {
    // Simple exponential backoff with defaults
    // (100ms initial, 30s max, 2.0 multiplier, 10% jitter)
    for delay := range backoff.Iter() {
        fmt.Printf("Waiting %v before retry\n", delay)
        time.Sleep(delay)
        
        if tryOperation() {
            fmt.Println("Success!")
            break
        }
    }
}
```

### Automatic Retry with Backoff

```go
// Retry a function automatically with exponential backoff
result, err := backoff.Retry(func() (string, error) {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 500 {
        return "", fmt.Errorf("server error: %d", resp.StatusCode)
    }
    
    // Permanent error - stop retrying immediately
    if resp.StatusCode == 401 {
        return "", backoff.Cancel(fmt.Errorf("unauthorized"))
    }
    
    body, err := io.ReadAll(resp.Body)
    return string(body), err
}, backoff.MaxRetries(5))

if err != nil {
    log.Printf("Failed after retries: %v", err)
}
```

## Configuration Options

### Exponential Backoff (Default)

```go
for delay := range backoff.Iter(
    backoff.InitialDelay(50*time.Millisecond),
    backoff.MaxDelay(10*time.Second),
    backoff.Multiplier(1.5),
    backoff.JitterFactor(0.1), // 10% jitter
    backoff.MaxRetries(5),
) {
    // Your retry logic here
}
```

### Constant Backoff

```go
for delay := range backoff.Iter(
    backoff.Constant(),
    backoff.InitialDelay(1*time.Second),
    backoff.MaxRetries(3),
) {
    // Will retry every 1 second, 3 times
}
```

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := backoff.RetryWithContext(ctx, func() (*http.Response, error) {
    return http.Get("https://api.example.com/data")
}, backoff.MaxRetries(10))

if err == context.DeadlineExceeded {
    log.Println("Operation timed out")
}
```

## API Reference

### Configuration Functions

- `InitialDelay(duration)` - Set the first retry delay
- `MaxDelay(duration)` - Cap the maximum delay
- `Multiplier(factor)` - Set delay multiplication factor
- `JitterFactor(factor)` - Add randomness (0.1 = 10% jitter)
- `MaxRetries(count)` - Limit retry attempts

### Strategy Presets

- `Exponential()` - Exponential backoff with 10% jitter (default)
- `Constant()` - Fixed delay intervals with no jitter

### Core Functions

- `Iter(options...)` - Returns an iterator over delay durations
- `Retry(fn, options...)` - Retry function with backoff
- `RetryWithContext(ctx, fn, options...)` - Context-aware retry
- `Cancel(err)` - Wrap error to stop retries immediately

## Examples

### Database Connection Retry

```go
db, err := backoff.Retry(func() (*sql.DB, error) {
    return sql.Open("postgres", connectionString)
}, 
    backoff.InitialDelay(100*time.Millisecond),
    backoff.MaxDelay(5*time.Second),
    backoff.MaxRetries(10),
)
```

### File Upload with Progress

```go
count := 0
for delay := range backoff.Iter(backoff.MaxRetries(3)) {
    count++
    fmt.Printf("Upload attempt %d...\n", count)
    
    time.Sleep(delay)
    if uploadFile() {
        fmt.Println("Upload successful!")
        break
    }
}
```

### Service Health Check

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

healthy, err := backoff.RetryWithContext(ctx, func() (bool, error) {
    resp, err := http.Get("http://service/health")
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    
    return resp.StatusCode == 200, nil
}, backoff.Constant(), backoff.InitialDelay(5*time.Second))
```

## Why Use This Library?

- **Modern Go**: Leverages Go 1.23+ iterators for clean, idiomatic code
- **Composable**: Mix and match options for exact behavior you need
- **Practical**: Includes jitter, context support, and early termination
- **Tested**: Comprehensive test suite with 100% coverage
- **Simple**: Zero dependencies, focused API

## Contributing

This library was designed to be simple and focused. For bugs or feature requests, please open an issue on GitHub.

## License

MIT License - see LICENSE file for details.

---

> **Note**: This library was generated with assistance from [Claude](https://claude.ai), an AI assistant by Anthropic, demonstrating AI-assisted software development for modern Go libraries.