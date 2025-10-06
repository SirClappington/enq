package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	r "github.com/redis/go-redis/v9"
	"github.com/you/enq/internal/config"
	"github.com/you/enq/internal/queue"
	"github.com/you/enq/internal/storage"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	db, _ := pgxpool.New(ctx, cfg.PostgresDSN)
	rdb := r.NewClient(&r.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	store := storage.New(db)
	q := queue.New(rdb)

	rtr := chi.NewRouter()
	// middleware: auth, logging, recover (omitted)

	rtr.Post("/v1/jobs", func(w http.ResponseWriter, r *http.Request) { /* enqueue handler: validate, persist, push */ })
	rtr.Get("/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) { /* fetch job + events */ })
	rtr.Post("/v1/lease", func(w http.ResponseWriter, r *http.Request) { /* pop + mark leased w/ lease_expires */ })
	rtr.Post("/v1/lease/{id}/extend", func(w http.ResponseWriter, r *http.Request) { /* extend */ })
	rtr.Post("/v1/complete", func(w http.ResponseWriter, r *http.Request) { /* mark succeeded, emit event */ })
	rtr.Post("/v1/fail", func(w http.ResponseWriter, r *http.Request) { /* retry or DLQ */ })

	http.ListenAndServe(cfg.APIAddr, rtr)
}
