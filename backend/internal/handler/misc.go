package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"goreadwise/internal/clock"
	"goreadwise/internal/httpx"
	"goreadwise/internal/service"
	"goreadwise/internal/worker"
)

type HealthHandler struct {
	Started time.Time
	Mode    string
}

func (h HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"time":      clock.Format(clock.Now()),
		"uptime_s":  int(time.Since(h.Started).Seconds()),
		"go":        runtime.Version(),
		"clip_mode": h.Mode,
		"env":       os.Getenv("APP_ENV"),
	})
}

type MetricsHandler struct {
	Pool  *worker.Pool
	Graph *service.GraphService
}

func (h MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{}
	if h.Pool != nil {
		s := h.Pool.Snapshot()
		out["queue_depth"] = s.QueueDepth
		out["queue_cap"] = s.QueueCap
		out["sync_fallback"] = s.SyncFallback
		out["jobs_done"] = s.JobsDone
		out["jobs_failed"] = s.JobsFailed
	}
	if h.Graph != nil {
		m, err := h.Graph.Metrics(r.Context())
		if err == nil {
			out["graph_version"] = m.GraphVersion
			out["card_count"] = m.CardCount
			out["edge_count"] = m.EdgeCount
		}
	}
	httpx.JSON(w, http.StatusOK, out)
}

type ClipHandler struct {
	Svc *service.ClipService
}

func (h ClipHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := httpx.DecodeJSON(r, &req, 8192); err != nil {
		httpx.FromError(w, err)
		return
	}
	card, err := h.Svc.Clip(r.Context(), req.URL)
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, card)
}

type RebuildHandler struct {
	Graph     *service.GraphService
	Rebuilder *service.Rebuilder
}

func (h RebuildHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := 0
	if h.Rebuilder != nil {
		var err error
		n, err = h.Rebuilder.All(r.Context())
		if err != nil {
			httpx.FromError(w, err)
			return
		}
	} else if h.Graph != nil {
		h.Graph.Invalidate()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"rebuilt": true, "cards": n})
}

type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[chan []byte]struct{}{}}
}

func (h *Hub) Broadcast(event string, payload any) {
	body, _ := json.Marshal(map[string]any{
		"event":   event,
		"payload": payload,
		"ts":      clock.FormatRFC3339(clock.Now()),
	})
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "event: %s\ndata: %s\n\n", event, body)
	msg := buf.Bytes()
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Hub) Subscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
	}()
	_, _ = w.Write([]byte("event: hello\ndata: {\"ok\":true}\n\n"))
	flusher.Flush()
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case msg := <-ch:
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}

func (h *Hub) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
