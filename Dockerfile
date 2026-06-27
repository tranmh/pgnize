# syntax=docker/dockerfile:1

# ---- Go build stage ----
FROM golang:1.25-bookworm AS go-build
WORKDIR /src
COPY go.work go.work.sum* ./
COPY go.mod go.sum* ./
COPY chesskit/go.mod chesskit/go.sum* ./chesskit/
RUN go mod download all || true
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

# ---- Stockfish stage (server-side engine for the conversational coach) ----
# A plain Debian stage that carries the Stockfish UCI binary; copied into the API
# runtime below. The engine is opt-in (ENGINE=stockfish), so this only matters when
# the conversational coach is enabled in prod.
FROM debian:bookworm-slim AS stockfish
RUN apt-get update \
 && apt-get install -y --no-install-recommends stockfish \
 && rm -rf /var/lib/apt/lists/*

# ---- Web build stage ----
FROM node:22-bookworm-slim AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- API runtime ----
# distroless/base (not static): the Stockfish binary is dynamically linked against
# glibc, which base provides. The C++ runtime it also needs (libstdc++, libgcc_s) is
# copied from the stockfish stage. With ENGINE=fake (the default) the engine binary
# is unused, but shipping it keeps the conversational coach a single env flip away.
FROM gcr.io/distroless/base-debian12 AS api
WORKDIR /app
COPY --from=go-build /out/api /app/api
COPY migrations /app/migrations
COPY --from=stockfish /usr/games/stockfish /app/stockfish
COPY --from=stockfish /usr/lib/x86_64-linux-gnu/libstdc++.so.6* /usr/lib/x86_64-linux-gnu/
COPY --from=stockfish /usr/lib/x86_64-linux-gnu/libgcc_s.so.1* /usr/lib/x86_64-linux-gnu/
ENV ENGINE_PATH=/app/stockfish
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]

# ---- Web runtime ----
FROM node:22-bookworm-slim AS web
WORKDIR /web
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1
COPY --from=web-build /web/.next/standalone ./
COPY --from=web-build /web/.next/static ./.next/static
COPY --from=web-build /web/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
