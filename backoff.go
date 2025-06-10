package backoff

import (
	"iter"
	"math/rand/v2"
	"time"
)

type Config struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
	MaxRetries   int
}

func NewConfig() *Config {
	return &Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   10,
	}
}

func (c *Config) WithInitialDelay(d time.Duration) *Config {
	if d <= 0 {
		d = 1 * time.Millisecond
	}
	c.InitialDelay = d
	return c
}

func (c *Config) WithMaxDelay(d time.Duration) *Config {
	if d <= 0 {
		d = 30 * time.Second
	}
	c.MaxDelay = d
	return c
}

func (c *Config) WithMultiplier(m float64) *Config {
	if m <= 1.0 {
		m = 2.0
	}
	c.Multiplier = m
	return c
}

func (c *Config) WithJitter(enabled bool) *Config {
	c.Jitter = enabled
	return c
}

func (c *Config) WithMaxRetries(retries int) *Config {
	if retries < 0 {
		retries = 0
	}
	c.MaxRetries = retries
	return c
}

func (c *Config) Iterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		delay := c.InitialDelay
		if c.MaxDelay < c.InitialDelay {
			c.MaxDelay = c.InitialDelay
		}
		
		for attempt := 0; attempt < c.MaxRetries; attempt++ {
			currentDelay := delay
			
			if c.Jitter {
				jitterRange := float64(delay) * 0.1
				jitter := (rand.Float64() - 0.5) * 2 * jitterRange
				currentDelay = time.Duration(float64(delay) + jitter)
			}
			
			if currentDelay > c.MaxDelay {
				currentDelay = c.MaxDelay
			}
			
			if !yield(currentDelay) {
				return
			}
			
			nextDelay := time.Duration(float64(delay) * c.Multiplier)
			if nextDelay > c.MaxDelay {
				delay = c.MaxDelay
			} else {
				delay = nextDelay
			}
		}
	}
}

func (c *Config) InfiniteIterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		delay := c.InitialDelay
		if c.MaxDelay < c.InitialDelay {
			c.MaxDelay = c.InitialDelay
		}
		
		for {
			currentDelay := delay
			
			if c.Jitter {
				jitterRange := float64(delay) * 0.1
				jitter := (rand.Float64() - 0.5) * 2 * jitterRange
				currentDelay = time.Duration(float64(delay) + jitter)
			}
			
			if currentDelay > c.MaxDelay {
				currentDelay = c.MaxDelay
			}
			
			if !yield(currentDelay) {
				return
			}
			
			nextDelay := time.Duration(float64(delay) * c.Multiplier)
			if nextDelay > c.MaxDelay {
				delay = c.MaxDelay
			} else {
				delay = nextDelay
			}
		}
	}
}

func Retry[T any](config *Config, fn func() (T, error)) (T, error) {
	var lastErr error
	var result T
	
	result, lastErr = fn()
	if lastErr == nil {
		return result, nil
	}
	
	for delay := range config.Iterator() {
		time.Sleep(delay)
		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}
	}
	
	return result, lastErr
}