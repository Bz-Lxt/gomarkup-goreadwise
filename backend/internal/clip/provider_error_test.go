package clip_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"goreadwise/internal/clip"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type partialFailureBody struct {
	payload []byte
	err     error
	reads   int
}

func (b *partialFailureBody) Read(p []byte) (int, error) {
	switch b.reads {
	case 0:
		b.reads++
		return copy(p, b.payload), nil
	case 1:
		b.reads++
		return 0, b.err
	default:
		return 0, io.EOF
	}
}

func (b *partialFailureBody) Close() error {
	return nil
}

func TestRealProviderRejectsPartialResponseRead(t *testing.T) {
	streamErr := errors.New("read tcp: connection reset by peer")
	body := []byte("<html><head><title>Partial article</title></head><body><article><p>only the first half")
	provider := clip.NewReal(time.Second, 1<<20)
	provider.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:        "200 OK",
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          &partialFailureBody{payload: body, err: streamErr},
			ContentLength: -1,
			Request:       req,
		}, nil
	})}

	result, err := provider.Clip(context.Background(), "http://8.8.8.8/article")

	if !errors.Is(err, streamErr) {
		t.Fatalf("Clip() result = %+v, error = %v; want response stream error", result, err)
	}
}
