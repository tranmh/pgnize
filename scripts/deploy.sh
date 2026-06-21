#!/usr/bin/env bash
#
# deploy.sh — deploy PGNize to a production host (default: pgnize.openpairing.org).
#
# Mirrors OpenPairing.org's deploy approach: pull the latest code, (re)build the
# images, bring up the `prod` Docker Compose stack, and verify the API is
# healthy. Migrations run automatically on API boot (see cmd/api/main.go), so no
# separate migration step is needed.
#
# This script is meant to run ON the production host, from the repository root or
# the scripts/ directory. It is idempotent: re-running it ships the current
# checkout and recreates only the containers whose images changed.
#
# Topology (see docker-compose.yml):
#   prod profile -> api, web, caddy, db-backup   (always deployed)
#   vlm  profile -> ollama                        (local vision model)
#   default      -> db, minio                     (data services)
#
# Recognition backend (RECOGNIZER in .env) drives whether the vlm profile is
# included:
#   RECOGNIZER=gemini  -> cloud VLM, no ollama (needs GEMINI_API_KEY)
#   RECOGNIZER=ollama  -> local VLM, ollama is started and the model is pulled
#   RECOGNIZER=fake    -> deterministic stub (not for real production use)
#
# Usage:
#   scripts/deploy.sh [options]
#
# Options:
#   --no-pull        Skip `git pull` (deploy the current checkout as-is).
#   --no-build       Reuse existing images instead of rebuilding api/web.
#   --pull-model     Force (re)pull of the Ollama model even if present.
#   --domain HOST    Override the public domain (default: pgnize.openpairing.org).
#   -h, --help       Show this help.
#
# Required environment (in ./.env — copy from .env.example and fill in):
#   AUTH_SECRET            32+ byte random secret (must not be the placeholder).
#   POSTGRES_PASSWORD      Strong DB password (must not be the default `pgnize`).
#   MINIO_ROOT_PASSWORD    Strong object-store password (not the default).
#   GEMINI_API_KEY         Required only when RECOGNIZER=gemini.
#
set -euo pipefail

# ---- Locate the repo root (script lives in <root>/scripts) ------------------
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." >/dev/null 2>&1 && pwd)"
cd "${ROOT_DIR}"

# ---- Defaults / option parsing ----------------------------------------------
DOMAIN="${PGNIZE_DOMAIN:-pgnize.openpairing.org}"
DO_PULL=1
DO_BUILD=1
FORCE_MODEL_PULL=0

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarn:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

usage() { sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//; /^set -euo/d'; }

while [ $# -gt 0 ]; do
	case "$1" in
		--no-pull)     DO_PULL=0 ;;
		--no-build)    DO_BUILD=0 ;;
		--pull-model)  FORCE_MODEL_PULL=1 ;;
		--domain)      shift; [ $# -gt 0 ] || die "--domain needs an argument"; DOMAIN="$1" ;;
		-h|--help)     usage; exit 0 ;;
		*)             die "unknown option: $1 (try --help)" ;;
	esac
	shift
done
export PGNIZE_DOMAIN="${DOMAIN}"

# ---- Resolve the docker compose command -------------------------------------
if docker compose version >/dev/null 2>&1; then
	COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
	COMPOSE=(docker-compose)
else
	die "docker compose is not available — install Docker Compose v2"
fi
command -v docker >/dev/null 2>&1 || die "docker is not installed"

# ---- Preflight: .env and required secrets -----------------------------------
[ -f .env ] || die ".env not found — copy .env.example to .env and fill in production values"

# Read a value from .env without sourcing arbitrary shell.
env_val() { grep -E "^${1}=" .env | tail -n1 | cut -d= -f2- | sed -e 's/^"//' -e 's/"$//'; }

require_secret() {
	local name="$1" placeholder="$2" val
	val="$(env_val "${name}")"
	[ -n "${val}" ]                 || die "${name} is empty in .env"
	[ "${val}" != "${placeholder}" ] || die "${name} still has its placeholder/default value — set a real secret in .env"
}

require_secret AUTH_SECRET         "change-me-in-production-please-32-bytes-min"
require_secret POSTGRES_PASSWORD   "pgnize"
require_secret MINIO_ROOT_PASSWORD "minioadmin"

RECOGNIZER="$(env_val RECOGNIZER)"; RECOGNIZER="${RECOGNIZER:-ollama}"

# ---- Decide which profiles to bring up --------------------------------------
PROFILES=(--profile prod)
case "${RECOGNIZER}" in
	ollama)
		PROFILES+=(--profile vlm)
		;;
	gemini)
		[ -n "$(env_val GEMINI_API_KEY)" ] || die "RECOGNIZER=gemini but GEMINI_API_KEY is empty in .env"
		;;
	fake)
		warn "RECOGNIZER=fake — the deterministic stub is not suitable for real production traffic"
		;;
	*)
		warn "unrecognized RECOGNIZER=${RECOGNIZER}; defaulting profiles to prod-only"
		;;
esac

log "Deploying PGNize to ${DOMAIN} (recognizer: ${RECOGNIZER})"

# ---- Pull latest code -------------------------------------------------------
if [ "${DO_PULL}" -eq 1 ]; then
	if [ -d .git ]; then
		BRANCH="$(git rev-parse --abbrev-ref HEAD)"
		log "Pulling latest ${BRANCH}"
		git pull --ff-only origin "${BRANCH}"
	else
		warn "not a git checkout — skipping pull"
	fi
fi

# ---- Build images -----------------------------------------------------------
if [ "${DO_BUILD}" -eq 1 ]; then
	log "Building api + web images"
	"${COMPOSE[@]}" "${PROFILES[@]}" build api web
fi

# ---- Bring up the stack -----------------------------------------------------
log "Starting data services (db, minio)"
"${COMPOSE[@]}" up -d db minio

if printf '%s\n' "${PROFILES[@]}" | grep -q '^vlm$' || [ "${RECOGNIZER}" = "ollama" ]; then
	log "Starting Ollama"
	"${COMPOSE[@]}" --profile vlm up -d ollama
	MODEL="$(env_val RECOGNIZER_MODEL)"; MODEL="${MODEL:-minicpm-v}"
	if [ "${FORCE_MODEL_PULL}" -eq 1 ] || \
	   ! "${COMPOSE[@]}" exec -T ollama ollama list 2>/dev/null | grep -q "${MODEL}"; then
		log "Pulling Ollama model ${MODEL} (may download several GB)"
		"${COMPOSE[@]}" exec -T ollama ollama pull "${MODEL}"
	fi
fi

log "Starting application services (api, web, caddy, db-backup)"
"${COMPOSE[@]}" "${PROFILES[@]}" up -d

# ---- Health check -----------------------------------------------------------
log "Waiting for the API to become healthy"
healthy=0
for i in $(seq 1 30); do
	if "${COMPOSE[@]}" exec -T api /app/api -healthcheck >/dev/null 2>&1; then
		healthy=1
		break
	fi
	sleep 2
done

if [ "${healthy}" -ne 1 ]; then
	warn "API did not report healthy after ~60s — recent api logs:"
	"${COMPOSE[@]}" logs --tail 50 api || true
	die "deploy failed: API unhealthy"
fi

log "API healthy."
"${COMPOSE[@]}" "${PROFILES[@]}" ps

cat <<EOF

✅ PGNize deployed to ${DOMAIN}
   - Caddy serves https://${DOMAIN} (TLS auto-provisioned on first request;
     ensure DNS for ${DOMAIN} points at this host and ports 80/443 are open).
   - Verify:  curl -fsS https://${DOMAIN}/healthz
EOF
