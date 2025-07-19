// Retry package is used for retrying operations.
package retry

import (
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type RetryableError interface {
	IsRetryable() bool
}

// RetryConfig contains max attempts and delay.
type RetryConfig struct {
	MaxAttempts int
	Delays      []time.Duration
}

// DefaultRetryConfig makes default config for retryerror.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second},
	}
}

// IsRetryableError checks if error is retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// PG errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.ConnectionException
	}

	// Non PG errors
	var retryableErr RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}

	return false
}

// Do initialises retry cycle for a selected operation.
func Do(config *RetryConfig, operation func() error) error {
	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		if !IsRetryableError(err) {
			return err
		}

		if attempt < len(config.Delays) {
			time.Sleep(config.Delays[attempt])
		}
	}
	return lastErr
}
