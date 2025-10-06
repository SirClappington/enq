package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	r "github.com/redis/go-redis/v9"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	dsn := getenv("POSTGRES_DSN", "postgres://enq:enq@localhost:5432/enq?sslmode=disable")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_ = r.NewClient(&r.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	ctx := context.Background()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var ok bool
		if err := db.QueryRowContext(ctx, "select pg_try_advisory_lock(42)").Scan(&ok); err != nil {
			log.Println("lock error:", err)
			continue
		}
		if !ok {
			continue
		}

		// TODO: move delayed jobs, requeue expired leases, fire schedules
	}
}
