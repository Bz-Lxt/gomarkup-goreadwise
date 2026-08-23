package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"goreadwise/internal/httpx"
	"goreadwise/internal/service"
	"goreadwise/internal/store"
	"goreadwise/internal/worker"
)

type Deps struct {
	Cards     *service.CardService
	Graph     *service.GraphService
	Clip      *service.ClipService
	DB        *store.DB
	Pool      *worker.Pool
	Hub       *Hub
	ClipMode  string
	Started   time.Time
	Rebuilder *service.Rebuilder
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(httpx.Middleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id", "Location"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	cards := CardHandler{Svc: d.Cards}
	graph := GraphHandler{Graph: d.Graph, Tags: d.DB}
	clips := ClipHandler{Svc: d.Clip}

	r.Route("/api/v1", func(r chi.Router) {
		r.Method(http.MethodGet, "/health", HealthHandler{Started: d.Started, Mode: d.ClipMode})
		r.Method(http.MethodGet, "/metrics", MetricsHandler{Pool: d.Pool, Graph: d.Graph})
		r.Get("/events", d.Hub.Subscribe)

		r.Get("/cards", cards.List)
		r.Post("/cards", cards.Create)
		r.Get("/cards/suggest", cards.Suggest)
		r.Get("/cards/{id}", cards.Get)
		r.Patch("/cards/{id}", cards.Update)
		r.Delete("/cards/{id}", cards.Delete)
		r.Get("/cards/{id}/links", cards.Links)

		r.Get("/graph", graph.Full)
		r.Patch("/graph/positions", graph.Positions)
		r.Get("/tags", graph.TagTree)

		r.Post("/clips", clips.Create)
		r.Method(http.MethodPost, "/admin/rebuild", RebuildHandler{Graph: d.Graph, Rebuilder: d.Rebuilder})
	})
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return r
}
