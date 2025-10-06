package main

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	r "github.com/redis/go-redis/v9"
)

func main() {
	dsn := getenv("POSTGRES_DSN", "")
	db, _ := sql.Open("pgx", dsn)
	rdb := r.NewClient(&r.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	ctx := context.Background()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		// try advisory lock; if not leader, continue
		var ok bool
		_ = db.QueryRowContext(ctx, "select pg_try_advisory_lock(42)").Scan(&ok)
		if !ok {
			continue
		}

		// 1) move due delayed jobs per-tenant
		// 2) requeue expired leases
		// 3) fire schedules: where now() >= next_run_at -> enqueue + compute next

		// release automatically on connection close; or explicitly call pg_advisory_unlock
	}
}
