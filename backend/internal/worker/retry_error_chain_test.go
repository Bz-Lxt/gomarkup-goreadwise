package worker_test

import (
	"context"
	"fmt"
	"testing"

	"goreadwise/internal/worker"
)

func TestShouldRetryWrappedDeadline(t *testing.T) {
	err := fmt.Errorf("apply graph update: %w", fmt.Errorf("fetch page: %w", context.DeadlineExceeded))

	if !worker.ShouldRetry(err, 0) {
		t.Fatalf("ShouldRetry(%v, 0) = false, want true", err)
	}
}
