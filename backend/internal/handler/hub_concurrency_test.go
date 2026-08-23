package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"goreadwise/internal/handler"
)

type hubStreamRecorder struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
}

func newHubStreamRecorder() *hubStreamRecorder {
	return &hubStreamRecorder{header: make(http.Header)}
}

func (w *hubStreamRecorder) Header() http.Header { return w.header }

func (w *hubStreamRecorder) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}

func (*hubStreamRecorder) WriteHeader(int) {}
func (*hubStreamRecorder) Flush() {}

func (w *hubStreamRecorder) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

type gatedHubPayload struct {
	ready   chan<- struct{}
	release <-chan struct{}
	raw     []byte
}

func (p gatedHubPayload) MarshalJSON() ([]byte, error) {
	p.ready <- struct{}{}
	<-p.release
	return p.raw, nil
}

func TestHubConcurrentBroadcastKeepsSSEFramesIntact(t *testing.T) {
	hub := handler.NewHub()
	recorder := newHubStreamRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
		hub.Subscribe(recorder, req)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	})

	waitForHubState(t, func() bool {
		return hub.Size() == 1 && strings.Contains(recorder.String(), "event: hello\n")
	})

	const broadcasts = 8
	ready := make(chan struct{}, broadcasts)
	release := make(chan struct{})
	expected := make(map[string]int, broadcasts)
	var wg sync.WaitGroup
	for i := 0; i < broadcasts; i++ {
		event := "graph:invalidated:" + strconv.Itoa(i)
		expected[event] = i
		raw, err := json.Marshal(map[string]any{
			"id":      i,
			"padding": strings.Repeat(string(rune('a'+i)), 64<<10),
		})
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func(event string, payload gatedHubPayload) {
			defer wg.Done()
			hub.Broadcast(event, payload)
		}(event, gatedHubPayload{ready: ready, release: release, raw: raw})
	}

	for i := 0; i < broadcasts; i++ {
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent broadcasts did not reach the payload encoder")
		}
	}
	close(release)
	broadcastDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(broadcastDone)
	}()
	select {
	case <-broadcastDone:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent broadcasts did not return")
	}

	waitForHubState(t, func() bool {
		return strings.Count(recorder.String(), "\n\n") >= broadcasts+1
	})
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("subscription did not stop after cancellation")
	}

	frames := strings.Split(strings.TrimSpace(recorder.String()), "\n\n")
	if len(frames) != broadcasts+1 {
		t.Fatalf("got %d SSE frames, want %d", len(frames), broadcasts+1)
	}
	seen := make(map[string]bool, broadcasts)
	for _, frame := range frames[1:] {
		lines := strings.Split(frame, "\n")
		if len(lines) != 2 || !strings.HasPrefix(lines[0], "event: ") || !strings.HasPrefix(lines[1], "data: ") {
			t.Fatalf("malformed SSE frame: %q", frame)
		}
		event := strings.TrimPrefix(lines[0], "event: ")
		wantID, ok := expected[event]
		if !ok || seen[event] {
			t.Fatalf("unexpected or duplicate event %q", event)
		}
		var envelope struct {
			Event   string `json:"event"`
			Payload struct {
				ID int `json:"id"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &envelope); err != nil {
			t.Fatalf("event %q contains invalid JSON: %v", event, err)
		}
		if envelope.Event != event || envelope.Payload.ID != wantID {
			t.Fatalf("mismatched SSE frame: line event=%q envelope event=%q payload id=%d", event, envelope.Event, envelope.Payload.ID)
		}
		seen[event] = true
	}
}

func waitForHubState(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for SSE state")
		}
		time.Sleep(time.Millisecond)
	}
}
