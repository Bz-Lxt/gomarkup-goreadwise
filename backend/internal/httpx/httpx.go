package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"goreadwise/internal/clock"
	"goreadwise/internal/logger"
)

type Envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
	Meta  *PageMeta `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PageMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

var (
	ErrValidation = errors.New("validation")
	ErrNotFound   = errors.New("not_found")
	ErrConflict   = errors.New("conflict")
	ErrDenied     = errors.New("denied")
)

// StatusClientClosed is the non-standard 499 used by nginx to signal that the
// client went away before the response could be written. It is returned for
// context-cancellation errors so handlers don't log a noisy 500.
const StatusClientClosed = 499

func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, Envelope{Data: data})
}

func JSONPage(w http.ResponseWriter, status int, data any, meta PageMeta) {
	write(w, status, Envelope{Data: data, Meta: &meta})
}

func Fail(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Envelope{Error: &APIError{Code: code, Message: message}})
}

func FromError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		// Client went away or the request deadline elapsed; nothing useful to
		// write back to a (likely already-closed) connection. Avoid a noisy 500.
		Fail(w, StatusClientClosed, "CLIENT_CLOSED", "client closed connection")
	case errors.Is(err, ErrValidation):
		Fail(w, http.StatusBadRequest, "VALIDATION", err.Error())
	case errors.Is(err, ErrNotFound):
		Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrConflict):
		Fail(w, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, ErrDenied):
		Fail(w, http.StatusForbidden, "CLIP_DENIED", err.Error())
	default:
		logger.L().Error("unhandled", slog.String("err", err.Error()))
		Fail(w, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

func write(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(body)
}

func DecodeJSON(r *http.Request, dst any, maxBytes int64) error {
	if r.Body == nil {
		return wrap(ErrValidation, "empty body")
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return wrap(ErrValidation, "invalid json: "+err.Error())
	}
	return nil
}

func QueryInt(r *http.Request, key string, fallback, min, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func QueryString(r *http.Request, key string, maxLen int) string {
	s := strings.TrimSpace(r.URL.Query().Get(key))
	if maxLen > 0 && utf8.RuneCountInString(s) > maxLen {
		rs := []rune(s)
		s = string(rs[:maxLen])
	}
	return s
}

func PathUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, wrap(ErrValidation, "invalid id")
	}
	return id, nil
}

func RequireTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	n := utf8.RuneCountInString(title)
	if n == 0 {
		return "", wrap(ErrValidation, "title is required")
	}
	if n > 200 {
		return "", wrap(ErrValidation, "title exceeds 200 characters")
	}
	if strings.ContainsAny(title, "[]\n") {
		return "", wrap(ErrValidation, "title cannot contain brackets or newlines")
	}
	return title, nil
}

func RequireBody(body string) (string, error) {
	if utf8.RuneCountInString(body) > 200_000 {
		return "", wrap(ErrValidation, "body exceeds 200000 characters")
	}
	return body, nil
}

func RequireURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", wrap(ErrValidation, "url is required")
	}
	if utf8.RuneCountInString(raw) > 2048 {
		return "", wrap(ErrValidation, "url exceeds 2048 characters")
	}
	return raw, nil
}

func RequestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-Id"); v != "" {
		return v
	}
	return uuid.NewString()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := clock.Now()
		ww := &statusWriter{ResponseWriter: w, code: 200}
		id := RequestID(r)
		ww.Header().Set("X-Request-Id", id)
		next.ServeHTTP(ww, r)
		logger.L().Info("http",
			slog.String("id", id),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.code),
			slog.Int64("ms", time.Since(start).Milliseconds()),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func wrap(base error, msg string) error {
	return errors.Join(base, errors.New(msg))
}

func ClampPage(page, size, defSize, maxSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defSize
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}

func Offset(page, size int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * size
}

func SanitizeQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ReplaceAll(q, "\x00", "")
	if utf8.RuneCountInString(q) > 200 {
		return string([]rune(q)[:200])
	}
	return q
}

func SanitizeTagPath(p string) string {
	p = strings.ToLower(strings.Trim(strings.TrimSpace(p), "/"))
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	ok := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.ContainsAny(part, " \t\n") {
			continue
		}
		ok = append(ok, part)
	}
	return strings.Join(ok, "/")
}

func ErrorCodeOf(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return "CLIENT_CLOSED"
	case errors.Is(err, ErrValidation):
		return "VALIDATION"
	case errors.Is(err, ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, ErrConflict):
		return "CONFLICT"
	case errors.Is(err, ErrDenied):
		return "CLIP_DENIED"
	default:
		return "INTERNAL"
	}
}
