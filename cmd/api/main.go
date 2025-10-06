package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	r "github.com/redis/go-redis/v9"

	"github.com/SirClappington/enq/internal/config"
	"github.com/SirClappington/enq/internal/queue"
	"github.com/SirClappington/enq/internal/storage"
)

type EnqueueReq struct {
	Type                 string          `json:"type"`
	Payload              json.RawMessage `json:"payload"`
	RunAt                *time.Time      `json:"runAt"`
	Priority             *int            `json:"priority"`
	DedupeKey            *string         `json:"dedupeKey"`
	DedupeTtlSec         *int            `json:"dedupeTtlSec"`
	MaxAttempts          *int            `json:"maxAttempts"`
	BackoffPolicy        *string         `json:"backoffPolicy"`
	VisibilityTimeoutSec *int            `json:"visibilityTimeoutSec"`
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rdb := r.NewClient(&r.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})

	store := storage.New(db)
	q := queue.New(rdb)

	rtr := chi.NewRouter()

	// MVP tenant
	const tenantID = "demo"

	rtr.Post("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		var body EnqueueReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		runAt := now
		if body.RunAt != nil {
			runAt = *body.RunAt
		}
		priority := 100
		if body.Priority != nil {
			priority = *body.Priority
		}
		maxAttempts := 10
		if body.MaxAttempts != nil {
			maxAttempts = *body.MaxAttempts
		}
		backoff := "exponential"
		if body.BackoffPolicy != nil {
			backoff = *body.BackoffPolicy
		}
		vt := cfg.DefaultVisibilityTOSec
		if body.VisibilityTimeoutSec != nil {
			vt = *body.VisibilityTimeoutSec
		}

		id, err := store.InsertJob(r.Context(), &storage.InsertJobParams{
			TenantID: tenantID, Type: body.Type, Payload: body.Payload,
			Priority: priority, RunAt: runAt, DedupeKey: body.DedupeKey,
			DedupeTTL: body.DedupeTtlSec, MaxAttempts: maxAttempts,
			BackoffPolicy: backoff, VisibilityTimeoutSec: vt,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := q.Enqueue(r.Context(), tenantID, id, runAt); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "queued"})
	})

	http.ListenAndServe(cfg.APIAddr, rtr)
}
