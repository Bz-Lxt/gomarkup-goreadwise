package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/clock"
	"goreadwise/internal/model"
)

func (d *DB) EnqueueJob(ctx context.Context, tx pgx.Tx, cardID uuid.UUID, hash, kind string) (model.GraphJob, bool, error) {
	now := clock.Now()
	q := `
		INSERT INTO graph_jobs (card_id, content_hash, kind, status, attempts, created_at, updated_at)
		VALUES ($1,$2,$3,'pending',0,$4,$4)
		ON CONFLICT (card_id, content_hash, kind)
		DO UPDATE SET updated_at=EXCLUDED.updated_at
		RETURNING id, card_id, content_hash, kind, status, attempts, last_error, created_at, updated_at,
		          (xmax = 0) AS inserted`
	var job model.GraphJob
	var inserted bool
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, q, cardID, hash, kind, now).Scan(
			&job.ID, &job.CardID, &job.ContentHash, &job.Kind, &job.Status, &job.Attempts,
			&job.LastError, &job.CreatedAt, &job.UpdatedAt, &inserted)
	} else {
		err = d.Pool.QueryRow(ctx, q, cardID, hash, kind, now).Scan(
			&job.ID, &job.CardID, &job.ContentHash, &job.Kind, &job.Status, &job.Attempts,
			&job.LastError, &job.CreatedAt, &job.UpdatedAt, &inserted)
	}
	return job, inserted, err
}

func (d *DB) MarkJob(ctx context.Context, id uuid.UUID, status, lastErr string) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE graph_jobs
		SET status=$2, last_error=NULLIF($3,''), attempts=attempts+1, updated_at=$4
		WHERE id=$1`, id, status, lastErr, clock.Now())
	return err
}

func (d *DB) RecoverPendingJobs(ctx context.Context) ([]model.GraphJob, error) {
	_, _ = d.Pool.Exec(ctx, `
		UPDATE graph_jobs SET status='pending', updated_at=$1
		WHERE status='running'`, clock.Now())
	rows, err := d.Pool.Query(ctx, `
		SELECT id, card_id, content_hash, kind, status, attempts, last_error, created_at, updated_at
		FROM graph_jobs WHERE status='pending' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GraphJob
	for rows.Next() {
		var j model.GraphJob
		if err := rows.Scan(&j.ID, &j.CardID, &j.ContentHash, &j.Kind, &j.Status, &j.Attempts, &j.LastError, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (d *DB) InsertOpLog(ctx context.Context, tx pgx.Tx, kind string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO op_logs (kind, payload, created_at) VALUES ($1,$2,$3)`, kind, raw, clock.Now())
	return err
}

func (d *DB) LatestOpLogs(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Pool.Query(ctx, `SELECT kind, payload, created_at FROM op_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var kind string
		var payload []byte
		var created any
		if err := rows.Scan(&kind, &payload, &created); err != nil {
			return nil, err
		}
		var obj any
		_ = json.Unmarshal(payload, &obj)
		out = append(out, map[string]any{"kind": kind, "payload": obj, "created_at": created})
	}
	return out, rows.Err()
}

func IgnoreNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}
