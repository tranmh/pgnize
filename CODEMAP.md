# Codemap

One-line orientation per directory. Conventions live in the nearest `CLAUDE.md`.

## Top level

- `chesskit/` — **reusable Go chess library** (separate module). Board/FEN, SAN parse+generate, legality,
  PGN read/write, results. Wraps `notnil/chess`; no pgnize imports. `chesskit/httpsvc/` = optional HTTP facade.
- `cmd/api/` — REST server entrypoint; runs goose migrations on boot; recognition worker folded in.
- `internal/` — pgnize application code (not importable externally):
  - `config/` — env parsing
  - `httpapi/` — chi router, middleware (auth, rate-limit, recover), handlers, `/healthz`
  - `auth/` — sessions, password hashing, request context
  - `store/` — Postgres repositories (pgx); one file per aggregate
  - `recognition/` — `Recognizer` interface + `ollama`/`fake` impls, prompt, postprocess, few-shot
  - `jobs/` — DB-backed job queue (SKIP LOCKED) + worker pool
  - `storage/` — blob storage abstraction (`auto`/`s3`/`filesystem`)
  - `domain/` — shared app types
- `migrations/` — goose SQL migrations
- `web/` — Next.js + TS + React frontend (app router); the review workbench lives here
- `e2e/` — Playwright tests; `api` (no browser, default) + `ui` (chromium) projects
- `docs/plans/` — committed implementation plans
- `go.work` — links the root module + `chesskit` for local dev
