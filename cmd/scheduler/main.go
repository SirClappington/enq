package main

import (
	"context"
	"database/sql"
	"fmt"
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

	rdb := r.NewClient(&r.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	ctx := context.Background()

	tick := time.NewTicker(1000 * time.Millisecond)
	defer tick.Stop()

	for range tick.C {
		// leader election
		var ok bool
		if err := db.QueryRowContext(ctx, "select pg_try_advisory_lock(42)").Scan(&ok); err != nil {
			log.Println("lock error:", err)
			continue
		}
		if !ok {
			continue
		}

		// 1) list tenants (small table; cache later)
		tenants, err := fetchTenants(ctx, db)
		if err != nil {
			log.Println("tenants error:", err)
			continue
		}
		now := time.Now().UTC().Unix()

		// 2) for each tenant: move due delayed jobs from ZSET -> queue
		for _, t := range tenants {
			_ = moveDue(ctx, rdb, t, now, 200)
			if err := reconcileQueued(ctx, db, rdb, t, 500); err != nil {
				log.Printf("reconcile(%s): %v\n", t, err)
			}
		}

		// 3) requeue expired leases (DB authoritative)
		if err := requeueExpiredLeases(ctx, db, rdb, tenants, 500); err != nil {
			log.Println("requeueExpired:", err)
		}

		// (Optional) cron schedules would go here: read schedules.next_run_at <= now, enqueue, compute next.
	}
}

func fetchTenants(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `select id from tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		// MVP: ensure a default exists if table is empty (dev convenience)
		out = []string{"demo"}
	}
	return out, nil
}

func moveDue(ctx context.Context, rdb *r.Client, tenant string, now int64, batch int64) error {
	ids, err := rdb.ZRangeByScore(ctx, "delay:"+tenant, &r.ZRangeBy{
		Min: "-inf", Max: fmt.Sprintf("%d", now), Offset: 0, Count: batch,
	}).Result()
	if err != nil || len(ids) == 0 {
		return err
	}

	pipe := rdb.TxPipeline()
	for _, id := range ids {
		pipe.LPush(ctx, "queue:"+tenant, id)
		pipe.ZRem(ctx, "delay:"+tenant, id)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func requeueExpiredLeases(ctx context.Context, db *sql.DB, rdb *r.Client, tenants []string, batch int) error {
	// scan per-tenant to keep it simple; in practice you could scan once
	for _, t := range tenants {
		rows, err := db.QueryContext(ctx,
			`select id from jobs
			   where tenant_id = $1
			     and status = 'leased'
			     and lease_expires_at is not null
			     and lease_expires_at < now()
			   limit $2`, t, batch)
		if err != nil {
			return err
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if len(ids) == 0 {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx,
				`update jobs
				    set status = 'queued',
				        leased_by = null,
				        lease_expires_at = null,
				        updated_at = now()
				  where id = $1`, id); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		pipe := rdb.TxPipeline()
		for _, id := range ids {
			pipe.LPush(ctx, "queue:"+t, id)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func reconcileQueued(ctx context.Context, db *sql.DB, rdb *r.Client, tenant string, batch int) error {
	rows, err := db.QueryContext(ctx, `
    select id from jobs
     where tenant_id = $1 and status = 'queued' and run_at <= now()
     order by created_at asc limit $2`, tenant, batch)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	pipe := rdb.TxPipeline()
	for _, id := range ids {
		pipe.LPush(ctx, "queue:"+tenant, id)
	}
	_, err = pipe.Exec(ctx)
	return err
}
