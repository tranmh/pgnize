# pgnize API & types contract (authoritative)

Single source of truth for the REST API and shared JSON shapes. Backend (`internal/httpapi`) and frontend
(`web/src/lib/api-client.ts`) MUST both conform to this. All bodies are JSON unless noted. Auth is a
session cookie (`pgnize_session`, HttpOnly, SameSite=Lax). Base path `/api`.

## Shared JSON shapes

### Header
```json
{ "white": "", "black": "", "event": "", "site": "", "date": "", "round": "", "board": "", "result": "*" }
```
`result` ∈ `"1-0" | "0-1" | "1/2-1/2" | "*"`. `date` is `YYYY.MM.DD` or `""`.

### Move (in a draft)
```json
{ "ply": 1, "side": "white", "san": "e4", "fenAfter": "rnbq...",
  "clockSec": null, "isLegal": true, "recognizedText": "e4", "corrected": false }
```
`side` ∈ `"white" | "black"`.

### GameDraft  (GET /games/{id}, GET /convert/{jobId}/game)
```json
{ "id": "uuid", "source": "recognized|manual", "status": "draft|reviewing|saved",
  "header": { ...Header }, "startFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "moves": [ ...Move ], "imageUrl": "https://presigned-or-null", "confidence": 0.0 }
```

### GameSummary  (list items)
```json
{ "id": "uuid", "white": "", "black": "", "event": "", "date": "", "result": "*",
  "moveCount": 0, "savedAt": "RFC3339-or-null" }
```

### Error
```json
{ "error": "machine_code", "message": "human readable", "failedAt": 0 }
```
`failedAt` present only for `error:"illegal_move"` (0-based ply index that failed).

## Endpoints

### Health
- `GET /healthz` → 200 `{"status":"ok"}` (liveness)
- `GET /readyz` → 200 `{"status":"ok"}` / 503 (DB ping)

### Auth
- `POST /api/auth/register` `{name,email,password}` → 201 `{user:{id,name,email}}` + cookie. 409 if email taken.
- `POST /api/auth/login` `{email,password}` → 200 `{user}` + cookie. 401 on bad creds.
- `POST /api/auth/logout` → 204 (clears cookie).
- `GET  /api/auth/me` → 200 `{user}` / 401.

### Anonymous convert (no account; ephemeral, TTL-purged)
- `POST /api/convert` multipart field `image` → 202 `{jobId}`. Tight per-IP rate limit.
- `GET  /api/convert/{jobId}` → 200 `{status:"queued|running|done|failed", gameId?, error?}`.
- `GET  /api/convert/{jobId}/game` → 200 `GameDraft` (once done).
- `POST /api/convert/{jobId}/export` `{header, moves:[{ply,san,clockSec?}]}` → 200 `text/plain` PGN.
  Server replays moves via chesskit; 422 `{error:"illegal_move",failedAt}` if any move illegal.

### Account: upload → job → review → save
- `POST /api/uploads` multipart `image`, optional form field `consentTraining=true` → 202 `{uploadId, jobId}`.
- `GET  /api/jobs/{jobId}` → 200 `{status, gameId?, error?}`.
- `POST /api/games` `{source:"manual"}` → 201 `{game: GameDraft}` (empty draft for manual entry).
- `GET  /api/games/{id}` → 200 `GameDraft`.
- `PATCH /api/games/{id}` `{header, moves:[{ply,san,clockSec?}], startFen?}` → 200 `{game: GameDraft}`.
  Server replays via chesskit; 422 `{error:"illegal_move", failedAt}` on first illegal ply. On success
  sets `status:"saved"`, writes the canonical PGN, records a feedback row, bumps player usage.
- `DELETE /api/games/{id}` → 204.

### Library
- `GET /api/games?q=&player=&event=&from=&to=&page=1&pageSize=20` → 200
  `{games:[GameSummary], total, page, pageSize}` (saved games for the current user; `q` = trigram search).
- `GET /api/games/{id}/pgn` → 200 `text/plain` (single game PGN).
- `POST /api/games/export` `{ids:["uuid", ...]}` → 200 `text/plain` (concatenated multi-game PGN).

### Players (autocomplete)
- `GET /api/players?q=` → 200 `{players:[{id, fullName, club, fideId}]}` (current user's pool).

## Notes for both sides
- Frontend talks to the API at `/api/*`; in dev Next.js rewrites `/api/*` → `PGNIZE_API_URL` (`:8080`).
- Polling cadence for jobs: 1.5s, give up after ~5 min, surface `failed`/`error` to the user.
- The board/move legality shown live in the UI is advisory (client-side chess lib); the **server is
  authoritative** on save/export.
