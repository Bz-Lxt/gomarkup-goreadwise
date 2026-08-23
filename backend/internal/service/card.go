package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/engine"
	"goreadwise/internal/httpx"
	"goreadwise/internal/logger"
	"goreadwise/internal/model"
	"goreadwise/internal/store"
)

type Broadcaster interface {
	Broadcast(event string, payload any)
}

type Enqueuer interface {
	Submit(job model.GraphJob)
	SubmitOrRun(job model.GraphJob, syncFn func(context.Context) error)
}

type CardService struct {
	DB    *store.DB
	Queue Enqueuer
	Hub   Broadcaster
}

func (s *CardService) Create(ctx context.Context, in model.CreateCardInput) (model.Card, error) {
	title, err := httpx.RequireTitle(in.Title)
	if err != nil {
		return model.Card{}, err
	}
	body, err := httpx.RequireBody(in.Body)
	if err != nil {
		return model.Card{}, err
	}
	in.Title, in.Body = title, body
	parsed := engine.Parse(body)
	var card model.Card
	err = s.DB.WithTx(ctx, func(tx pgx.Tx) error {
		c, err := s.DB.InsertCard(ctx, tx, in, parsed.ContentHash)
		if err != nil {
			return err
		}
		tags := mergeTagPaths(in.Tags, parsed.Tags)
		c.Tags, err = s.DB.ReplaceCardTags(ctx, tx, c.ID, tags)
		if err != nil {
			return err
		}
		if err := s.writeOutgoing(ctx, tx, c.ID, nil, parsed.Links); err != nil {
			return err
		}
		if _, err := s.DB.ResolveDanglingByTitle(ctx, tx, c.Title, c.ID); err != nil {
			return err
		}
		if err := s.enqueueAsync(ctx, tx, c.ID, parsed.ContentHash, model.JobKindSnapshot, model.JobKindResolve, model.JobKindStats); err != nil {
			return err
		}
		card = c
		return nil
	})
	if err != nil {
		return model.Card{}, err
	}
	return s.Get(ctx, card.ID)
}

func (s *CardService) Update(ctx context.Context, id uuid.UUID, in model.UpdateCardInput) (model.Card, error) {
	cur, err := s.DB.GetCard(ctx, id)
	if err != nil {
		return model.Card{}, err
	}
	title := cur.Title
	if in.Title != nil {
		title, err = httpx.RequireTitle(*in.Title)
		if err != nil {
			return model.Card{}, err
		}
	}
	body := cur.Body
	if in.Body != nil {
		body, err = httpx.RequireBody(*in.Body)
		if err != nil {
			return model.Card{}, err
		}
	}
	renamed := engine.NormalizeTitle(title) != cur.TitleNorm
	parsed := engine.Parse(body)
	err = s.DB.WithTx(ctx, func(tx pgx.Tx) error {
		oldOutgoing, err := s.DB.ListOutgoing(ctx, tx, id)
		if err != nil {
			return err
		}
		updated, err := s.DB.UpdateCard(ctx, tx, id, title, body, parsed.ContentHash, true)
		if err != nil {
			return err
		}
		tags := parsed.Tags
		if in.Tags != nil {
			tags = mergeTagPaths(in.Tags, parsed.Tags)
		}
		if _, err := s.DB.ReplaceCardTags(ctx, tx, id, tags); err != nil {
			return err
		}
		if err := s.writeOutgoing(ctx, tx, id, oldOutgoing, parsed.Links); err != nil {
			return err
		}
		kinds := []string{model.JobKindSnapshot, model.JobKindStats}
		if renamed {
			if err := s.cascadeRename(ctx, tx, cur.Title, title); err != nil {
				return err
			}
			kinds = append(kinds, model.JobKindRename)
		}
		if err := s.enqueueAsync(ctx, tx, updated.ID, parsed.ContentHash, kinds...); err != nil {
			return err
		}
		_ = s.DB.InsertOpLog(ctx, tx, "card.update", map[string]any{
			"id": id.String(), "renamed": renamed, "hash": parsed.ContentHash,
		})
		return nil
	})
	if err != nil {
		return model.Card{}, err
	}
	return s.Get(ctx, id)
}

func (s *CardService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.DB.WithTx(ctx, func(tx pgx.Tx) error {
		if err := s.DB.SoftDeleteCard(ctx, tx, id); err != nil {
			return err
		}
		return s.enqueueAsync(ctx, tx, id, "deleted", model.JobKindSnapshot, model.JobKindStats)
	})
}

func (s *CardService) Get(ctx context.Context, id uuid.UUID) (model.Card, error) {
	c, err := s.DB.GetCard(ctx, id)
	if err != nil {
		return c, err
	}
	c.Tags, err = s.DB.ListCardTags(ctx, id)
	if err != nil {
		return c, err
	}
	c.OutLinks, err = s.DB.ListOutgoing(ctx, nil, id)
	if err != nil {
		return c, err
	}
	c.BackLinks, err = s.DB.ListBacklinks(ctx, id, c.Title)
	if c.Tags == nil {
		c.Tags = []model.Tag{}
	}
	if c.OutLinks == nil {
		c.OutLinks = []model.Link{}
	}
	if c.BackLinks == nil {
		c.BackLinks = []model.Link{}
	}
	return c, err
}

func (s *CardService) List(ctx context.Context, f model.CardListFilter) ([]model.Card, int, error) {
	cards, total, err := s.DB.ListCards(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	tagMap, err := s.DB.ListCardTagPaths(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range cards {
		for _, p := range tagMap[cards[i].ID] {
			cards[i].Tags = append(cards[i].Tags, model.Tag{FullPath: p, Name: leafName(p)})
		}
	}
	return cards, total, nil
}

func (s *CardService) Suggest(ctx context.Context, q string) ([]model.Card, error) {
	return s.DB.SuggestTitles(ctx, q, 12)
}

func (s *CardService) writeOutgoing(ctx context.Context, tx pgx.Tx, source uuid.UUID, existing []model.Link, parsed []engine.WikiLink) error {
	diff := engine.DiffOutgoing(existing, parsed)
	if err := s.DB.DeleteLinksByIDs(ctx, tx, diff.ToDelete); err != nil {
		return err
	}
	for _, w := range diff.ToInsert {
		var target *uuid.UUID
		found, err := s.DB.GetCardByTitleNorm(ctx, tx, engine.NormalizeTitle(w.Target))
		if err == nil {
			id := found.ID
			target = &id
		} else if !errors.Is(err, httpx.ErrNotFound) {
			return err
		}
		if err := s.DB.InsertLink(ctx, tx, source, w, target); err != nil {
			return err
		}
	}
	logger.L().Debug("link-diff",
		slog.Int("insert", len(diff.ToInsert)),
		slog.Int("delete", len(diff.ToDelete)),
		slog.Int("keep", diff.Unchanged),
	)
	return nil
}

func (s *CardService) cascadeRename(ctx context.Context, tx pgx.Tx, oldTitle, newTitle string) error {
	refs, err := s.DB.ListLinksByTargetTitle(ctx, oldTitle)
	if err != nil {
		return err
	}
	seen := map[uuid.UUID]struct{}{}
	for _, ref := range refs {
		if _, ok := seen[ref.SourceCardID]; ok {
			continue
		}
		seen[ref.SourceCardID] = struct{}{}
		src, err := s.DB.GetCard(ctx, ref.SourceCardID)
		if err != nil {
			continue
		}
		rewritten := engine.RewriteWikilinks(src.Body, oldTitle, newTitle)
		if rewritten == src.Body {
			continue
		}
		parsed := engine.Parse(rewritten)
		if _, err := s.DB.UpdateCard(ctx, tx, src.ID, src.Title, rewritten, parsed.ContentHash, true); err != nil {
			return err
		}
		oldOut, _ := s.DB.ListOutgoing(ctx, tx, src.ID)
		if err := s.writeOutgoing(ctx, tx, src.ID, oldOut, parsed.Links); err != nil {
			return err
		}
	}
	if err := s.DB.RewriteTargetTitles(ctx, tx, oldTitle, newTitle); err != nil {
		return err
	}
	return s.DB.InsertOpLog(ctx, tx, "rename.cascade", map[string]any{
		"from": oldTitle, "to": newTitle, "sources": len(seen),
	})
}

func (s *CardService) enqueueAsync(ctx context.Context, tx pgx.Tx, cardID uuid.UUID, hash string, kinds ...string) error {
	for _, kind := range kinds {
		job, inserted, err := s.DB.EnqueueJob(ctx, tx, cardID, hash, kind)
		if err != nil {
			return err
		}
		if !inserted {
			continue
		}
		if s.Queue != nil {
			s.Queue.Submit(job)
		}
	}
	return nil
}

func (s *CardService) ApplyAsync(ctx context.Context, job model.GraphJob) error {
	switch job.Kind {
	case model.JobKindResolve:
		card, err := s.DB.GetCard(ctx, job.CardID)
		if err != nil {
			return nil
		}
		return s.DB.WithTx(ctx, func(tx pgx.Tx) error {
			n, err := s.DB.ResolveDanglingByTitle(ctx, tx, card.Title, card.ID)
			if err != nil {
				return err
			}
			if n > 0 {
				_, _ = s.DB.IncrGlobalVersion(ctx, tx)
			}
			return nil
		})
	case model.JobKindSnapshot, model.JobKindStats, model.JobKindRename:
		return s.DB.WithTx(ctx, func(tx pgx.Tx) error {
			ver, err := s.DB.IncrGlobalVersion(ctx, tx)
			if err != nil {
				return err
			}
			if s.Hub != nil {
				s.Hub.Broadcast("graph:invalidated", map[string]any{"version": ver})
			}
			return nil
		})
	default:
		return fmt.Errorf("%w: unknown job kind %s", httpx.ErrValidation, job.Kind)
	}
}

func mergeTagPaths(manual, parsed []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, src := range [][]string{manual, parsed} {
		for _, p := range src {
			p = strings.ToLower(strings.Trim(strings.TrimSpace(p), "/"))
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

func leafName(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

type Rebuilder struct {
	DB    *store.DB
	Cards *CardService
	Graph *GraphService
}

func (r *Rebuilder) All(ctx context.Context) (int, error) {
	cards, err := r.DB.ListAllAliveBodies(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range cards {
		if err := r.one(ctx, c); err != nil {
			logger.L().Error("rebuild card", slog.String("id", c.ID.String()), slog.String("err", err.Error()))
			return n, err
		}
		n++
	}
	if r.Graph != nil {
		r.Graph.Invalidate()
	}
	_ = r.DB.WithTx(ctx, func(tx pgx.Tx) error {
		_, err := r.DB.IncrGlobalVersion(ctx, tx)
		return err
	})
	return n, nil
}

func (r *Rebuilder) one(ctx context.Context, c model.Card) error {
	parsed := engine.Parse(c.Body)
	return r.DB.WithTx(ctx, func(tx pgx.Tx) error {
		old, err := r.DB.ListOutgoing(ctx, tx, c.ID)
		if err != nil {
			return err
		}
		if r.Cards != nil {
			if err := r.Cards.writeOutgoing(ctx, tx, c.ID, old, parsed.Links); err != nil {
				return err
			}
		}
		_, err = r.DB.ReplaceCardTags(ctx, tx, c.ID, parsed.Tags)
		return err
	})
}
