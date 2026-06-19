# e2e (Playwright)

Two projects, mirroring swiss-manager:

- **api** (default, no browser) — pure REST against the Go backend.
- **ui** (chromium) — the Next.js frontend.

Servers are started **externally** so the suite stays fast and explicit. Always run with
`RECOGNIZER=fake` so recognition is deterministic and CI is hermetic.

## Run

```bash
# 1. Test database + API (fake recognizer)
docker run -d --name pgnize-e2e-db -e POSTGRES_USER=pgnize -e POSTGRES_PASSWORD=pgnize \
  -e POSTGRES_DB=pgnize -p 5432:5432 postgres:16-alpine
DATABASE_URL=postgres://pgnize:pgnize@localhost:5432/pgnize?sslmode=disable \
  AUTH_SECRET=dev-secret-32-bytes-xxxxxxxxxxxx RECOGNIZER=fake STORAGE_DRIVER=filesystem \
  go run ./cmd/api &

# 2. (ui only) the frontend
cd web && PGNIZE_API_URL=http://localhost:8080 npm run dev &

# 3. Tests
cd e2e && npm install && npx playwright install chromium
npm run test:api      # REST journeys
npm run test:ui       # browser smoke
```

Override `PGNIZE_API_BASE` / `PGNIZE_WEB_BASE` to point at other hosts.
