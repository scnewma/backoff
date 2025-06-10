package backoff_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/scnewma/backoff"
)

func ExampleIter() {
	fmt.Println("Backoff delays:")
	for delay := range backoff.Iter(
		backoff.InitialDelay(50*time.Millisecond),
		backoff.MaxDelay(1*time.Second),
		backoff.Multiplier(2.0),
		backoff.JitterFactor(0), // No jitter
		backoff.MaxRetries(3),
	) {
		fmt.Printf("Waiting %v before retry\n", delay)
	}
	// Output:
	// Backoff delays:
	// Waiting 50ms before retry
	// Waiting 100ms before retry
	// Waiting 200ms before retry
}

func ExampleRetry() {
	attempts := 0

	result, err := backoff.Retry(func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}, backoff.InitialDelay(10*time.Millisecond), backoff.MaxRetries(3), backoff.JitterFactor(0))

	fmt.Printf("Result: %s, Error: %v, Attempts: %d\n", result, err, attempts)
	// Output:
	// Result: success, Error: <nil>, Attempts: 3
}

func ExampleIter_infinite() {
	count := 0
	for delay := range backoff.Iter(
		backoff.InitialDelay(100*time.Millisecond),
		backoff.MaxDelay(500*time.Millisecond),
		backoff.JitterFactor(0), // No jitter
		// No MaxRetries specified, so it defaults to math.MaxInt (effectively infinite)
	) {
		fmt.Printf("Delay %d: %v\n", count+1, delay)
		count++
		if count >= 5 {
			break
		}
	}
	// Output:
	// Delay 1: 100ms
	// Delay 2: 200ms
	// Delay 3: 400ms
	// Delay 4: 500ms
	// Delay 5: 500ms
}

func Example_customUsage() {
	fmt.Println("Custom backoff with 15% jitter:")
	count := 0
	for delay := range backoff.Iter(
		backoff.InitialDelay(25*time.Millisecond),
		backoff.MaxDelay(200*time.Millisecond),
		backoff.Multiplier(1.5),
		backoff.JitterFactor(0.15), // 15% jitter
		backoff.MaxRetries(4),
	) {
		fmt.Printf("Attempt %d: ~%v\n", count+1, delay.Round(time.Millisecond))
		count++
		if count >= 4 {
			break
		}
	}
}

func Example_networkRetry() {
	_, err := backoff.Retry(func() ([]byte, error) {
		log.Println("Attempting network request...")
		return nil, errors.New("network timeout")
	}, backoff.InitialDelay(100*time.Millisecond), backoff.MaxDelay(5*time.Second), backoff.MaxRetries(5))

	if err != nil {
		fmt.Printf("All retries failed: %v\n", err)
	}
}

func ExampleRetryWithContext() {
	attempts := 0

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	result, err := backoff.RetryWithContext(ctx, func() (string, error) {
		attempts++
		if attempts < 5 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}, backoff.InitialDelay(5*time.Millisecond), backoff.MaxRetries(10))

	if err == context.DeadlineExceeded {
		fmt.Printf("Operation timed out after %d attempts\n", attempts)
	} else {
		fmt.Printf("Result: %s, Attempts: %d\n", result, attempts)
	}
}

func ExampleRetry_cancelError() {
	attempts := 0

	_, err := backoff.Retry(func() (string, error) {
		attempts++
		if attempts == 1 {
			return "", errors.New("network timeout") // recoverable error, will retry
		}
		if attempts == 2 {
			return "", backoff.Cancel(errors.New("invalid credentials")) // cancel, stops immediately
		}
		return "success", nil // this will never be reached
	}, backoff.InitialDelay(10*time.Millisecond), backoff.MaxRetries(5))

	fmt.Printf("Stopped after %d attempts due to cancel error: %v\n", attempts, err)
	// Output:
	// Stopped after 2 attempts due to cancel error: invalid credentials
}

func ExampleIter_constant() {
	fmt.Println("Constant backoff delays:")
	count := 0
	for delay := range backoff.Iter(
		backoff.Constant(),
		backoff.MaxRetries(3),
	) {
		fmt.Printf("Attempt %d: %v\n", count+1, delay)
		count++
	}
	// Output:
	// Constant backoff delays:
	// Attempt 1: 1s
	// Attempt 2: 1s
	// Attempt 3: 1s
}

func ExampleIter_exponential() {
	fmt.Println("Exponential backoff delays:")
	count := 0
	for delay := range backoff.Iter(
		backoff.Exponential(),
		backoff.MaxRetries(4),
	) {
		fmt.Printf("Attempt %d: %v\n", count+1, delay)
		count++
	}
	// Output:
	// Exponential backoff delays:
	// Attempt 1: 100ms
	// Attempt 2: 200ms
	// Attempt 3: 400ms
	// Attempt 4: 800ms
}

func ExampleRetry_constantBackoff() {
	attempts := 0

	result, err := backoff.Retry(func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}, backoff.Constant(), backoff.MaxRetries(5))

	fmt.Printf("Result: %s, Error: %v, Attempts: %d\n", result, err, attempts)
	// Output:
	// Result: success, Error: <nil>, Attempts: 3
}
