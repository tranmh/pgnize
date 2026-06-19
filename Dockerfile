# syntax=docker/dockerfile:1

# ---- Go build stage ----
FROM golang:1.22-bookworm AS go-build
WORKDIR /src
COPY go.work go.work.sum* ./
COPY go.mod go.sum* ./
COPY chesskit/go.mod chesskit/go.sum* ./chesskit/
RUN go mod download all || true
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

# ---- Web build stage ----
FROM node:22-bookworm-slim AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- API runtime ----
FROM gcr.io/distroless/static-debian12 AS api
WORKDIR /app
COPY --from=go-build /out/api /app/api
COPY migrations /app/migrations
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
