## What
<!-- Short summary of the change -->
## Why
<!-- Problem/motivation, linked issues -->

Closes #NNN

## How
<!-- High-level approach, notable decisions/trade-offs -->
## Testing

 Unit tests pass (go test ./...)

 Manual smoke test (API + Scheduler)

 Docker build works (if applicable)

## Steps (if manual)

Example:

Run: docker compose up -d

Enqueue: curl -X POST http://localhost:8080/v1/jobs
 -H "Content-Type: application/json" -d '{"type":"demo.echo","payload":{"msg":"hi"}}'

## Screenshots / Logs (if relevant)
<!-- Paste snippets or images -->
Breaking changes

 None

 Yes (describe migration/compat)

## Checklist

 I read the Contributing Guide (./CONTRIBUTING.md)

 I updated docs (README/CHANGELOG/examples) as needed

 I added or updated tests

 I considered security impact (secrets, auth, data exposure)

 Conventional Commit message (e.g., feat: ..., fix: ..., docs: ...)

## Notes for reviewers
<!-- Call out risky areas, follow-up items, or things to watch -->