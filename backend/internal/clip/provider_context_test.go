package clip_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"goreadwise/internal/clip"
)

type blockingTransport struct {
	started chan struct{}
	release chan struct{}
}

func (t *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	close(t.started)
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-t.release:
		return nil, errors.New("transport released")
	}
}

func TestRealProviderClipPropagatesCancellation(t *testing.T) {
	transport := &blockingTransport{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	provider := clip.NewReal(time.Minute, 1<<20)
	provider.Client.Transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := provider.Clip(ctx, "http://8.8.8.8/article")
		done <- err
	}()

	select {
	case <-transport.started:
	case <-time.After(time.Second):
		t.Fatal("HTTP request did not reach the transport")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Clip returned %v after cancellation, want context.Canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		close(transport.release)
		<-done
		t.Fatal("Clip remained blocked after its context was canceled")
	}
}
