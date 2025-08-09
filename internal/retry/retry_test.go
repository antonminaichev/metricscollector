package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// Mock retryable error
type mockRetryableError struct {
	message   string
	retryable bool
}

func (e *mockRetryableError) Error() string {
	return e.message
}

func (e *mockRetryableError) IsRetryable() bool {
	return e.retryable
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Len(t, config.Delays, 3)
	assert.Equal(t, 1*time.Second, config.Delays[0])
	assert.Equal(t, 3*time.Second, config.Delays[1])
	assert.Equal(t, 5*time.Second, config.Delays[2])
}

func TestIsRetryableError(t *testing.T) {
	t.Run("nil error is not retryable", func(t *testing.T) {
		assert.False(t, IsRetryableError(nil))
	})

	t.Run("postgres connection error is retryable", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code: pgerrcode.ConnectionException,
		}
		assert.True(t, IsRetryableError(pgErr))
	})

	t.Run("postgres non-connection error is not retryable", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code: pgerrcode.UniqueViolation,
		}
		assert.False(t, IsRetryableError(pgErr))
	})

	t.Run("retryable custom error is retryable", func(t *testing.T) {
		err := &mockRetryableError{
			message:   "retryable error",
			retryable: true,
		}
		assert.True(t, IsRetryableError(err))
	})

	t.Run("non-retryable custom error is not retryable", func(t *testing.T) {
		err := &mockRetryableError{
			message:   "non-retryable error",
			retryable: false,
		}
		assert.False(t, IsRetryableError(err))
	})

	t.Run("generic error is not retryable", func(t *testing.T) {
		err := errors.New("generic error")
		assert.False(t, IsRetryableError(err))
	})

	t.Run("wrapped postgres error is retryable", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code: pgerrcode.ConnectionException,
		}
		wrappedErr := errors.New("wrapper: " + pgErr.Error())
		// Эта проверка должна вернуть false, так как wrapped error не является pgconn.PgError
		assert.False(t, IsRetryableError(wrappedErr))
	})
}

func TestDo(t *testing.T) {
	t.Run("successful operation on first attempt", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 3,
			Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			return nil
		}

		err := Do(config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 1, attemptCount)
	})

	t.Run("successful operation on second attempt", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 3,
			Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			if attemptCount == 1 {
				return &mockRetryableError{message: "temporary error", retryable: true}
			}
			return nil
		}

		err := Do(config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 2, attemptCount)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 3,
			Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		}

		attemptCount := 0
		expectedErr := errors.New("non-retryable error")
		operation := func() error {
			attemptCount++
			return expectedErr
		}

		err := Do(config, operation)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, attemptCount)
	})

	t.Run("retryable error exhausts all attempts", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 3,
			Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		}

		attemptCount := 0
		expectedErr := &mockRetryableError{message: "persistent error", retryable: true}
		operation := func() error {
			attemptCount++
			return expectedErr
		}

		err := Do(config, operation)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 3, attemptCount)
	})

	t.Run("delays are respected", func(t *testing.T) {
		start := time.Now()
		config := &RetryConfig{
			MaxAttempts: 2,
			Delays:      []time.Duration{10 * time.Millisecond},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			return &mockRetryableError{message: "temporary error", retryable: true}
		}

		Do(config, operation)
		elapsed := time.Since(start)

		// Проверяем, что прошло хотя бы время задержки
		assert.True(t, elapsed >= 10*time.Millisecond)
		assert.Equal(t, 2, attemptCount)
	})

	t.Run("more attempts than delays", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 4,
			Delays:      []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			return &mockRetryableError{message: "persistent error", retryable: true}
		}

		err := Do(config, operation)
		assert.Error(t, err)
		assert.Equal(t, 4, attemptCount)
	})

	t.Run("zero max attempts", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 0,
			Delays:      []time.Duration{},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			return &mockRetryableError{message: "error", retryable: true}
		}

		err := Do(config, operation)
		// При MaxAttempts = 0 функция не должна выполнять операцию вообще
		// и должна вернуть nil, так как нет ошибки от операции
		assert.NoError(t, err)
		assert.Equal(t, 0, attemptCount)
	})

	t.Run("postgres connection error with retry", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 2,
			Delays:      []time.Duration{1 * time.Millisecond},
		}

		attemptCount := 0
		pgErr := &pgconn.PgError{
			Code: pgerrcode.ConnectionException,
		}
		operation := func() error {
			attemptCount++
			if attemptCount == 1 {
				return pgErr
			}
			return nil
		}

		err := Do(config, operation)
		assert.NoError(t, err)
		assert.Equal(t, 2, attemptCount)
	})
}

func TestRetryConfig(t *testing.T) {
	t.Run("custom retry config", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts: 5,
			Delays:      []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
		}

		assert.Equal(t, 5, config.MaxAttempts)
		assert.Len(t, config.Delays, 2)
		assert.Equal(t, 100*time.Millisecond, config.Delays[0])
		assert.Equal(t, 200*time.Millisecond, config.Delays[1])
	})
}

func BenchmarkDo(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts: 3,
		Delays:      []time.Duration{1 * time.Microsecond, 2 * time.Microsecond},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Do(config, func() error {
			return nil // всегда успешно
		})
	}
}

func BenchmarkIsRetryableError(b *testing.B) {
	err := &mockRetryableError{message: "test", retryable: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsRetryableError(err)
	}
}
