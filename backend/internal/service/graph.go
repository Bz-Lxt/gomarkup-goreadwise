package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"goreadwise/internal/model"
	"goreadwise/internal/store"
)

type GraphService struct {
	DB *store.DB

	mu       sync.RWMutex
	cached   model.Graph
	cachedAt time.Time
	version  int64
}

func (s *GraphService) Full(ctx context.Context) (model.Graph, error) {
	ver, err := s.DB.GlobalVersion(ctx)
	if err != nil {
		ver = 0
	}
	s.mu.RLock()
	if s.version == ver && ver > 0 && time.Since(s.cachedAt) < 10*time.Second {
		g := s.cached
		s.mu.RUnlock()
		return g, nil
	}
	s.mu.RUnlock()

	g, err := s.DB.LoadGraph(ctx)
	if err != nil {
		return g, err
	}
	s.mu.Lock()
	s.cached = g
	s.version = g.Version
	s.cachedAt = time.Now()
	s.mu.Unlock()
	return g, nil
}

func (s *GraphService) Subgraph(ctx context.Context, root uuid.UUID, depth int) (model.Graph, error) {
	return s.DB.LoadSubgraph(ctx, root, depth)
}

func (s *GraphService) SavePositions(ctx context.Context, updates []model.PositionUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	if len(updates) > 2000 {
		updates = updates[:2000]
	}
	if err := s.DB.UpdatePositions(ctx, updates); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *GraphService) Invalidate() {
	s.mu.Lock()
	s.version = -1
	s.mu.Unlock()
}

func (s *GraphService) Metrics(ctx context.Context) (model.MetricsSnapshot, error) {
	cards, _ := s.DB.CountAliveCards(ctx)
	edges, _ := s.DB.CountEdges(ctx)
	ver, _ := s.DB.GlobalVersion(ctx)
	return model.MetricsSnapshot{
		GraphVersion: ver,
		CardCount:    cards,
		EdgeCount:    edges,
	}, nil
}
