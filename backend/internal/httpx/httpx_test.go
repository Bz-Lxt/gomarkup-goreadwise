package httpx

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRequireTitle(t *testing.T) {
	if _, err := RequireTitle("  "); err == nil {
		t.Fatal("empty")
	}
	if _, err := RequireTitle(strings.Repeat("x", 201)); err == nil {
		t.Fatal("long")
	}
	if _, err := RequireTitle("bad[title]"); err == nil {
		t.Fatal("bracket")
	}
	got, err := RequireTitle("  合法标题  ")
	if err != nil || got != "合法标题" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestRequireBodyLimit(t *testing.T) {
	if _, err := RequireBody(strings.Repeat("汉", 200_001)); err == nil {
		t.Fatal("expected overflow")
	}
}

func TestRequireURL(t *testing.T) {
	if _, err := RequireURL(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := RequireURL("https://example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestPathUUID(t *testing.T) {
	if _, err := PathUUID("nope"); err == nil {
		t.Fatal("expected")
	}
}

func TestDecodeJSONRejectsUnknown(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"nope":1}`))
	var dst struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(req, &dst, 1024); err == nil {
		t.Fatal("unknown field")
	}
}

func TestFromErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{ErrValidation, 400},
		{ErrNotFound, 404},
		{ErrConflict, 409},
		{ErrDenied, 403},
		{errors.New("x"), 500},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		FromError(rr, c.err)
		if rr.Code != c.code {
			t.Fatalf("%v -> %d", c.err, rr.Code)
		}
	}
}

func TestQueryIntClamp(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?n=999", nil)
	if QueryInt(req, "n", 1, 1, 10) != 10 {
		t.Fatal("max")
	}
	req = httptest.NewRequest(http.MethodGet, "/?n=abc", nil)
	if QueryInt(req, "n", 3, 1, 10) != 3 {
		t.Fatal("fallback")
	}
}

func TestJSONEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	JSON(rr, 200, map[string]string{"ok": "yes"})
	if !strings.Contains(rr.Body.String(), `"data"`) {
		t.Fatalf("%s", rr.Body.String())
	}
}

func TestClampAndSanitize(t *testing.T) {
	p, s := ClampPage(0, 0, 30, 100)
	if p != 1 || s != 30 {
		t.Fatal(p, s)
	}
	if Offset(3, 10) != 20 {
		t.Fatal(Offset(3, 10))
	}
	if SanitizeTagPath(" Tech/Go/ ") != "tech/go" {
		t.Fatal(SanitizeTagPath(" Tech/Go/ "))
	}
	if ErrorCodeOf(ErrConflict) != "CONFLICT" || ErrorCodeOf(nil) != "" {
		t.Fatal("codes")
	}
	long := strings.Repeat("问", 250)
	if utf8.RuneCountInString(SanitizeQuery(long)) != 200 {
		t.Fatal("query clamp")
	}
}
