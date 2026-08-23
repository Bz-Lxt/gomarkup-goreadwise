package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthRoute(t *testing.T) {
	r := NewRouter(Deps{Hub: NewHub(), ClipMode: "mock", Started: time.Now()})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	data := env["data"].(map[string]any)
	if data["status"] != "ok" || data["clip_mode"] != "mock" {
		t.Fatalf("%v", data)
	}
}

func TestHealthz(t *testing.T) {
	r := NewRouter(Deps{Hub: NewHub(), Started: time.Now()})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
}

func TestMetricsWithoutPool(t *testing.T) {
	r := NewRouter(Deps{Hub: NewHub(), Started: time.Now()})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
}
