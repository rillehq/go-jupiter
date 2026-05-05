package jupiter

import (
	"context"
	"math"
	"net/http"
	"time"
)

// retryable returns true if the HTTP status code indicates a transient error
// that may succeed on retry.
func retryable(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode >= http.StatusInternalServerError
}

// backoff computes the wait duration for the given attempt using exponential
// backoff: 500ms × 2^attempt, capped at 10 seconds.
func backoff(attempt int) time.Duration {
	base := 500 * time.Millisecond
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	const maxBackoff = 10 * time.Second
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

// sleepCtx sleeps for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
