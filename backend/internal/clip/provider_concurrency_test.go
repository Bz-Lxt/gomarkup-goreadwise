package clip_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"goreadwise/internal/clip"
)

type redirectTransport struct {
	aEntered chan struct{}
	bEntered chan struct{}
	releaseA chan struct{}
	releaseB chan struct{}
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Path {
	case "/a-start":
		close(t.aEntered)
		<-t.releaseA
		return redirectResponse(req, "http://8.8.8.8/a-final"), nil
	case "/a-final":
		return htmlResponse(req, "Article A"), nil
	case "/b":
		close(t.bEntered)
		<-t.releaseB
		return nil, req.Context().Err()
	default:
		return htmlResponse(req, "Unexpected"), nil
	}
}

func redirectResponse(req *http.Request, location string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusFound,
		Status:     "302 Found",
		Header:     http.Header{"Location": []string{location}},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}
}

func htmlResponse(req *http.Request, title string) *http.Response {
	body := "<html><head><title>" + title + "</title></head><body><article>ok</article></body></html>"
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

type clipOutcome struct {
	result clip.Result
	err    error
}

func TestRealProviderConcurrentRedirectContextsAreIsolated(t *testing.T) {
	transport := &redirectTransport{
		aEntered: make(chan struct{}),
		bEntered: make(chan struct{}),
		releaseA: make(chan struct{}),
		releaseB: make(chan struct{}),
	}
	var releaseAOnce sync.Once
	var releaseBOnce sync.Once
	releaseA := func() { releaseAOnce.Do(func() { close(transport.releaseA) }) }
	releaseB := func() { releaseBOnce.Do(func() { close(transport.releaseB) }) }
	defer releaseA()
	defer releaseB()

	provider := clip.NewReal(2*time.Second, 1<<20)
	provider.Client.Transport = transport
	provider.Client.Timeout = 0

	aDone := make(chan clipOutcome, 1)
	go func() {
		result, err := provider.Clip(context.Background(), "http://8.8.8.8/a-start")
		aDone <- clipOutcome{result: result, err: err}
	}()
	waitForSignal(t, transport.aEntered, "first request to enter transport")

	bCtx, cancelB := context.WithCancel(context.Background())
	defer cancelB()
	bDone := make(chan error, 1)
	go func() {
		_, err := provider.Clip(bCtx, "http://8.8.8.8/b")
		bDone <- err
	}()
	waitForSignal(t, transport.bEntered, "second request to enter transport")

	cancelB()
	releaseA()

	var a clipOutcome
	select {
	case a = <-aDone:
	case <-time.After(time.Second):
		t.Fatal("first request did not finish after its redirect was released")
	}
	releaseB()
	var bErr error
	select {
	case bErr = <-bDone:
	case <-time.After(time.Second):
		t.Fatal("canceled second request did not finish")
	}

	if !errors.Is(bErr, context.Canceled) {
		t.Fatalf("second request returned %v, want context cancellation", bErr)
	}
	if a.err != nil {
		t.Fatalf("canceling a concurrent request interrupted the first request: %v", a.err)
	}
	if a.result.Title != "Article A" {
		t.Fatalf("first request returned title %q, want %q", a.result.Title, "Article A")
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
