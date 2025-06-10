package backoff_test

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/scnewma/backoff"
)

func ExampleConfig_Iterator() {
	config := backoff.NewConfig().
		WithInitialDelay(50 * time.Millisecond).
		WithMaxDelay(1 * time.Second).
		WithMultiplier(2.0).
		WithJitter(false).
		WithMaxRetries(3)

	fmt.Println("Backoff delays:")
	for delay := range config.Iterator() {
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
	config := backoff.NewConfig().
		WithInitialDelay(10 * time.Millisecond).
		WithMaxRetries(3).
		WithJitter(false)

	result, err := backoff.Retry(config, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	})

	fmt.Printf("Result: %s, Error: %v, Attempts: %d\n", result, err, attempts)
	// Output:
	// Result: success, Error: <nil>, Attempts: 3
}

func ExampleConfig_InfiniteIterator() {
	config := backoff.NewConfig().
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(500 * time.Millisecond).
		WithJitter(false)

	count := 0
	for delay := range config.InfiniteIterator() {
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
	config := backoff.NewConfig().
		WithInitialDelay(25 * time.Millisecond).
		WithMaxDelay(200 * time.Millisecond).
		WithMultiplier(1.5)

	fmt.Println("Custom backoff with jitter:")
	count := 0
	for delay := range config.Iterator() {
		fmt.Printf("Attempt %d: ~%v\n", count+1, delay.Round(time.Millisecond))
		count++
		if count >= 4 {
			break
		}
	}
}

func Example_networkRetry() {
	config := backoff.NewConfig().
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(5 * time.Second).
		WithMaxRetries(5)

	_, err := backoff.Retry(config, func() ([]byte, error) {
		log.Println("Attempting network request...")
		return nil, errors.New("network timeout")
	})

	if err != nil {
		fmt.Printf("All retries failed: %v\n", err)
	}
}