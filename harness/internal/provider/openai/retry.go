package openai

import (
	"context"
	"time"
)

const maxAttempts = 3
const initialBackoff = time.Second

func withRetry(ctx context.Context, fn func() error) error {
	var err error
	backoff := initialBackoff
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = fn(); err == nil {
			return nil
		}
		if attempt == maxAttempts-1 {
			break
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}
	return err
}
