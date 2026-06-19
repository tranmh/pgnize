# PGNize

Convert photos of handwritten chess score sheets (German *Partieformular*) into
human-verified PGN, and build a browsable, searchable library of your games.

## Stack

- **Backend**: Go (REST) — chi router, pgx, goose migrations.
- **Reusable core**: [`chesskit`](./chesskit) — a standalone Go chess library (SAN/FEN/PGN/legality)
  wrapping `notnil/chess`, designed for reuse by other projects (e.g. swiss-manager).
- **Frontend**: Next.js + TypeScript + React.
- **Recognition**: local open-source vision model via Ollama, behind a swappable `Recognizer` interface.
  A mandatory move-by-move review loop guarantees every saved PGN is human-verified.
- **Infra**: Docker Compose, PostgreSQL 16, MinIO (image storage), Caddy (prod reverse proxy).

## Quick start (dev)

```bash
cp .env.example .env
docker compose up -d db minio        # Postgres + object storage
make migrate                         # apply schema
make dev                             # Go API (:8080) + Next.js (:3000)
```

Recognition defaults to `RECOGNIZER=fake` (deterministic, no model needed). For the real local VLM:

```bash
docker compose --profile vlm up -d ollama
docker exec -it $(docker compose ps -q ollama) ollama pull minicpm-v
RECOGNIZER=ollama make dev
```

## Tests (TDD)

```bash
make test        # Go unit (chesskit + internal), no DB
make test-int    # Go integration, needs Postgres
make e2e-api     # Playwright API project (no browser)
make e2e-ui      # Playwright UI project (chromium)
```

## Layout

See [`docs/plans/pgnize.md`](./docs/plans/pgnize.md) for the full design and `CODEMAP.md` for a
per-directory orientation.
