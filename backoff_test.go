package backoff

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	
	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected initial delay of 100ms, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("Expected max delay of 30s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Expected multiplier of 2.0, got %v", config.Multiplier)
	}
	if !config.Jitter {
		t.Errorf("Expected jitter to be enabled by default")
	}
	if config.MaxRetries != 10 {
		t.Errorf("Expected max retries of 10, got %v", config.MaxRetries)
	}
}

func TestConfigChaining(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(50 * time.Millisecond).
		WithMaxDelay(1 * time.Second).
		WithMultiplier(1.5).
		WithJitter(false).
		WithMaxRetries(5)
	
	if config.InitialDelay != 50*time.Millisecond {
		t.Errorf("Expected initial delay of 50ms, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 1*time.Second {
		t.Errorf("Expected max delay of 1s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 1.5 {
		t.Errorf("Expected multiplier of 1.5, got %v", config.Multiplier)
	}
	if config.Jitter {
		t.Errorf("Expected jitter to be disabled")
	}
	if config.MaxRetries != 5 {
		t.Errorf("Expected max retries of 5, got %v", config.MaxRetries)
	}
}

func TestIteratorWithoutJitter(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(1 * time.Second).
		WithMultiplier(2.0).
		WithJitter(false).
		WithMaxRetries(4)
	
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
	}
	
	var actual []time.Duration
	for delay := range config.Iterator() {
		actual = append(actual, delay)
	}
	
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d delays, got %d", len(expected), len(actual))
	}
	
	for i, expectedDelay := range expected {
		if actual[i] != expectedDelay {
			t.Errorf("Delay %d: expected %v, got %v", i, expectedDelay, actual[i])
		}
	}
}

func TestIteratorWithMaxDelay(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(300 * time.Millisecond).
		WithMultiplier(2.0).
		WithJitter(false).
		WithMaxRetries(4)
	
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond, // capped at max delay
		300 * time.Millisecond, // stays at max delay
	}
	
	var actual []time.Duration
	for delay := range config.Iterator() {
		actual = append(actual, delay)
	}
	
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d delays, got %d", len(expected), len(actual))
	}
	
	for i, expectedDelay := range expected {
		if actual[i] != expectedDelay {
			t.Errorf("Delay %d: expected %v, got %v", i, expectedDelay, actual[i])
		}
	}
}

func TestIteratorWithJitter(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(1 * time.Second).
		WithMultiplier(2.0).
		WithJitter(true).
		WithMaxRetries(3)
	
	var delays []time.Duration
	for delay := range config.Iterator() {
		delays = append(delays, delay)
	}
	
	if len(delays) != 3 {
		t.Fatalf("Expected 3 delays, got %d", len(delays))
	}
	
	// With jitter, delays should be roughly around expected values but not exact
	baseDelays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	
	for i, delay := range delays {
		base := baseDelays[i]
		minDelay := time.Duration(float64(base) * 0.9)
		maxDelay := time.Duration(float64(base) * 1.1)
		
		if delay < minDelay || delay > maxDelay {
			t.Errorf("Delay %d: %v is outside expected jitter range [%v, %v]", i, delay, minDelay, maxDelay)
		}
	}
}

func TestInfiniteIterator(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(50 * time.Millisecond).
		WithMaxDelay(200 * time.Millisecond).
		WithMultiplier(2.0).
		WithJitter(false)
	
	expected := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		200 * time.Millisecond, // stays at max delay
		200 * time.Millisecond,
	}
	
	var actual []time.Duration
	count := 0
	for delay := range config.InfiniteIterator() {
		actual = append(actual, delay)
		count++
		if count >= 5 {
			break
		}
	}
	
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d delays, got %d", len(expected), len(actual))
	}
	
	for i, expectedDelay := range expected {
		if actual[i] != expectedDelay {
			t.Errorf("Delay %d: expected %v, got %v", i, expectedDelay, actual[i])
		}
	}
}

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	config := NewConfig().
		WithInitialDelay(1 * time.Millisecond).
		WithMaxRetries(3).
		WithJitter(false)
	
	result, err := Retry(config, func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryFailure(t *testing.T) {
	attempts := 0
	config := NewConfig().
		WithInitialDelay(1 * time.Millisecond).
		WithMaxRetries(2).
		WithJitter(false)
	
	result, err := Retry(config, func() (int, error) {
		attempts++
		return 0, errors.New("persistent failure")
	})
	
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "persistent failure" {
		t.Errorf("Expected 'persistent failure', got %v", err)
	}
	if result != 0 {
		t.Errorf("Expected result 0, got %v", result)
	}
	if attempts != 3 { // initial attempt + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryImmediateSuccess(t *testing.T) {
	attempts := 0
	config := NewConfig().WithMaxRetries(3)
	
	result, err := Retry(config, func() (bool, error) {
		attempts++
		return true, nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !result {
		t.Errorf("Expected result true, got %v", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestIteratorWithContext_Cancellation(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(1 * time.Millisecond).
		WithMaxRetries(10).
		WithJitter(false)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	var delays []time.Duration
	go func() {
		time.Sleep(5 * time.Millisecond) // Cancel quickly
		cancel()
	}()
	
	for delay := range config.IteratorWithContext(ctx) {
		delays = append(delays, delay)
		time.Sleep(delay) // Simulate actual delay usage
	}
	
	// Should have stopped early due to cancellation
	if len(delays) >= 10 {
		t.Errorf("Expected fewer than 10 delays due to cancellation, got %d", len(delays))
	}
}

func TestInfiniteIteratorWithContext_Cancellation(t *testing.T) {
	config := NewConfig().
		WithInitialDelay(1 * time.Millisecond).
		WithJitter(false)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	var delays []time.Duration
	for delay := range config.InfiniteIteratorWithContext(ctx) {
		delays = append(delays, delay)
		time.Sleep(delay) // Simulate using the delay
		if len(delays) >= 100 { // Safety valve
			t.Fatal("Iterator should have been cancelled by context")
		}
	}
	
	// Should have stopped due to timeout
	if len(delays) == 0 {
		t.Errorf("Expected at least 1 delay before timeout, got %d", len(delays))
	}
}

func TestRetryWithContext_Cancellation(t *testing.T) {
	attempts := 0
	config := NewConfig().
		WithInitialDelay(10 * time.Millisecond).
		WithMaxRetries(5)
	
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	
	result, err := RetryWithContext(ctx, config, func() (string, error) {
		attempts++
		return "", errors.New("always fails")
	})
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty result, got %v", result)
	}
	// Should have made some attempts but not all 6 (initial + 5 retries)
	if attempts == 0 {
		t.Errorf("Expected at least 1 attempt, got %d", attempts)
	}
	if attempts > 6 {
		t.Errorf("Expected at most 6 attempts, got %d", attempts)
	}
}

func TestRetryWithContext_Success(t *testing.T) {
	attempts := 0
	config := NewConfig().
		WithInitialDelay(1 * time.Millisecond).
		WithMaxRetries(3)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	result, err := RetryWithContext(ctx, config, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary failure")
		}
		return 42, nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got %v", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestIteratorWithContext_ImmediateCancellation(t *testing.T) {
	config := NewConfig().WithMaxRetries(3)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	var delays []time.Duration
	for delay := range config.IteratorWithContext(ctx) {
		delays = append(delays, delay)
	}
	
	if len(delays) != 0 {
		t.Errorf("Expected 0 delays with immediately cancelled context, got %d", len(delays))
	}
}