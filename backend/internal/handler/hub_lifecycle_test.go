package handler_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"goreadwise/internal/handler"
)

func TestHubSubscribeRemovesQuietClientOnDisconnect(t *testing.T) {
	hub := handler.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		hub.Subscribe(recorder, req)
	}()

	deadline := time.Now().Add(time.Second)
	for hub.Size() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := hub.Size(); got != 1 {
		t.Fatalf("active subscriptions = %d, want 1", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscription did not stop after its request was canceled")
	}

	if got := hub.Size(); got != 0 {
		t.Fatalf("active subscriptions after disconnect = %d, want 0", got)
	}
}
