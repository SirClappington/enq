Contributing to Enq

Thanks for your interest in improving Enq—a simple, production-minded job scheduler without Redis. Contributions of all kinds are welcome: code, docs, tests, examples, and feedback.

Table of contents

Code of Conduct

Ways to contribute

Development quickstart

Running with Docker

Testing & quality

Commit style

Pull requests

Issue reports

Project layout

Release notes & tagging

Security

Community & support

Code of Conduct

By participating, you agree to uphold our Code of Conduct
. Please report unacceptable behavior as described there.

Ways to contribute

Bug reports & fixes – Reproducible reports and minimal fixes are gold.

Docs – Improve README, architecture diagrams, and examples.

Tests – Unit/integration tests for API & scheduler semantics (leases, retries).

Features – Observability (metrics/logging), health checks, more DBs, clients.

Examples – Docker Compose, k8s manifests/Helm, Fly.io/Render recipes.

Before starting a large feature, open a discussion/issue to align on scope.

Development quickstart
Prereqs

Go ≥ 1.22

Make (optional but nice)

Docker (optional for Postgres)

A Postgres instance (local or container)

Repo setup
git clone https://github.com/sirclappington/enq.git
cd enq
go mod download

Start Postgres locally (via Docker)
docker run -d --name enq-pg -p 5432:5432 \
  -e POSTGRES_DB=enq -e POSTGRES_USER=user -e POSTGRES_PASSWORD=pass \
  postgres:16-alpine

Environment

Set a DB URL that the API & scheduler share:

export ENQ_DB_URL="postgres://user:pass@localhost:5432/enq?sslmode=disable"
export ENQ_POLL_INTERVAL_MS=1000   # how often workers poll (default sensible)

Run the API
go run ./cmd/api
# serves on :8080 (configure via env if you add it)

Run the Scheduler
go run ./cmd/scheduler

Smoke test
curl -X POST 'http://localhost:8080/v1/jobs' \
  -H 'Content-Type: application/json' \
  -d '{"type":"demo.echo","payload":{"msg":"hi"},"run_at":"now"}'

# Then GET by returned ID
curl 'http://localhost:8080/v1/jobs/<id>'

Running with Docker

We ship multi-arch images for api and scheduler.

Compose example (dev):

services:
  api:
    image: dclaywell/enq-api:latest
    ports: ["8080:8080"]
    environment:
      ENQ_DB_URL: postgres://user:pass@db:5432/enq?sslmode=disable
  scheduler:
    image: dclaywell/enq-scheduler:latest
    depends_on: [db]
    environment:
      ENQ_DB_URL: postgres://user:pass@db:5432/enq?sslmode=disable
      ENQ_POLL_INTERVAL_MS: "1000"
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: enq
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - enq_pg:/var/lib/postgresql/data
volumes:
  enq_pg:


Build locally from the repo (single root Dockerfile):

# API
docker build -t enq-api:dev \
  --build-arg COMPONENT=api --build-arg BUILD_PATH=./cmd/api -f Dockerfile .

# Scheduler
docker build -t enq-scheduler:dev \
  --build-arg COMPONENT=scheduler --build-arg BUILD_PATH=./cmd/scheduler -f Dockerfile .

Testing & quality
Unit tests
go test ./...

Recommended linters/tools

We aim to keep the codebase “boringly clean.” If you have these, please run them:

go vet ./...
golangci-lint run      # if installed; PRs welcome adding a config

Style

Use gofmt (or set your editor to format on save).

Keep public APIs small and explicit.

Prefer context-aware functions and return errors, not panics.

Add tests for concurrency/lease semantics when touching scheduler logic.

Commit style

We follow Conventional Commits to keep history tidy and automate changelogs:

Examples:

feat(api): add POST /v1/jobs delay parameter
fix(scheduler): prevent double-claim when lease expires mid-run
docs: add architecture SVG and quickstart
chore(ci): push multi-arch images to GHCR and Docker Hub
refactor: extract backoff with jitter into package
test: add integration test for retry limits


Scope suggestions: api, scheduler, db, docs, ci, examples, build.

If you prefer, add DCO sign-off:

Signed-off-by: Your Name <you@example.com>

Pull requests

Before you open a PR

 Tests pass: go test ./...

 Lint passes (if you have it locally)

 Add/adjust docs (README, examples) for user-visible changes

 Update error messages/logging as needed

PR template (paste into description):

## What
Short summary of the change.

## Why
Problem statement / motivation / linked issue.

## How
High-level approach, trade-offs, and risk.

## Testing
- [ ] Unit tests added/updated
- [ ] Manual smoke test steps included

## Screenshots/Logs (if relevant)

## Notes
Breaking changes? Migrations? Config/env additions?


We squash-merge by default. Keep PRs focused and small where possible.

Issue reports

Please include:

Version (git SHA or tag) and how you ran Enq (Go vs Docker)

Environment (OS/arch, Docker version, Postgres version)

Steps to reproduce (minimal)

Expected vs actual

Logs (redact secrets)

Config (ENQ_DB_URL, poll interval, relevant env vars)

Use labels when you can: bug, enhancement, docs, good first issue.

Project layout
.
├─ cmd/
│  ├─ api/         # HTTP API binary
│  └─ scheduler/   # background worker binary
├─ internal/…      # packages shared internally (domain, backoff, leases)
├─ Dockerfile      # single root Dockerfile (COMPONENT/BUILD_PATH args)
├─ docs/           # diagrams, guides (e.g., architecture.svg)
└─ .github/workflows/docker-publish.yaml


(Names may evolve; see README for the latest.)

Release notes & tagging

We tag releases as vX.Y.Z. CI publishes multi-arch images with the semver tag, latest, and the short SHA. If your change needs a release note, add a bullet in CHANGELOG.md or the release PR.

Security

Please do not file public issues for sensitive vulnerabilities. Report privately via:

Email: [REPLACE-ME] (e.g., security@sirclappington.com)

Or open a private GitHub security advisory

See SECURITY.md if present.

Community & support

GitHub Issues/Discussions for questions and proposals

PRs for fixes and improvements

Social preview: “Enq — Job scheduler without Redis (Go + Docker, Postgres-backed)”

Thanks for contributing! 🙌