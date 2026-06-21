# PGNize

Turn photos of chess into data you can use:

- **Score sheet → PGN.** Photograph a handwritten German score sheet (*Partieformular*) and get a
  move-by-move game back. Recognition is paired with a mandatory review loop, so every saved PGN is
  human-verified.
- **Board photo → position (FEN).** Photograph a physical board or a digital diagram and get the
  position back as an editable FEN, then export it as PGN. A position is modeled as a draft game
  whose start FEN is the recognized position, reusing the same upload/job/review/export machinery.

Both flows are available anonymously (no account) or signed in. Signed-in games land in a browsable,
searchable per-user library with an in-browser Stockfish analysis viewer.

## Stack

- **Backend**: Go (REST) — chi router, pgx, goose migrations.
- **Reusable core**: [`chesskit`](./chesskit) — a standalone Go chess library (SAN/FEN/PGN/legality)
  wrapping `notnil/chess`, designed for reuse by other projects (e.g. swiss-manager).
- **Frontend**: Next.js + TypeScript + React — review workbench, editable position editor, and a
  Stockfish (WASM) analysis viewer. German (default) and English i18n.
- **Recognition**: a swappable `Recognizer` interface with `fake` (deterministic, CI default),
  local Ollama VLM, and Google Gemini Flash backends. Recognizers return moves with a per-move
  confidence and an 8×8 grid for positions; FEN is assembled deterministically in Go. A mandatory
  move-by-move review loop — server-authoritative, re-validated via `chesskit.ApplyMoves` on save —
  guarantees every saved PGN is human-verified.
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

### Recognition quality (developer harness)

`make poseval` scores board photo → FEN accuracy (Ollama and/or Gemini) over the
`testdata/positions` corpus (physical photos + digital diagrams), reporting per-square accuracy and
exact-match rate. See [`poseval-report.md`](./poseval-report.md) for a recorded run: exact
recognition is near-0% for both backends and Gemini is strong on digital diagrams but weak on
physical photos — which is why the manual correction UI is essential, not optional.

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
