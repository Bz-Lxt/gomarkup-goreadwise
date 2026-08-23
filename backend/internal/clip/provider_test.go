package clip

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// slowUpstream blocks until the request's context is done, simulating a page
// that never finishes sending the body while the connection stays open.
type slowUpstream struct {
	hits atomic.Int64
}

func (s *slowUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	<-r.Context().Done()
	_, _ = w.Write([]byte("<html><body><article>late</article></body></html>"))
}

// waitForHit polls the upstream hit counter until at least one request has
// arrived, proving the fetch is in-flight before the test proceeds.
func waitForHit(t *testing.T, hits *atomic.Int64, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for hits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hits.Load() == 0 {
		t.Fatalf("upstream never received request within %v", d)
	}
}

// TestRealProviderFetchCancelPropagation guards against the original bug where
// http.NewRequest (without context) meant cancelling the caller's ctx did NOT
// cancel the in-flight fetch. The upstream never replies; cancelling ctx must
// cause fetch to return promptly (well under the Client.Timeout) with an error
// that wraps context.Canceled.
func TestRealProviderFetchCancelPropagation(t *testing.T) {
	var upstream slowUpstream
	srv := httptest.NewServer(&upstream)
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/post")
	rp := NewReal(30*time.Second, 1<<20) // long client timeout on purpose

	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		_, err := rp.fetch(ctx, u)
		errs <- err
	}()

	waitForHit(t, &upstream.hits, 2*time.Second)
	cancel() // simulate the HTTP client going away

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected error from cancelled fetch, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected error to wrap context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fetch did not return within 2s after cancel; ctx was not propagated to the HTTP request")
	}
}

// TestRealProviderFetchDeadlinePropagation verifies that a context deadline also
// reaches the in-flight fetch, in addition to explicit cancellation.
func TestRealProviderFetchDeadlinePropagation(t *testing.T) {
	var upstream slowUpstream
	srv := httptest.NewServer(&upstream)
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/post")
	rp := NewReal(30*time.Second, 1<<20)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := rp.fetch(ctx, u)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from expired deadline, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected error to wrap context.DeadlineExceeded, got %v", err)
	}
	// Must return close to the 50ms deadline, not the 30s Client.Timeout.
	if elapsed > 2*time.Second {
		t.Fatalf("fetch took %v; deadline was not propagated to the HTTP request", elapsed)
	}
}

// TestRealProviderFetchHappyPath confirms the normal flow still works once ctx
// is attached, so the fix didn't break the non-cancel path.
func TestRealProviderFetchHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>Hi</title></head><body><article><p>hello</p></article></body></html>"))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/p")
	rp := NewReal(5*time.Second, 1<<20)
	res, err := rp.fetch(context.Background(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title != "Hi" || res.Provider != "real" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Markdown == "" {
		t.Fatal("expected non-empty markdown")
	}
}
