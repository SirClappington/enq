package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Store { return &Store{db} }

// InsertJob persists job metadata (source of truth)
func (s *Store) InsertJob(ctx context.Context, j *InsertJobParams) (string, error) {
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `insert into jobs(
id, tenant_id, type, payload, priority, run_at, dedupe_key, dedupe_ttl_sec,
attempt, max_attempts, backoff_policy, visibility_timeout_sec, status
) values ($1,$2,$3,$4,$5,$6,$7,$8,0,$9,$10,$11,'queued')`,
		id, j.TenantID, j.Type, j.Payload, j.Priority, j.RunAt, j.DedupeKey, j.DedupeTTL,
		j.MaxAttempts, j.BackoffPolicy, j.VisibilityTimeoutSec,
	)
	return id, err
}

type InsertJobParams struct {
	TenantID, Type       string
	Payload              []byte
	Priority             int
	RunAt                time.Time
	DedupeKey            *string
	DedupeTTL            *int
	MaxAttempts          int
	BackoffPolicy        string
	VisibilityTimeoutSec int
}
