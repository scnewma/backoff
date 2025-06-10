// Package backoff provides configurable exponential backoff functionality using Go iterators.
//
// This package offers a flexible and composable approach to implementing retry logic
// with support for exponential and constant backoff strategies, jitter, context cancellation,
// and early termination through cancel errors.
//
// The package leverages Go 1.23+ range-over-func iterators for a clean and intuitive API.
//
// Basic usage:
//
//	// Simple exponential backoff with defaults
//	for delay := range backoff.Iter() {
//	    time.Sleep(delay)
//	    if tryOperation() {
//	        break
//	    }
//	}
//
//	// Retry with automatic backoff
//	result, err := backoff.Retry(func() (string, error) {
//	    return callAPI()
//	}, backoff.MaxRetries(5))
//
//	// Custom configuration
//	for delay := range backoff.Iter(
//	    backoff.InitialDelay(100*time.Millisecond),
//	    backoff.MaxDelay(30*time.Second),
//	    backoff.Multiplier(2.0),
//	    backoff.JitterFactor(0.1),
//	    backoff.MaxRetries(10),
//	) {
//	    time.Sleep(delay)
//	    if tryOperation() {
//	        break
//	    }
//	}
package backoff

import (
	"context"
	"iter"
	"math"
	"math/rand/v2"
	"time"
)

// CancelError wraps an error to indicate that retries should be cancelled.
// Use Cancel() to create a cancel error that will stop retries immediately.
type CancelError struct {
	Err error
}

// Error returns the error message of the wrapped error.
// This implements the error interface.
func (e CancelError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error.
// This allows CancelError to work with Go's error unwrapping functions like errors.Is and errors.As.
func (e CancelError) Unwrap() error {
	return e.Err
}

// Cancel wraps an error to indicate that retries should be cancelled.
// When a function returns a cancel error, retries will stop immediately.
func Cancel(err error) error {
	return CancelError{Err: err}
}

// Option is a function that configures backoff behavior.
// Options are applied to modify backoff parameters like delays, retry limits, and jitter.
type Option func(*config)

type config struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitterFactor float64
	maxRetries   int
}

// InitialDelay sets the initial delay duration for the first retry attempt.
// If d is <= 0, it defaults to 1 millisecond.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.InitialDelay(100*time.Millisecond)) {
//	    // First delay will be 100ms
//	}
func InitialDelay(d time.Duration) Option {
	return func(c *config) {
		if d <= 0 {
			d = 1 * time.Millisecond
		}
		c.initialDelay = d
	}
}

// MaxDelay sets the maximum delay duration that backoff delays will not exceed.
// If d is <= 0, it defaults to 30 seconds.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.MaxDelay(5*time.Second)) {
//	    // Delays will never exceed 5 seconds
//	}
func MaxDelay(d time.Duration) Option {
	return func(c *config) {
		if d <= 0 {
			d = 30 * time.Second
		}
		c.maxDelay = d
	}
}

// Multiplier sets the factor by which delays are multiplied for each retry attempt.
// If m is <= 1.0, it defaults to 2.0 for exponential backoff.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.Multiplier(1.5)) {
//	    // Each delay will be 1.5x the previous delay
//	}
func Multiplier(m float64) Option {
	return func(c *config) {
		if m <= 1.0 {
			m = 2.0
		}
		c.multiplier = m
	}
}

// JitterFactor sets the amount of randomness to add to delays to avoid thundering herd problems.
// The factor represents a percentage (0.1 = 10% jitter). If factor < 0, it defaults to 0.
// Jitter is applied as a random value between -factor*delay and +factor*delay.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.JitterFactor(0.1)) {
//	    // Each delay will have ±10% random variation
//	}
func JitterFactor(factor float64) Option {
	return func(c *config) {
		if factor < 0 {
			factor = 0
		}
		c.jitterFactor = factor
	}
}

// MaxRetries sets the maximum number of retry attempts.
// If retries < 0, it defaults to 0 (no retries).
// The default when no MaxRetries is specified is math.MaxInt (effectively infinite).
//
// Example:
//
//	for delay := range backoff.Iter(backoff.MaxRetries(5)) {
//	    // Will perform at most 5 retry attempts
//	}
func MaxRetries(retries int) Option {
	return func(c *config) {
		if retries < 0 {
			retries = 0
		}
		c.maxRetries = retries
	}
}

// Constant returns an Option that configures a constant backoff strategy.
// All retry delays will be the same duration (default 1 second) with no jitter.
// Use with other options to customize the constant delay duration.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.Constant(), backoff.MaxRetries(3)) {
//	    // Will retry 3 times with 1 second delays
//	}
//
//	for delay := range backoff.Iter(backoff.Constant(), backoff.InitialDelay(500*time.Millisecond)) {
//	    // Will retry with constant 500ms delays
//	}
func Constant() Option {
	return func(c *config) {
		c.initialDelay = 1 * time.Second
		c.maxDelay = 1 * time.Second
		c.multiplier = 1.0
		c.jitterFactor = 0.0
	}
}

// Exponential returns an Option that configures an exponential backoff strategy.
// Uses sensible defaults: 100ms initial delay, 30s max delay, 2.0 multiplier, 10% jitter.
// This provides good default behavior for most retry scenarios.
//
// Example:
//
//	for delay := range backoff.Iter(backoff.Exponential(), backoff.MaxRetries(5)) {
//	    // Will retry 5 times with exponential backoff: ~100ms, ~200ms, ~400ms, ~800ms, ~1.6s
//	}
//
//	for delay := range backoff.Iter(backoff.Exponential(), backoff.JitterFactor(0)) {
//	    // Exponential backoff without jitter for predictable timing
//	}
func Exponential() Option {
	return func(c *config) {
		c.initialDelay = 100 * time.Millisecond
		c.maxDelay = 30 * time.Second
		c.multiplier = 2.0
		c.jitterFactor = 0.1
	}
}

// Iter returns an iterator that yields backoff delay durations.
// If no options are provided, it defaults to exponential backoff with sensible defaults.
// The iterator will yield delay durations that should be waited before each retry attempt.
//
// The iterator supports Go's range-over-func feature (Go 1.23+):
//
// Example:
//
//	// Basic usage with defaults (exponential backoff)
//	for delay := range backoff.Iter() {
//	    time.Sleep(delay)
//	    // perform retry operation
//	    if success {
//	        break
//	    }
//	}
//
//	// Custom configuration
//	for delay := range backoff.Iter(
//	    backoff.InitialDelay(50*time.Millisecond),
//	    backoff.MaxDelay(5*time.Second),
//	    backoff.Multiplier(1.5),
//	    backoff.MaxRetries(3),
//	) {
//	    time.Sleep(delay)
//	    // perform retry operation
//	}
//
//	// Constant backoff
//	for delay := range backoff.Iter(backoff.Constant(), backoff.MaxRetries(5)) {
//	    time.Sleep(delay)
//	    // perform retry operation
//	}
func Iter(options ...Option) iter.Seq[time.Duration] {
	cfg := &config{
		maxRetries: math.MaxInt,
	}
	Exponential()(cfg)

	// Apply user options to override defaults
	for _, opt := range options {
		opt(cfg)
	}

	return func(yield func(time.Duration) bool) {
		delay := cfg.initialDelay
		if cfg.maxDelay < cfg.initialDelay {
			cfg.maxDelay = cfg.initialDelay
		}

		attempt := 0
		for attempt < cfg.maxRetries {

			currentDelay := delay

			if cfg.jitterFactor > 0 {
				jitterRange := float64(delay) * cfg.jitterFactor
				jitter := (rand.Float64() - 0.5) * 2 * jitterRange
				currentDelay = time.Duration(float64(delay) + jitter)
			}

			if currentDelay > cfg.maxDelay {
				currentDelay = cfg.maxDelay
			}

			if !yield(currentDelay) {
				return
			}

			nextDelay := time.Duration(float64(delay) * cfg.multiplier)
			delay = min(cfg.maxDelay, nextDelay)
			attempt++
		}
	}
}

// Retry executes a function with automatic retry logic using exponential backoff.
// It will retry the function until it succeeds, returns a CancelError, or exhausts the retry limit.
// This is a convenience wrapper around RetryWithContext using context.Background().
//
// The function fn should return a value and an error. If the error is nil, the operation
// is considered successful. If the error is a CancelError (created with Cancel()), retries
// will stop immediately.
//
// Example:
//
//	result, err := backoff.Retry(func() (string, error) {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return "", err
//	    }
//	    defer resp.Body.Close()
//
//	    if resp.StatusCode >= 500 {
//	        return "", fmt.Errorf("server error: %d", resp.StatusCode)
//	    }
//	    if resp.StatusCode == 401 {
//	        return "", backoff.Cancel(fmt.Errorf("unauthorized"))
//	    }
//
//	    body, err := io.ReadAll(resp.Body)
//	    return string(body), err
//	}, backoff.MaxRetries(3))
func Retry[T any](fn func() (T, error), options ...Option) (T, error) {
	return RetryWithContext(context.Background(), fn, options...)
}

// RetryWithContext executes a function with automatic retry logic and context cancellation.
// It will retry the function until it succeeds, returns a CancelError, the context is cancelled,
// or the retry limit is exhausted.
//
// The function fn should return a value and an error. If the error is nil, the operation
// is considered successful. If the error is a CancelError (created with Cancel()), retries
// will stop immediately. If the context is cancelled, the function returns immediately with
// the context error.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	result, err := backoff.RetryWithContext(ctx, func() (*http.Response, error) {
//	    return http.Get("https://api.example.com/data")
//	}, backoff.MaxRetries(5), backoff.MaxDelay(2*time.Second))
//
//	if err == context.DeadlineExceeded {
//	    // Operation timed out after 30 seconds
//	}
func RetryWithContext[T any](ctx context.Context, fn func() (T, error), options ...Option) (T, error) {
	var lastErr error
	var result T

	result, lastErr = fn()
	if lastErr == nil {
		return result, nil
	}

	// Check if the initial error is a cancel error
	if _, ok := lastErr.(CancelError); ok {
		return result, lastErr
	}

	for delay := range Iter(options...) {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
			result, lastErr = fn()
			if lastErr == nil {
				return result, nil
			}
			// Check if the error is a cancel error and stop retrying
			if _, ok := lastErr.(CancelError); ok {
				return result, lastErr
			}
		}
	}

	return result, lastErr
}
