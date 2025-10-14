package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"

	"github.com/SirClappington/enq/internal/config"
	"github.com/SirClappington/enq/internal/queue"
	"github.com/SirClappington/enq/internal/storage"
)

var (
	version = "dev"
	commit  = "none"
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

type LeaseReq struct {
	WorkerID     string   `json:"workerId"`
	Capabilities []string `json:"capabilities"` // unused in MVP; later for typed queues
	MaxBatch     int      `json:"maxBatch"`
}
type LeasedJob struct {
	ID                   string          `json:"id"`
	Type                 string          `json:"type"`
	Payload              json.RawMessage `json:"payload"`
	Attempt              int             `json:"attempt"`
	MaxAttempts          int             `json:"maxAttempts"`
	LeaseExpiresAt       time.Time       `json:"leaseExpiresAt"`
	VisibilityTimeoutSec int             `json:"visibilityTimeoutSec"`
}
type LeaseResp struct {
	Job *LeasedJob `json:"job"`
}
type CompleteReq struct {
	WorkerID string `json:"workerId"`
	JobID    string `json:"jobId"`
}
type FailReq struct {
	WorkerID  string `json:"workerId"`
	JobID     string `json:"jobId"`
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
}

// --- Auth & tenant helpers ---

type ctxKey int

const tenantKey ctxKey = iota

func setTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

func getTenant(ctx context.Context) (string, bool) {
	v := ctx.Value(tenantKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	const p = "Bearer "
	if len(h) <= len(p) || h[:len(p)] != p {
		return "", false
	}
	return h[len(p):], true
}

// tenantForKey looks up the tenant for a given API key.
// MVP behavior:
//  1. If key == "dev-key", return "demo" (no DB hit).
//  2. Otherwise, try DB: select id from tenants where api_key_hash = $1
//     (For now we treat api_key_hash as plaintext. You can switch to SHA-256 later.)
func tenantForKey(ctx context.Context, db *pgxpool.Pool, apiKey string) (string, bool) {
	if apiKey == "dev-key" {
		return "demo", true
	}
	var tenantID string
	err := db.QueryRow(ctx, `select id from tenants where api_key_hash = $1`, apiKey).Scan(&tenantID)
	if err != nil {
		return "", false
	}
	return tenantID, true
}

// RFC 6750-ish 401 writer
func writeUnauthorized(w http.ResponseWriter, errCode, desc string) {
	// errCode: "invalid_request" | "invalid_token" | "insufficient_scope" | ""
	// desc: short human message (optional)
	h := `Bearer realm="enq", charset="UTF-8"`
	if errCode != "" {
		h += `, error="` + errCode + `"`
	}
	if desc != "" {
		h += `, error_description="` + desc + `"`
	}
	w.Header().Set("WWW-Authenticate", h)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func main() {
	cfg := config.Load()
	log.Printf("Enq API starting — version=%s commit=%s", version, commit)
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})

	store := storage.New(db)
	q := queue.New(rdb)

	rtr := chi.NewRouter()

	rtr.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Require auth for API routes; allow /health and OPTIONS
	rtr.Group(func(protected chi.Router) {
		// attach auth middleware
		protected.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Allow CORS preflight to pass through
				if r.Method == http.MethodOptions {
					next.ServeHTTP(w, r)
					return
				}
				// Health check is outside this group, so all here are protected
				key, ok := bearerToken(r)
				if !ok {
					writeUnauthorized(w, "invalid_request", "missing or malformed Authorization header")
					return
				}
				tenantID, ok := tenantForKey(r.Context(), db, key)
				if !ok {
					writeUnauthorized(w, "invalid_token", "invalid API key")
					return
				}
				ctx := setTenant(r.Context(), tenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		// All /v1/* routes go here (move your existing routes into this block)
		protected.Post("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := getTenant(r.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// push to Redis; if it fails, mark failed_perm (visible in UI)
			if err := q.Enqueue(r.Context(), tenantID, id, runAt); err != nil {
				_, _ = db.Exec(r.Context(),
					`update jobs set status='failed_perm', error=$2, updated_at=now() where id=$1`,
					id, "enqueue push to redis failed: "+err.Error())
				http.Error(w, "enqueue failed", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "queued"})
		})

		protected.Get("/v1/jobs", func(w http.ResponseWriter, req *http.Request) {
			tenantID, ok := getTenant(req.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

			type row struct {
				ID      string          `json:"id"`
				Type    string          `json:"type"`
				Status  string          `json:"status"`
				Attempt int             `json:"attempt"`
				RunAt   time.Time       `json:"run_at"`
				Payload json.RawMessage `json:"payload"`
			}
			rows, err := db.Query(req.Context(),
				`select id, type, status::text, attempt, run_at, payload
			   from jobs
			  where tenant_id = $1
			  order by created_at desc
			  limit 50`, tenantID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var out []row
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.ID, &r.Type, &r.Status, &r.Attempt, &r.RunAt, &r.Payload); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				out = append(out, r)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jobs": out})
		})

		protected.Post("/v1/lease", func(w http.ResponseWriter, req *http.Request) {
			tenantID, ok := getTenant(req.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

			var body LeaseReq
			_ = json.NewDecoder(req.Body).Decode(&body)
			if body.WorkerID == "" {
				body.WorkerID = "dev-worker"
			}

			// Pop one job id
			res, err := rdb.BRPop(req.Context(), 1*time.Second, "queue:"+tenantID).Result()
			if err != nil && err != redis.Nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if len(res) < 2 {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(LeaseResp{Job: nil})
				return
			}
			jobID := res[1]

			tx, txErr := db.Begin(req.Context())
			if txErr != nil {
				http.Error(w, txErr.Error(), http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(req.Context())

			var typ string
			var payload []byte
			var attempt, maxAttempts, vt int
			row := tx.QueryRow(req.Context(),
				`select type, payload, attempt, max_attempts, visibility_timeout_sec
			   from jobs
			  where id=$1 and tenant_id=$2 and status='queued'
			  for update`, jobID, tenantID)
			if err := row.Scan(&typ, &payload, &attempt, &maxAttempts, &vt); err != nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(LeaseResp{Job: nil})
				return
			}

			leaseExpires := time.Now().UTC().Add(time.Duration(vt) * time.Second)
			if _, err := tx.Exec(req.Context(),
				`update jobs
			    set status='leased', leased_by=$2, lease_expires_at=$3, updated_at=now()
			  where id=$1`, jobID, body.WorkerID, leaseExpires); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := tx.Commit(req.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			lj := LeasedJob{
				ID: jobID, Type: typ, Payload: payload,
				Attempt: attempt, MaxAttempts: maxAttempts,
				LeaseExpiresAt: leaseExpires, VisibilityTimeoutSec: vt,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(LeaseResp{Job: &lj})
		})

		protected.Post("/v1/lease/{id}/extend", func(w http.ResponseWriter, req *http.Request) {
			tenantID, ok := getTenant(req.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

			type ExtendReq struct {
				WorkerID    string `json:"workerId"`
				ExtendBySec int    `json:"extendBySec"`
			}
			var body ExtendReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			jobID := chi.URLParam(req, "id")
			if body.ExtendBySec <= 0 {
				body.ExtendBySec = 60
			}

			_, err := db.Exec(req.Context(),
				`update jobs
			    set lease_expires_at = now() + ($2 || ' seconds')::interval,
			        updated_at = now()
			  where id = $1 and tenant_id=$3 and status='leased'`,
				jobID, body.ExtendBySec, tenantID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		protected.Post("/v1/complete", func(w http.ResponseWriter, req *http.Request) {
			tenantID, ok := getTenant(req.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

			var body CompleteReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_, err := db.Exec(req.Context(),
				`update jobs
			    set status='succeeded', updated_at=now()
			  where id=$1 and tenant_id=$2 and status in ('leased','failed_temp')`,
				body.JobID, tenantID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		protected.Post("/v1/fail", func(w http.ResponseWriter, req *http.Request) {
			tenantID, ok := getTenant(req.Context())
			if !ok {
				writeUnauthorized(w, "Unauthorized", "Unauthorized Tenant")
				return
			}

			var body FailReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var attempt, maxAttempts int
			var backoff string
			if err := db.QueryRow(req.Context(),
				`select attempt, max_attempts, backoff_policy
			   from jobs where id=$1 and tenant_id=$2`,
				body.JobID, tenantID).Scan(&attempt, &maxAttempts, &backoff); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if body.Retryable && attempt+1 < maxAttempts {
				base := 30 * time.Second
				next := time.Now().UTC().Add(base * (1 << attempt))

				_, err := db.Exec(req.Context(),
					`update jobs
				    set attempt=attempt+1, status='failed_temp', error=$2, run_at=$3, updated_at=now()
				  where id=$1 and tenant_id=$4`,
					body.JobID, body.Error, next, tenantID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := rdb.ZAdd(req.Context(), "delay:"+tenantID,
					redis.Z{Score: float64(next.Unix()), Member: body.JobID}).Err(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				_, err := db.Exec(req.Context(),
					`update jobs
				    set status='failed_perm', error=$2, updated_at=now()
				  where id=$1 and tenant_id=$3`,
					body.JobID, body.Error, tenantID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})

	rtr.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	// Start HTTP server with Graceful Shutdown
	srv := &http.Server{
		Addr:         cfg.APIAddr,
		Handler:      rtr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxTimeout)
}
