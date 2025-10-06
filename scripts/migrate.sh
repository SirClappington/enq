#!/usr/bin/env bash
set -euo pipefail
GOOSE_DRIVER=postgres \
GOOSE_DBSTRING="${POSTGRES_DSN:-postgres://enq:enq@localhost:5432/enq?sslmode=disable}" \
goose -dir ./db/migrations up