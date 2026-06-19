# pgnize — Handwritten Chess Score Sheet → PGN

## Context

Convert photos of handwritten German chess score sheets ("Partieformular") into
human-verified PGN, giving each user a browsable, searchable library of games with PGN
export. The hard part is reading handwriting; the chess logic is the easy, reusable part.

Confirmed decisions:
- **Stack**: Go backend (REST), Next.js + TypeScript + React frontend, Docker Compose, PostgreSQL 16.
- **Recognition**: local/open-source vision model (VLM) on a **CPU-only modest box** (small quantized VLM
  via an Ollama HTTP server), behind a clean `Recognizer` interface so a cloud GPU / API can drop in
  later. **A mandatory manual review loop guarantees correctness** — saved PGN is always human-verified.
- **Learning, phased**: store every photo + corrected transcription as a feedback dataset. v1 feeds a
  user's past corrected examples as few-shot context; store shaped to later export JSONL for cloud-GPU
  LoRA. No on-box training.
- **Reusable Go chess library (`chesskit`)** = the core job: wraps `github.com/notnil/chess`, exposes
  board/FEN, SAN parse+generate, legality, PGN read/write, results. Own module so swiss-manager (or
  anything) can reuse it; designed to be exposable as an HTTP microservice with minimal change. Chess
  domain only — recognition orchestration stays in pgnize app code.
- **Auth**: multi-user accounts (private library) **plus** anonymous one-off convert (no saved library).
- **Auto-extract**: full header (players, event, date, round, board, result, clocks) + moves; player
  names autocomplete against the user's saved pool.
- **TDD at every level**: Go unit, Go integration (real Postgres), Playwright e2e (api + ui). Tests first.

## Repo layout

```
chesskit/          separate Go module github.com/tranmh/chesskit (reusable core) + httpsvc/ facade
cmd/api/           REST server; recognition worker folded in as a goroutine pool
internal/          config httpapi auth store recognition jobs storage domain
migrations/        goose .sql
web/               Next.js + TS + React (app router)
e2e/               Playwright: projects "api" (default) + "ui"
```
`go.work` links the root module + chesskit for local dev.

## chesskit public surface

Types `FEN SAN Result Move{SAN,FromFEN,ToFEN,ClockSec} Header Game{Header,Moves,StartFEN}`; functions
`StartingFEN ParseSAN Validate LegalMovesSAN ApplyMoves(start,sans)->(positions,err,failedAt)
ParsePGN(tolerant multi-game,%clk,truncate-on-illegal) WritePGN WriteBundlePGN NormalizeResult`; errors
`ErrIllegalMove ErrAmbiguousMove`. No notnil types leak across the public API.

## Recognition subsystem (app, not chesskit)

`Recognizer` interface returns raw move tokens (not yet legality-checked) + header + confidence + raw
JSON. `ollama.go` (local VLM, JSON-schema-constrained), `fake.go` (deterministic, for CI), `prompt.go`,
`postprocess.go` (German K/D/T/L/S→K/Q/R/B/N, castling/e.p./promotion/capture-colon/check-mate/draw-resign
normalization, then `chesskit.ApplyMoves` reconciliation), `fewshot.go`. Async job model: upload → queued
`recognition_job` → goroutine pool claims via `SELECT … FOR UPDATE SKIP LOCKED` → draft game → poll
`GET /jobs/{id}`. Per-IP rate limiting.

## Data model (Postgres; goose; text+CHECK enums; uuid pks)

users · sessions (opaque token hash) · players (per-user autocomplete pool) · uploads (user nullable,
storage_key, consent_training) · recognition_jobs · games (header + final_pgn, trigram search) · moves
(san, fen_after, is_legal, recognized_text) · feedback_corrections (before/after json) · rate_limit_entries.

## API (chi, /api, session cookie)

Health `/healthz /readyz`. Auth `register login logout me`. Anonymous `POST /convert`→job, poll, export
(ephemeral). Account `POST /uploads`→job; `GET /jobs/{id}`; `GET /games/{id}` draft; `PATCH /games/{id}`
(server replays via chesskit, rejects with failedAt, writes final_pgn + feedback); library
`GET /games?q…`, `GET /games/{id}/pgn`, `POST /games/export` bundle; `GET /players?q`; manual
`POST /games {source:"manual"}`.

## Review UX

Split screen photo|board+movelist. Click move→board jumps to fen_after. Illegal→red badge + downstream
blocked + correction dropdown from LegalMovesSAN; ambiguous→amber + disambiguated options; `?` placeholder;
truncation allowed. Save gated on all-legal; server authoritative.

## Storage & retention

`STORAGE_DRIVER=auto|s3|filesystem`; S3 via aws-sdk-go-v2 against minio in dev; presigned URLs to UI.
`consent_training` gate; anonymous + non-consented uploads TTL-purged nightly.

## TDD plan

Unit: chesskit SAN/FEN/PGN/legality tables + ApplyMoves failedAt; recognition German→SAN normalization +
postprocess reconciliation + few-shot + prompt schema (with fake recognizer). Integration (real Postgres):
repos CRUD, rate-limit windows, job pipeline with stub recognizer (SKIP LOCKED, no double-claim),
save-illegal rejected / save-legal writes pgn+feedback. E2E (Playwright api+ui, RECOGNIZER=fake):
anonymous convert; account upload→review→save→export; player autocomplete; rate-limit 429; review
workbench UI.

## Tooling

go-chi/chi/v5, jackc/pgx/v5, pressly/goose/v3 (goose up on boot), x/crypto/bcrypt, aws-sdk-go-v2,
testcontainers-go. Makefile: build dev migrate test test-int e2e-api e2e-ui lint seed. Compose: db,
minio, ollama (vlm profile), api/web/caddy/db-backup (prod profile).

## Milestones (riskiest = M3)

M1 chesskit + PGN export · M2 manual entry + review (no AI) · M3 local VLM recognizer (riskiest) ·
M4 feedback + few-shot · M5 auth + library + anonymous convert · M6 deploy.

## Verification

`make test` + `make test-int` green; `make e2e-api` (anonymous convert + account upload→review→save→export
with RECOGNIZER=fake) + `make e2e-ui` (review workbench); manual smoke with a real sheet via
`docker compose --profile vlm up`; reuse check: `go test ./chesskit/...` standalone with no pgnize imports.
