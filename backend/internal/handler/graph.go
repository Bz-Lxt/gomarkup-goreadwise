package handler

import (
	"net/http"

	"goreadwise/internal/httpx"
	"goreadwise/internal/model"
	"goreadwise/internal/service"
	"goreadwise/internal/store"
)

type GraphHandler struct {
	Graph *service.GraphService
	Tags  *store.DB
}

func (h GraphHandler) Full(w http.ResponseWriter, r *http.Request) {
	root := httpx.QueryString(r, "root", 64)
	depth := httpx.QueryInt(r, "depth", 0, 0, 3)
	if root != "" && depth > 0 {
		id, err := httpx.PathUUID(root)
		if err != nil {
			httpx.FromError(w, err)
			return
		}
		g, err := h.Graph.Subgraph(r.Context(), id, depth)
		if err != nil {
			httpx.FromError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, g)
		return
	}
	g, err := h.Graph.Full(r.Context())
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

func (h GraphHandler) Positions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Positions []model.PositionUpdate `json:"positions"`
	}
	if err := httpx.DecodeJSON(r, &req, 1<<20); err != nil {
		httpx.FromError(w, err)
		return
	}
	if err := h.Graph.SavePositions(r.Context(), req.Positions); err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": len(req.Positions)})
}

func (h GraphHandler) TagTree(w http.ResponseWriter, r *http.Request) {
	tags, err := h.Tags.ListTagTree(r.Context())
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	httpx.JSON(w, http.StatusOK, tags)
}
