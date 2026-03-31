package retry

import (
	"log/slog"
	"time"
)

// Do executes the given function with exponential backoff retries.
func Do(maxRetries int, operation string, fn func() error) error {
	var err error
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		slog.Warn("Operation failed, retrying",
			"operation", operation,
			"attempt", i+1,
			"max_retries", maxRetries,
			"error", err,
		)

		time.Sleep(backoff)
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}

	slog.Error("Operation failed after retries", "operation", operation, "error", err)
	return err
}
