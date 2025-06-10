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

func (e CancelError) Error() string {
	return e.Err.Error()
}

func (e CancelError) Unwrap() error {
	return e.Err
}

// Cancel wraps an error to indicate that retries should be cancelled.
// When a function returns a cancel error, retries will stop immediately.
func Cancel(err error) error {
	return CancelError{Err: err}
}

type Option func(*config)

type config struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitterFactor float64
	maxRetries   int
}

func InitialDelay(d time.Duration) Option {
	return func(c *config) {
		if d <= 0 {
			d = 1 * time.Millisecond
		}
		c.initialDelay = d
	}
}

func MaxDelay(d time.Duration) Option {
	return func(c *config) {
		if d <= 0 {
			d = 30 * time.Second
		}
		c.maxDelay = d
	}
}

func Multiplier(m float64) Option {
	return func(c *config) {
		if m <= 1.0 {
			m = 2.0
		}
		c.multiplier = m
	}
}

func JitterFactor(factor float64) Option {
	return func(c *config) {
		if factor < 0 {
			factor = 0
		}
		c.jitterFactor = factor
	}
}

func MaxRetries(retries int) Option {
	return func(c *config) {
		if retries < 0 {
			retries = 0
		}
		c.maxRetries = retries
	}
}

// Constant returns an Option that configures a constant backoff strategy.
// All retry delays will be the same duration (default 1 second).
func Constant() Option {
	return func(c *config) {
		c.initialDelay = 1 * time.Second
		c.maxDelay = 1 * time.Second
		c.multiplier = 1.0
		c.jitterFactor = 0.0
	}
}

// Exponential returns an Option that configures an exponential backoff strategy.
// Uses sensible defaults: 100ms initial delay, 30s max delay, 2.0 multiplier, no jitter.
func Exponential() Option {
	return func(c *config) {
		c.initialDelay = 100 * time.Millisecond
		c.maxDelay = 30 * time.Second
		c.multiplier = 2.0
		c.jitterFactor = 0.0
	}
}

func Iter(options ...Option) iter.Seq[time.Duration] {
	cfg := &config{
		maxRetries: math.MaxInt,
	}

	if len(options) == 0 {
		Exponential()(cfg)
	} else {
		for _, opt := range options {
			opt(cfg)
		}
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

func Retry[T any](fn func() (T, error), options ...Option) (T, error) {
	return RetryWithContext(context.Background(), fn, options...)
}

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
