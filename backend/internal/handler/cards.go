package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"goreadwise/internal/httpx"
	"goreadwise/internal/model"
	"goreadwise/internal/service"
)

type CardHandler struct {
	Svc *service.CardService
}

type cardWriteReq struct {
	Title *string  `json:"title"`
	Body  *string  `json:"body"`
	Tags  []string `json:"tags"`
}

func (h CardHandler) List(w http.ResponseWriter, r *http.Request) {
	f := model.CardListFilter{
		Query:    httpx.QueryString(r, "q", 200),
		TagPath:  httpx.QueryString(r, "tag", 200),
		Page:     httpx.QueryInt(r, "page", 1, 1, 10000),
		PageSize: httpx.QueryInt(r, "page_size", 30, 1, 100),
	}
	cards, total, err := h.Svc.List(r.Context(), f)
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	if cards == nil {
		cards = []model.Card{}
	}
	httpx.JSONPage(w, http.StatusOK, cards, httpx.PageMeta{Page: f.Page, PageSize: f.PageSize, Total: total})
}

func (h CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req cardWriteReq
	if err := httpx.DecodeJSON(r, &req, 1<<20); err != nil {
		httpx.FromError(w, err)
		return
	}
	if req.Title == nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION", "title is required")
		return
	}
	body := ""
	if req.Body != nil {
		body = *req.Body
	}
	card, err := h.Svc.Create(r.Context(), model.CreateCardInput{
		Title: *req.Title, Body: body, Tags: req.Tags,
	})
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/cards/"+card.ID.String())
	httpx.JSON(w, http.StatusCreated, card)
}

func (h CardHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathUUID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	card, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, card)
}

func (h CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathUUID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	var req cardWriteReq
	if err := httpx.DecodeJSON(r, &req, 1<<20); err != nil {
		httpx.FromError(w, err)
		return
	}
	card, err := h.Svc.Update(r.Context(), id, model.UpdateCardInput{
		Title: req.Title, Body: req.Body, Tags: req.Tags,
	})
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, card)
}

func (h CardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathUUID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	if err := h.Svc.Delete(r.Context(), id); err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id.String()})
}

func (h CardHandler) Links(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathUUID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	card, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"out_links":  card.OutLinks,
		"back_links": card.BackLinks,
	})
}

func (h CardHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	q := httpx.QueryString(r, "q", 200)
	cards, err := h.Svc.Suggest(r.Context(), q)
	if err != nil {
		httpx.FromError(w, err)
		return
	}
	if cards == nil {
		cards = []model.Card{}
	}
	httpx.JSON(w, http.StatusOK, cards)
}
