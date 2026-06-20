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

For Google Gemini Flash (cloud), set an API key — the `gemini` backend is then offered to
clients and becomes the default recognizer:

```bash
GEMINI_API_KEY=… make dev   # optional: GEMINI_MODEL (default gemini-2.5-flash)
```

The recognition backend is also selectable per upload/convert request when more than one is
configured; `GET /api/recognizers` lists the available engines.

## Deploy (production)

Production runs the Docker Compose `prod` profile (api · web · Caddy · db-backup)
behind Caddy, which auto-provisions TLS for `$PGNIZE_DOMAIN`. On the production
host:

```bash
cp .env.example .env          # then set AUTH_SECRET, POSTGRES_PASSWORD,
                              # MINIO_ROOT_PASSWORD (+ GEMINI_API_KEY if RECOGNIZER=gemini)
scripts/deploy.sh             # builds, brings up the stack, waits for /healthz
```

`scripts/deploy.sh` pulls the latest code, rebuilds the api/web images, starts
the stack (adding the `vlm`/Ollama profile and pulling the model when
`RECOGNIZER=ollama`), and verifies the API is healthy. Migrations run on API
boot. It defaults to `pgnize.openpairing.org`; override with `--domain HOST`.
Point DNS at the host and open ports 80/443 before the first run. See
`scripts/deploy.sh --help` for all options.

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

## License

PGNize is licensed under the **GNU Affero General Public License v3.0** (AGPL-3.0) — see
[`LICENSE`](./LICENSE). Because the app is network-served, AGPL §13 requires that users
interacting with it over a network can obtain the complete corresponding source.

### Third-party: Stockfish

The web app bundles [Stockfish.js](https://github.com/nmrugg/stockfish.js) (a WebAssembly build of
[Stockfish](https://github.com/official-stockfish/Stockfish)) under `web/public/engine/` for
in-browser position evaluation. Stockfish is licensed under **GPL-3.0**, which is compatible with
AGPL-3.0 (see AGPL §13 / GPL §13): the engine remains under GPL-3.0 within this AGPL-3.0 project.
The corresponding source and full license text are referenced in
[`web/public/engine/NOTICE.txt`](./web/public/engine/NOTICE.txt) and `Copying.txt`.
