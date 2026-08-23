package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"goreadwise/internal/clip"
	"goreadwise/internal/httpx"
	"goreadwise/internal/model"
)

type ClipService struct {
	Cards    *CardService
	Provider clip.Provider
	Mode     string
}

func (s *ClipService) Clip(ctx context.Context, rawURL string) (model.Card, error) {
	rawURL, err := httpx.RequireURL(rawURL)
	if err != nil {
		return model.Card{}, err
	}
	if s.Provider == nil {
		return model.Card{}, fmt.Errorf("%w: clip provider not configured", httpx.ErrValidation)
	}
	res, err := s.Provider.Clip(ctx, rawURL)
	if err != nil {
		return model.Card{}, err
	}
	title := strings.TrimSpace(res.Title)
	if title == "" {
		title = "Untitled clip"
	}
	body := res.Markdown
	if !strings.Contains(body, res.URL) {
		body = body + "\n\n> source: " + res.URL + "\n"
	}
	url := res.URL
	site := res.Site
	clipped := res.ClippedAt
	card, err := s.Cards.Create(ctx, model.CreateCardInput{
		Title:      title,
		Body:       body,
		Tags:       []string{"inbox/clip"},
		SourceURL:  &url,
		SourceSite: &site,
		ClippedAt:  &clipped,
	})
	if err != nil && errors.Is(err, httpx.ErrConflict) {
		title = fmt.Sprintf("%s (%s)", title, clipped.Format("150405"))
		card, err = s.Cards.Create(ctx, model.CreateCardInput{
			Title: title, Body: body, Tags: []string{"inbox/clip"},
			SourceURL: &url, SourceSite: &site, ClippedAt: &clipped,
		})
	}
	return card, err
}
