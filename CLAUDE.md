# CLAUDE.md

@CODEMAP.md

PGNize — convert photos of handwritten chess score sheets (German *Partieformular*) into human-verified
PGN, and photos of a board (physical or digital diagram) into an editable position (FEN), with a
browsable per-user game library. (The Go module path / code identifiers stay lowercase
`pgnize` by convention; the product name is **PGNize**.)

## Stack

Go 1.22 backend (chi · pgx · goose) · `chesskit` reusable Go chess module (wraps `notnil/chess`) ·
Next.js + TypeScript + React frontend · PostgreSQL 16 · MinIO/S3 image storage · Ollama local VLM
(swappable `Recognizer`) · Docker Compose · Playwright e2e.

## Workflow

- **Always work in a git worktree** (not the primary checkout) — parallel sessions share this repo.
- **TDD everywhere**: write the failing test first. Unit (no DB), integration (real Postgres, build tag
  `integration` + `RUN_INTEGRATION=1`), e2e (Playwright `api` + `ui` projects, `RECOGNIZER=fake`).

## Hard rules

- **`chesskit` is the reusable core and must stay clean**: it is a separate Go module
  (`github.com/tranmh/chesskit`) that MUST NOT import any pgnize `internal/` package. No `notnil/chess`
  types may appear in its public API — only the JSON-friendly value types it defines. It must be usable
  and testable standalone (`cd chesskit && go test ./...`).
- **The review loop is the correctness guarantee.** Never let recognition output reach a saved PGN
  unverified. The server is always authoritative: re-validate moves via `chesskit.ApplyMoves` on save and
  reject with `failedAt` on the first illegal ply.
- **Recognition is pluggable.** All model-specific code sits behind the `Recognizer` interface in
  `internal/recognition`. Tests and CI use the `fake` recognizer; the real model is opt-in only.
- **Recognition orchestration stays out of `chesskit`** — German-notation normalization, prompts, jobs,
  and few-shot all live in `internal/`.

## Commands

```bash
make test        # Go unit (chesskit + internal)
make test-int    # Go integration (Postgres)
make e2e-api     # Playwright api project
make e2e-ui      # Playwright ui project
make migrate     # apply goose migrations
make dev         # local API + web
make lint        # go vet + eslint
```

## Conventions

- Postgres enums as `text + CHECK` (goose-friendly). UUID pks (`gen_random_uuid()`).
- API: chi router, session-cookie auth, JSON in/out, Zod-equivalent validation in Go; rate-limit mutating
  + auth endpoints per-IP (port of OpenPairing.org `consumeRateLimit`).
- German score-sheet specifics (piece letters, castling, e.p., promotion, draw/resign words) are handled
  deterministically in `internal/recognition/postprocess.go`, never left to the model.
