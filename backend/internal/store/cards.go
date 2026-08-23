package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/clock"
	"goreadwise/internal/engine"
	"goreadwise/internal/httpx"
	"goreadwise/internal/model"
)

func scanCard(row pgx.Row) (model.Card, error) {
	var c model.Card
	err := row.Scan(
		&c.ID, &c.Title, &c.TitleNorm, &c.Body, &c.ContentHash,
		&c.SourceURL, &c.SourceSite, &c.ClippedAt, &c.PosX, &c.PosY,
		&c.GraphVersion, &c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

const cardCols = `id, title, title_norm, body, content_hash, source_url, source_site, clipped_at, pos_x, pos_y, graph_version, deleted_at, created_at, updated_at`

func (d *DB) InsertCard(ctx context.Context, tx pgx.Tx, in model.CreateCardInput, hash string) (model.Card, error) {
	now := clock.Now()
	norm := engine.NormalizeTitle(in.Title)
	row := tx.QueryRow(ctx, `
		INSERT INTO cards (title, title_norm, body, content_hash, source_url, source_site, clipped_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)
		RETURNING `+cardCols, in.Title, norm, in.Body, hash, in.SourceURL, in.SourceSite, in.ClippedAt, now)
	c, err := scanCard(row)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Card{}, fmt.Errorf("%w: title already exists", httpx.ErrConflict)
		}
		return model.Card{}, err
	}
	return c, nil
}

func (d *DB) UpdateCard(ctx context.Context, tx pgx.Tx, id uuid.UUID, title, body, hash string, bump bool) (model.Card, error) {
	now := clock.Now()
	norm := engine.NormalizeTitle(title)
	q := `
		UPDATE cards
		SET title=$2, title_norm=$3, body=$4, content_hash=$5, updated_at=$6`
	if bump {
		q += `, graph_version = graph_version + 1`
	}
	q += ` WHERE id=$1 AND deleted_at IS NULL RETURNING ` + cardCols
	c, err := scanCard(tx.QueryRow(ctx, q, id, title, norm, body, hash, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Card{}, fmt.Errorf("%w: card not found", httpx.ErrNotFound)
	}
	if isUniqueViolation(err) {
		return model.Card{}, fmt.Errorf("%w: title already exists", httpx.ErrConflict)
	}
	return c, err
}

func (d *DB) SoftDeleteCard(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	now := clock.Now()
	tag, err := tx.Exec(ctx, `UPDATE cards SET deleted_at=$2, updated_at=$2 WHERE id=$1 AND deleted_at IS NULL`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: card not found", httpx.ErrNotFound)
	}
	_, err = tx.Exec(ctx, `UPDATE card_links SET target_card_id=NULL WHERE target_card_id=$1`, id)
	return err
}

func (d *DB) GetCard(ctx context.Context, id uuid.UUID) (model.Card, error) {
	c, err := scanCard(d.Pool.QueryRow(ctx, `SELECT `+cardCols+` FROM cards WHERE id=$1 AND deleted_at IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Card{}, fmt.Errorf("%w: card not found", httpx.ErrNotFound)
	}
	return c, err
}

func (d *DB) GetCardByTitleNorm(ctx context.Context, tx pgx.Tx, norm string) (model.Card, error) {
	q := `SELECT ` + cardCols + ` FROM cards WHERE title_norm=$1 AND deleted_at IS NULL`
	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, q, norm)
	} else {
		row = d.Pool.QueryRow(ctx, q, norm)
	}
	c, err := scanCard(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Card{}, fmt.Errorf("%w: card not found", httpx.ErrNotFound)
	}
	return c, err
}

func (d *DB) ListCards(ctx context.Context, f model.CardListFilter) ([]model.Card, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 30
	}
	where := []string{"c.deleted_at IS NULL"}
	args := []any{}
	arg := 1
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, fmt.Sprintf("(c.title ILIKE $%d OR c.body ILIKE $%d)", arg, arg))
		args = append(args, "%"+q+"%")
		arg++
	}
	if p := strings.TrimSpace(f.TagPath); p != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM card_tags ct JOIN tags t ON t.id=ct.tag_id
			WHERE ct.card_id=c.id AND (t.full_path = $%d OR t.full_path LIKE $%d)
		)`, arg, arg+1))
		args = append(args, strings.ToLower(p), strings.ToLower(p)+"/%")
		arg += 2
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards c WHERE `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	rows, err := d.Pool.Query(ctx, `SELECT `+qualifyCardCols("c")+` FROM cards c WHERE `+w+`
		ORDER BY c.updated_at DESC LIMIT $`+itoa(arg)+` OFFSET $`+itoa(arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (d *DB) SuggestTitles(ctx context.Context, q string, limit int) ([]model.Card, error) {
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	q = strings.TrimSpace(q)
	rows, err := d.Pool.Query(ctx, `
		SELECT `+cardCols+` FROM cards
		WHERE deleted_at IS NULL AND title ILIKE $1
		ORDER BY updated_at DESC LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) CountAliveCards(ctx context.Context) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (d *DB) UpdatePositions(ctx context.Context, updates []model.PositionUpdate) error {
	return d.WithTx(ctx, func(tx pgx.Tx) error {
		for _, u := range updates {
			id, err := uuid.Parse(u.ID)
			if err != nil {
				continue
			}
			if _, err := tx.Exec(ctx, `UPDATE cards SET pos_x=$2, pos_y=$3, updated_at=$4 WHERE id=$1 AND deleted_at IS NULL`,
				id, u.X, u.Y, clock.Now()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) BumpGraphVersion(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE cards SET graph_version = graph_version + 1 WHERE id=$1`, id)
	return err
}

func (d *DB) ListAllAliveBodies(ctx context.Context) ([]model.Card, error) {
	rows, err := d.Pool.Query(ctx, `SELECT `+cardCols+` FROM cards WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Card
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func qualifyCardCols(alias string) string {
	parts := strings.Split(cardCols, ", ")
	for i, p := range parts {
		parts[i] = alias + "." + p
	}
	return strings.Join(parts, ", ")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505") || strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
