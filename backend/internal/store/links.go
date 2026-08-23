package store

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/clock"
	"goreadwise/internal/engine"
	"goreadwise/internal/model"
)

func scanLink(rows pgx.Row) (model.Link, error) {
	var l model.Link
	err := rows.Scan(
		&l.ID, &l.SourceCardID, &l.TargetCardID, &l.TargetTitle, &l.DisplayText,
		&l.OffsetStart, &l.OffsetEnd, &l.Excerpt, &l.CreatedAt, &l.SourceTitle,
	)
	l.Dangling = l.TargetCardID == nil
	return l, err
}

func (d *DB) ListOutgoing(ctx context.Context, tx pgx.Tx, source uuid.UUID) ([]model.Link, error) {
	q := `
		SELECT l.id, l.source_card_id, l.target_card_id, l.target_title, l.display_text,
		       l.offset_start, l.offset_end, l.excerpt, l.created_at, COALESCE(s.title,'')
		FROM card_links l
		JOIN cards s ON s.id = l.source_card_id
		WHERE l.source_card_id=$1
		ORDER BY l.offset_start`
	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, q, source)
	} else {
		rows, err = d.Pool.Query(ctx, q, source)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectLinks(rows)
}

func (d *DB) ListBacklinks(ctx context.Context, cardID uuid.UUID, title string) ([]model.Link, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT l.id, l.source_card_id, l.target_card_id, l.target_title, l.display_text,
		       l.offset_start, l.offset_end, l.excerpt, l.created_at, COALESCE(s.title,'')
		FROM card_links l
		JOIN cards s ON s.id = l.source_card_id AND s.deleted_at IS NULL
		WHERE l.target_card_id=$1 OR (l.target_card_id IS NULL AND lower(l.target_title)=lower($2))
		ORDER BY s.updated_at DESC`, cardID, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectLinks(rows)
}

func (d *DB) InsertLink(ctx context.Context, tx pgx.Tx, source uuid.UUID, w engine.WikiLink, target *uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO card_links (source_card_id, target_card_id, target_title, display_text, offset_start, offset_end, excerpt, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		source, target, w.Target, w.Display, w.OffsetStart, w.OffsetEnd, w.Excerpt, clock.Now())
	return err
}

func (d *DB) DeleteLinksByIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `DELETE FROM card_links WHERE id = ANY($1)`, ids)
	return err
}

func (d *DB) DeleteOutgoing(ctx context.Context, tx pgx.Tx, source uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM card_links WHERE source_card_id=$1`, source)
	return err
}

func (d *DB) ResolveDanglingByTitle(ctx context.Context, tx pgx.Tx, title string, target uuid.UUID) (int64, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE card_links SET target_card_id=$2
		WHERE target_card_id IS NULL AND lower(target_title)=lower($1)`, title, target)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (d *DB) ListLinksByTargetTitle(ctx context.Context, title string) ([]model.Link, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT l.id, l.source_card_id, l.target_card_id, l.target_title, l.display_text,
		       l.offset_start, l.offset_end, l.excerpt, l.created_at, COALESCE(s.title,'')
		FROM card_links l
		JOIN cards s ON s.id=l.source_card_id AND s.deleted_at IS NULL
		WHERE lower(l.target_title)=lower($1)`, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectLinks(rows)
}

func (d *DB) CountEdges(ctx context.Context) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM card_links l
		JOIN cards s ON s.id=l.source_card_id AND s.deleted_at IS NULL`).Scan(&n)
	return n, err
}

func collectLinks(rows pgx.Rows) ([]model.Link, error) {
	var out []model.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (d *DB) RewriteTargetTitles(ctx context.Context, tx pgx.Tx, oldTitle, newTitle string) error {
	_, err := tx.Exec(ctx, `
		UPDATE card_links SET target_title=$2
		WHERE lower(target_title)=lower($1)`, oldTitle, newTitle)
	return err
}

func JoinTitles(links []model.Link) string {
	parts := make([]string, 0, len(links))
	for _, l := range links {
		parts = append(parts, l.TargetTitle)
	}
	return strings.Join(parts, ",")
}
