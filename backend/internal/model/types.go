package model

import (
	"time"

	"github.com/google/uuid"
)

type Card struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	TitleNorm    string     `json:"-"`
	Body         string     `json:"body"`
	ContentHash  string     `json:"content_hash"`
	SourceURL    *string    `json:"source_url,omitempty"`
	SourceSite   *string    `json:"source_site,omitempty"`
	ClippedAt    *time.Time `json:"clipped_at,omitempty"`
	PosX         *float64   `json:"pos_x,omitempty"`
	PosY         *float64   `json:"pos_y,omitempty"`
	GraphVersion int64      `json:"graph_version"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Tags         []Tag      `json:"tags"`
	OutLinks     []Link     `json:"out_links"`
	BackLinks    []Link     `json:"back_links"`
}

type Link struct {
	ID           uuid.UUID  `json:"id"`
	SourceCardID uuid.UUID  `json:"source_card_id"`
	TargetCardID *uuid.UUID `json:"target_card_id,omitempty"`
	TargetTitle  string     `json:"target_title"`
	DisplayText  string     `json:"display_text"`
	OffsetStart  int        `json:"offset_start"`
	OffsetEnd    int        `json:"offset_end"`
	Excerpt      string     `json:"excerpt"`
	SourceTitle  string     `json:"source_title,omitempty"`
	Dangling     bool       `json:"dangling"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Tag struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	FullPath  string     `json:"full_path"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CardListFilter struct {
	Query    string
	TagPath  string
	Page     int
	PageSize int
}

type GraphNode struct {
	ID       string   `json:"id"`
	CardID   *string  `json:"card_id,omitempty"`
	Title    string   `json:"title"`
	Degree   int      `json:"degree"`
	Dangling bool     `json:"dangling"`
	Orphan   bool     `json:"orphan"`
	Tags     []string `json:"tags"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
}

type GraphEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Dangling bool   `json:"dangling"`
}

type Graph struct {
	Version int64       `json:"version"`
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
}

type GraphJob struct {
	ID          uuid.UUID
	CardID      uuid.UUID
	ContentHash string
	Kind        string
	Status      string
	Attempts    int
	LastError   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	JobKindSnapshot = "rebuild_snapshot"
	JobKindResolve  = "resolve_dangling"
	JobKindRename   = "rename_cascade"
	JobKindStats    = "stats"

	JobPending = "pending"
	JobRunning = "running"
	JobDone    = "done"
	JobFailed  = "failed"
	JobSkipped = "skipped"
)

type CreateCardInput struct {
	Title      string
	Body       string
	Tags       []string
	SourceURL  *string
	SourceSite *string
	ClippedAt  *time.Time
}

type UpdateCardInput struct {
	Title *string
	Body  *string
	Tags  []string
}

type PositionUpdate struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type MetricsSnapshot struct {
	QueueDepth   int   `json:"queue_depth"`
	QueueCap     int   `json:"queue_cap"`
	SyncFallback int64 `json:"sync_fallback"`
	JobsDone     int64 `json:"jobs_done"`
	JobsFailed   int64 `json:"jobs_failed"`
	GraphVersion int64 `json:"graph_version"`
	CardCount    int   `json:"card_count"`
	EdgeCount    int   `json:"edge_count"`
}
