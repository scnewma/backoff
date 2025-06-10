package backoff

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	var delays []time.Duration
	count := 0
	for delay := range Iter() {
		delays = append(delays, delay)
		count++
		if count >= 3 {
			break
		}
	}
	
	if len(delays) != 3 {
		t.Errorf("Expected 3 delays, got %d", len(delays))
	}
	
	// Check that delays are increasing (roughly)
	if delays[1] <= delays[0] {
		t.Errorf("Expected second delay to be greater than first")
	}
}

func TestOptions(t *testing.T) {
	var delays []time.Duration
	for delay := range Iter(
		InitialDelay(50*time.Millisecond),
		MaxDelay(1*time.Second),
		Multiplier(1.5),
		JitterFactor(0), // No jitter
		MaxRetries(3),
	) {
		delays = append(delays, delay)
	}
	
	expected := []time.Duration{
		50 * time.Millisecond,
		75 * time.Millisecond,                                       // 50 * 1.5
		time.Duration(float64(75*time.Millisecond) * 1.5), // 75 * 1.5 = 112.5ms
	}
	
	if len(delays) != len(expected) {
		t.Fatalf("Expected %d delays, got %d", len(expected), len(delays))
	}
	
	for i, expectedDelay := range expected {
		if delays[i] != expectedDelay {
			t.Errorf("Delay %d: expected %v, got %v", i, expectedDelay, delays[i])
		}
	}
}

func TestIteratorWithoutJitter(t *testing.T) {
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
	}
	
	var actual []time.Duration
	for delay := range Iter(
		InitialDelay(100*time.Millisecond),
		MaxDelay(1*time.Second),
		Multiplier(2.0),
		JitterFactor(0), // No jitter
		MaxRetries(4),
	) {
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
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond, // capped at max delay
		300 * time.Millisecond, // stays at max delay
	}
	
	var actual []time.Duration
	for delay := range Iter(
		InitialDelay(100*time.Millisecond),
		MaxDelay(300*time.Millisecond),
		Multiplier(2.0),
		JitterFactor(0), // No jitter
		MaxRetries(4),
	) {
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
	var delays []time.Duration
	for delay := range Iter(
		InitialDelay(100*time.Millisecond),
		MaxDelay(1*time.Second),
		Multiplier(2.0),
		JitterFactor(0.1), // 10% jitter
		MaxRetries(3),
	) {
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
	expected := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		200 * time.Millisecond, // stays at max delay
		200 * time.Millisecond,
	}
	
	var actual []time.Duration
	count := 0
	for delay := range Iter(
		InitialDelay(50*time.Millisecond),
		MaxDelay(200*time.Millisecond),
		Multiplier(2.0),
		JitterFactor(0), // No jitter
		// No MaxRetries specified, so it defaults to math.MaxInt (effectively infinite)
	) {
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
	
	result, err := Retry(func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}, InitialDelay(1*time.Millisecond), MaxRetries(3), JitterFactor(0))
	
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
	
	result, err := Retry(func() (int, error) {
		attempts++
		return 0, errors.New("persistent failure")
	}, InitialDelay(1*time.Millisecond), MaxRetries(2), JitterFactor(0))
	
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
	
	result, err := Retry(func() (bool, error) {
		attempts++
		return true, nil
	}, MaxRetries(3))
	
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

func TestRetryWithContext_Cancellation(t *testing.T) {
	attempts := 0
	
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	
	result, err := RetryWithContext(ctx, func() (string, error) {
		attempts++
		return "", errors.New("always fails")
	}, InitialDelay(10*time.Millisecond), MaxRetries(5))
	
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
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	result, err := RetryWithContext(ctx, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary failure")
		}
		return 42, nil
	}, InitialDelay(1*time.Millisecond), MaxRetries(3))
	
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

