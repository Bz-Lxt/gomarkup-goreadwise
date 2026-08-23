package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/clock"
	"goreadwise/internal/engine"
	"goreadwise/internal/model"
)

func (d *DB) EnsureTagPath(ctx context.Context, tx pgx.Tx, path string) ([]model.Tag, error) {
	paths := engine.AncestorPaths(path)
	out := make([]model.Tag, 0, len(paths))
	var parent *uuid.UUID
	for _, p := range paths {
		tag, err := d.upsertTag(ctx, tx, p, parent)
		if err != nil {
			return nil, err
		}
		id := tag.ID
		parent = &id
		out = append(out, tag)
	}
	return out, nil
}

func (d *DB) upsertTag(ctx context.Context, tx pgx.Tx, fullPath string, parent *uuid.UUID) (model.Tag, error) {
	var t model.Tag
	err := tx.QueryRow(ctx, `SELECT id, name, full_path, parent_id, created_at FROM tags WHERE full_path=$1`, fullPath).
		Scan(&t.ID, &t.Name, &t.FullPath, &t.ParentID, &t.CreatedAt)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Tag{}, err
	}
	name := fullPath
	if i := lastSlash(fullPath); i >= 0 {
		name = fullPath[i+1:]
	}
	now := clock.Now()
	err = tx.QueryRow(ctx, `
		INSERT INTO tags (name, full_path, parent_id, created_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (full_path) DO UPDATE SET name=EXCLUDED.name
		RETURNING id, name, full_path, parent_id, created_at`, name, fullPath, parent, now).
		Scan(&t.ID, &t.Name, &t.FullPath, &t.ParentID, &t.CreatedAt)
	return t, err
}

func (d *DB) ReplaceCardTags(ctx context.Context, tx pgx.Tx, cardID uuid.UUID, paths []string) ([]model.Tag, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM card_tags WHERE card_id=$1`, cardID); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var tags []model.Tag
	for _, p := range paths {
		chain, err := d.EnsureTagPath(ctx, tx, p)
		if err != nil {
			return nil, err
		}
		if len(chain) == 0 {
			continue
		}
		leaf := chain[len(chain)-1]
		if _, ok := seen[leaf.FullPath]; ok {
			continue
		}
		seen[leaf.FullPath] = struct{}{}
		if _, err := tx.Exec(ctx, `INSERT INTO card_tags (card_id, tag_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, cardID, leaf.ID); err != nil {
			return nil, err
		}
		tags = append(tags, leaf)
	}
	return tags, nil
}

func (d *DB) ListCardTags(ctx context.Context, cardID uuid.UUID) ([]model.Tag, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT t.id, t.name, t.full_path, t.parent_id, t.created_at
		FROM tags t JOIN card_tags ct ON ct.tag_id=t.id
		WHERE ct.card_id=$1 ORDER BY t.full_path`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTags(rows)
}

func (d *DB) ListTagTree(ctx context.Context) ([]model.Tag, error) {
	rows, err := d.Pool.Query(ctx, `SELECT id, name, full_path, parent_id, created_at FROM tags ORDER BY full_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectTags(rows)
}

func (d *DB) ListCardTagPaths(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string)
	if len(cardIDs) == 0 {
		return out, nil
	}
	rows, err := d.Pool.Query(ctx, `
		SELECT ct.card_id, t.full_path
		FROM card_tags ct JOIN tags t ON t.id=ct.tag_id
		WHERE ct.card_id = ANY($1)
		ORDER BY t.full_path`, cardIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		out[id] = append(out[id], path)
	}
	return out, rows.Err()
}

func collectTags(rows pgx.Rows) ([]model.Tag, error) {
	var out []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.FullPath, &t.ParentID, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
