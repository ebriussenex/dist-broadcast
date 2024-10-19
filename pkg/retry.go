package retry

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

func FixedInterval(fn func() error, times int, interval time.Duration) error {

	var retryErrs error

	for i := 0; i < times; i++ {
		if i != 0 {
			<-time.After(interval)
		}
		if err := fn(); err != nil {
			errors.Join(retryErrs, err)
			slog.Warn(fmt.Sprintf("failed to complete operation, retrying: %s", err.Error()))
		} else {
			return nil
		}
	}
	return fmt.Errorf("failed to complete operation with retry, err: %w", retryErrs)
}
