#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────
# Interactive defensive deploy of PGNize to root@openpairing01.
#
# Same server, same infrastructure, and the same SSH-driven approach as
# ~/work/swiss-manager/scripts/deploy.sh. Runs FROM your local machine and
# pushes to prod over ssh/scp — it does NOT run on the prod host.
#
# Steps (mirrors swiss-manager):
#   0  preflight        (go vet, go test, go build — local only)
#   0a docker preflight (local 'compose --profile prod build api web' — catches
#                        Dockerfile-recipe bugs before prod step 6 sees them)
#   0b runtime smoke    (ephemeral postgres + api container locally, runs
#                        migrations on a fresh DB, polls /healthz — catches
#                        container-runtime bugs docker build cannot see:
#                        failing migrations, missing env, startup crashes)
#   1  push             (origin/main)
#   2  capture state    (PREV_REMOTE_SHA, PREV_IMAGE_*, remote RECOGNIZER)
#   2a resource check   (RAM / CPU / disk on prod)
#   3  DB backup        (pg_dump inside the db-backup container -> ./backups)
#   4  code transfer    (tarball + scp + extract; .env and ./backups preserved)
#   5  tag images       (rollback tags for api + web)
#   6  rebuild & roll   (docker compose build api web -> up -d; pulls the ollama
#                        model when RECOGNIZER=ollama)
#   7  health + smoke   (internal + external + manual prompt)
#   8  post-deploy cleanup (keep last N rollback tags, prune old build cache)
#
# Default behaviour: prompt before every transition.
# Flags:
#   --yes        non-interactive deploy. Skips per-step prompts AND the
#                manual smoke checklist. Prompts that survive --yes are
#                limited to genuinely destructive choice points: prod
#                working-tree overwrite, post-failure rollback/continue
#                decisions, DB restore. Anything routine (backup, image
#                tag, build, roll, cleanup) auto-proceeds.
#   --rollback   read /tmp/pgnize-deploy.last and run rollback flow.
#   --cleanup    reclaim disk on prod (docker prune + builder prune); no deploy.
#
# State: /tmp/pgnize-deploy.last (KEY=VALUE; sourced for --rollback).
# Lock:  /tmp/pgnize-deploy.lock.
# ──────────────────────────────────────────────────────────

set -euo pipefail
IFS=$'\n\t'

# ─── Configuration (edit at top, no env-var indirection) ──
LOCAL_REPO="/home/tranmh/work/pgnize"
REMOTE_HOST="root@openpairing01"
REMOTE_PATH="/root/work/pgnize"
DEPLOY_BRANCH="main"
DOMAIN="pgnize.openpairing.org"
EXTERNAL_HEALTH_URL="https://${DOMAIN}/healthz"
COMPOSE_PROFILE="prod"
APP_SERVICE="api"          # the health-bearing service (migrations run on its boot)
WEB_SERVICE="web"
DB_SERVICE="db"
DB_BACKUP_SERVICE="db-backup"
# docker compose derives built-image names from <project>-<service>. Both the
# local checkout and the prod checkout live in a directory called "pgnize", so
# the compose project name is "pgnize" on both ends.
IMAGE_API="pgnize-api"
IMAGE_WEB="pgnize-web"
HEALTH_TIMEOUT_SECONDS=240
LOCK_FILE_LOCAL="/tmp/pgnize-deploy.lock"
STATE_FILE="/tmp/pgnize-deploy.last"

# Profiles activated on the remote. Refined to add --profile vlm once we read
# RECOGNIZER from the prod .env (step 2). Default keeps prod-only services.
REMOTE_PROFILE_ARGS="--profile $COMPOSE_PROFILE"
RECOGNIZER=""

# Pin the compose file explicitly. docker-compose.override.yml is a DEV-ONLY
# file that publishes db/minio/ollama host ports — it is git-tracked, so it
# ships to prod and would otherwise be auto-merged by a bare `docker compose`,
# colliding with the swiss-manager stack on 9000/9001 and exposing Postgres.
# `-f docker-compose.yml` makes prod ignore any override.
COMPOSE_BASE="-f docker-compose.yml"

# Resource thresholds (Step 2a)
DISK_ROOT_WARN_GB=5
DISK_ROOT_ABORT_GB=2
DISK_DOCKER_WARN_GB=8
DISK_DOCKER_ABORT_GB=3
MEM_AVAIL_WARN_MB=1500
MEM_AVAIL_ABORT_MB=800
LOAD_PER_CPU_WARN=2.0

# Post-deploy cleanup (Step 8). Each deploy creates new rollback images (api +
# web) and a fresh build-cache snapshot. Without bounds, prod disk grows every
# deploy until a manual `--cleanup`. Defaults keep enough headroom to roll back
# to the previous deploy *and* the one before it.
KEEP_ROLLBACK_TAGS=2
BUILD_CACHE_MAX_GB=5

# ─── Flags ──
ASSUME_YES=0
MODE=deploy

usage() {
  cat >&2 <<EOF
Usage: $0 [--yes] [--rollback | --cleanup]

  --yes        skip per-step prompts (destructive ops still confirm)
  --rollback   restore from saved state in $STATE_FILE
  --cleanup    reclaim disk on prod (docker system prune + builder prune); no deploy
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes)      ASSUME_YES=1; shift ;;
    --rollback) MODE=rollback; shift ;;
    --cleanup)  MODE=cleanup; shift ;;
    -h|--help)  usage; exit 0 ;;
    *) echo "Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

# ─── Helpers ──
log()  { echo "[$(date '+%H:%M:%S')] $*" >&2; }

step() {
  local n="$1"; shift
  echo >&2
  echo "═══════════════════════════════════════════════════════════════" >&2
  echo "  Step $n: $*" >&2
  echo "═══════════════════════════════════════════════════════════════" >&2
}

# confirm "prompt" [--always-prompt]
# Returns 0 on y/Y. With --yes, auto-yes unless --always-prompt is given.
confirm() {
  local prompt="$1"
  local always_prompt=0
  shift || true
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --always-prompt) always_prompt=1 ;;
    esac
    shift
  done
  if [[ $ASSUME_YES -eq 1 && $always_prompt -eq 0 ]]; then
    log "AUTO-YES: $prompt"
    return 0
  fi
  local ans=""
  read -r -p "$prompt [y/N] " ans || ans=""
  [[ "$ans" =~ ^[yY]$ ]]
}

# confirm with a literal phrase (e.g. "restore"). Always prompts.
confirm_phrase() {
  local prompt="$1" expected="$2"
  local ans=""
  read -r -p "$prompt (type '$expected' to confirm) " ans || ans=""
  [[ "$ans" == "$expected" ]]
}

abort() { log "ABORT: $*"; exit 1; }

ssh_remote() {
  ssh -o ServerAliveInterval=30 -o BatchMode=yes -o ConnectTimeout=15 "$REMOTE_HOST" "$@"
}

scp_remote() {
  scp -o ServerAliveInterval=30 -o BatchMode=yes -o ConnectTimeout=15 "$@"
}

# Run `docker compose <profiles> ...` on the remote in $REMOTE_PATH. PGNIZE_DOMAIN
# is exported so Caddy provisions TLS for the right host on every up/recreate
# (the Caddyfile falls back to localhost otherwise).
remote_compose() {
  ssh_remote "cd '$REMOTE_PATH' && PGNIZE_DOMAIN='$DOMAIN' docker compose $COMPOSE_BASE $REMOTE_PROFILE_ARGS $*"
}

# Read a single value from the prod .env without sourcing arbitrary shell.
remote_env_val() {
  ssh_remote "grep -E '^${1}=' '$REMOTE_PATH/.env' 2>/dev/null | tail -n1 | cut -d= -f2- | sed -e 's/^\"//' -e 's/\"\$//'" || true
}

record_state() {
  local key="$1" value="$2"
  printf '%s=%q\n' "$key" "$value" >> "$STATE_FILE"
}

# Cleanup runs on every exit. Safe to call before the lock is acquired.
LOCK_HELD=0
# Step 0b runtime-smoke resource names. Globals (not locals) so the EXIT trap
# can tear them down whether step 0b returned cleanly, aborted, or was Ctrl-C'd.
SMOKE_NET=""
SMOKE_DB_NAME=""
SMOKE_APP_NAME=""
cleanup() {
  local rc=$?
  if [[ -n "${TARBALL:-}" && -f "$TARBALL" ]]; then
    rm -f "$TARBALL" || true
  fi
  # Idempotent smoke teardown — docker rm -f on a missing container/network is a
  # no-op once stderr is silenced. Safe to call from any exit path.
  if [[ -n "$SMOKE_APP_NAME" || -n "$SMOKE_DB_NAME" ]]; then
    docker rm -f "$SMOKE_APP_NAME" "$SMOKE_DB_NAME" >/dev/null 2>&1 || true
  fi
  if [[ -n "$SMOKE_NET" ]]; then
    docker network rm "$SMOKE_NET" >/dev/null 2>&1 || true
  fi
  if [[ $LOCK_HELD -eq 1 ]]; then
    rm -f "$LOCK_FILE_LOCAL" || true
  fi
  if [[ $rc -ne 0 && $MODE == "deploy" ]]; then
    log "Deploy did not complete cleanly (exit=$rc). State preserved at $STATE_FILE — '$0 --rollback' is available."
  fi
}
trap cleanup EXIT

acquire_lock() {
  # Atomic lock acquisition: noclobber refuses to overwrite an existing file.
  if ! ( set -o noclobber; echo "$$" > "$LOCK_FILE_LOCAL" ) 2>/dev/null; then
    log "Deploy lock exists at $LOCK_FILE_LOCAL (pid $(cat "$LOCK_FILE_LOCAL" 2>/dev/null || echo unknown))"
    log "  If no other deploy is running, remove the stale lock with:"
    log "    rm $LOCK_FILE_LOCAL"
    exit 1
  fi
  LOCK_HELD=1
}

# Resolve the running api container id on prod (compose recreates it on roll,
# so resolve fresh each time rather than caching a name).
remote_api_container() {
  remote_compose "ps -q $APP_SERVICE" | tr -d '\r'
}

# Polls Docker's healthcheck on the api container until 'healthy' or timeout.
wait_for_health() {
  local deadline=$(( $(date +%s) + HEALTH_TIMEOUT_SECONDS ))
  log "Waiting for $APP_SERVICE to report healthy (timeout ${HEALTH_TIMEOUT_SECONDS}s)..."
  while [[ $(date +%s) -lt $deadline ]]; do
    local cid status
    cid=$(remote_api_container)
    if [[ -z "$cid" ]]; then
      printf '.' >&2; sleep 5; continue
    fi
    status=$(ssh_remote "docker inspect --format '{{.State.Health.Status}}' '$cid' 2>/dev/null || echo missing")
    status="${status//$'\r'/}"
    case "$status" in
      healthy)
        log "Container is healthy."
        return 0
        ;;
      unhealthy)
        log "Container reports unhealthy. Recent logs:"
        remote_compose "logs --tail=80 $APP_SERVICE" >&2 || true
        return 1
        ;;
      starting|missing|"")
        printf '.' >&2
        sleep 5
        ;;
      *)
        log "Unexpected health status '$status'; retrying."
        sleep 5
        ;;
    esac
  done
  echo >&2
  log "Healthcheck timed out after ${HEALTH_TIMEOUT_SECONDS}s. Last logs:"
  remote_compose "logs --tail=120 $APP_SERVICE" >&2 || true
  return 1
}

# Runs the api binary's own /healthz probe from inside the container
# (skips Caddy + DNS). Same probe docker-compose.yml uses for HEALTHCHECK.
check_health_internal() {
  log "Internal health check via '/app/api -healthcheck' (from inside the container)..."
  remote_compose "exec -T $APP_SERVICE /app/api -healthcheck"
}

# Hits the public URL from this machine (exercises DNS, Caddy, app).
check_health_external() {
  log "External health check against $EXTERNAL_HEALTH_URL (from this machine)..."
  curl --fail --silent --show-error --max-time 10 -o /dev/null "$EXTERNAL_HEALTH_URL"
}

# ─── Steps ──

step_0_preflight() {
  step 0 "Preflight (go vet, go test, go build) — local only"

  # 1. Working directory must be the canonical repo, not a worktree.
  if [[ "$(pwd)" != "$LOCAL_REPO" ]]; then
    abort "Run from $LOCAL_REPO (current: $(pwd))"
  fi

  # 2. Branch must be the deploy branch.
  local branch
  branch=$(git symbolic-ref --short HEAD 2>/dev/null || echo "")
  if [[ "$branch" != "$DEPLOY_BRANCH" ]]; then
    abort "Current branch is '$branch' — refusing to deploy non-$DEPLOY_BRANCH"
  fi

  # 3. Working tree must be clean.
  if [[ -n "$(git status --porcelain)" ]]; then
    git status --short >&2
    abort "Working tree has uncommitted changes — commit or stash first"
  fi

  # 4. SSH connectivity check (fail fast if keys aren't set up — saves the user
  #    from waiting through the build only to crash on ssh).
  log "Verifying ssh connectivity to $REMOTE_HOST ..."
  if ! ssh_remote 'echo ok' >/dev/null; then
    abort "Cannot reach $REMOTE_HOST via ssh (BatchMode is on — check ~/.ssh/config and key auth)"
  fi

  # 5. Read-only fetch with prune.
  log "git fetch origin --prune"
  git fetch origin --prune

  # 6. Lint — go vet across the workspace (override allowed). The web lint runs
  #    inside the authoritative docker build in step 0a, so preflight stays
  #    Go-only and node-dependency-free here.
  if ! ( go vet ./... && ( cd chesskit && go vet ./... ) ); then
    log "go vet failed."
    confirm "Continue anyway?" || abort "Cancelled at lint"
  fi

  # 7. Test — Go unit tests, no DB (override allowed).
  if ! make test-go; then
    log "go tests failed."
    confirm "Continue anyway?" || abort "Cancelled at test"
  fi

  # 8. Build the api binary (no override — must pass). The full prod recipe
  #    (api + web images) is built authoritatively in step 0a.
  log "Building the api binary to catch compile errors before the docker build..."
  go build -o bin/api ./cmd/api || abort "go build failed — fix before deploying"

  # 9. Capture commit + diff against origin/<branch>.
  DEPLOY_SHA=$(git rev-parse HEAD)
  record_state DEPLOY_SHA "$DEPLOY_SHA"
  local commit_count
  commit_count=$(git rev-list --count "origin/${DEPLOY_BRANCH}..HEAD")
  log "Commits to deploy ($commit_count):"
  git log --oneline "origin/${DEPLOY_BRANCH}..HEAD" >&2 || true

  if [[ $commit_count -eq 0 ]]; then
    log "Nothing to push — origin/${DEPLOY_BRANCH} already at $DEPLOY_SHA. Continuing in case prod is behind."
  fi
}

step_0a_docker_preflight() {
  step "0a" "Local Docker build (mirrors the prod build in step 6)"

  # Catches Dockerfile-recipe bugs (missing COPY, wrong base image, npm ci
  # failure, next build error, go build error) BEFORE prod sees them. Uses the
  # exact compose recipe step 6 runs on prod. No 'up' — the images built here
  # are reused by the runtime smoke in step 0b.
  if ! command -v docker >/dev/null 2>&1; then
    abort "Docker not installed locally. Install it or run preflight from a host that has it — refusing to deploy unverified."
  fi
  if ! docker info >/dev/null 2>&1; then
    abort "Local docker daemon is not running. Start it (e.g. 'sudo systemctl start docker') and re-run."
  fi

  log "Building '$APP_SERVICE' + '$WEB_SERVICE' images via 'docker compose --profile $COMPOSE_PROFILE build'..."
  log "  Cold build (no cache): several minutes. Warm build: under a minute."
  # NB: literal -f here (not $COMPOSE_BASE) — IFS=$'\n\t' above strips the space
  # as a word separator, so an unquoted "$COMPOSE_BASE" reaches docker as one arg.
  if ! docker compose -f docker-compose.yml --profile "$COMPOSE_PROFILE" build "$APP_SERVICE" "$WEB_SERVICE"; then
    log "Local docker build FAILED. The same recipe runs on prod in step 6."
    log "  Fix the Dockerfile or build context, then re-run preflight."
    abort "Local docker build failed — prod would have failed the same way at step 6"
  fi
  log "Local docker build OK — Dockerfile recipe is valid for prod."
}

step_0b_runtime_smoke() {
  step "0b" "Local container runtime smoke (boot, migrate, /healthz)"

  # Boots the api image built in step 0a against an ephemeral postgres on a
  # disposable docker network. Catches runtime bugs 'docker build' cannot see:
  # failing migration SQL, missing env that crashes startup, /healthz
  # regressions. ALWAYS tears down (success, failure, Ctrl-C).
  #
  # Synthetic env only — STORAGE_DRIVER=filesystem and RECOGNIZER=fake keep the
  # dependency surface to just postgres (no minio, no ollama/gemini), and no
  # prod credentials are read from the canonical clone's .env.
  local suffix
  suffix="preflight-$$"
  SMOKE_NET="pg-${suffix}-net"
  SMOKE_DB_NAME="pg-${suffix}-db"
  SMOKE_APP_NAME="pg-${suffix}-app"

  log "Creating ephemeral network $SMOKE_NET..."
  docker network create "$SMOKE_NET" >/dev/null 2>&1 || abort "Failed to create docker network $SMOKE_NET"

  log "Starting ephemeral postgres ($SMOKE_DB_NAME)..."
  if ! docker run -d --name "$SMOKE_DB_NAME" --network "$SMOKE_NET" \
        -e POSTGRES_USER=preflight \
        -e POSTGRES_PASSWORD=preflight \
        -e POSTGRES_DB=preflight \
        postgres:16-alpine >/dev/null 2>&1; then
    abort "Failed to start postgres container"
  fi

  # Wait for postgres to be FULLY initialized — a real 'SELECT 1' against the
  # named DB only succeeds once the entrypoint has finished initdb + restart.
  log "Waiting for postgres + preflight DB ready (cold init ~10-15s; max 60s)..."
  local pg_deadline=$(( $(date +%s) + 60 ))
  local pg_ready=0
  while [[ $(date +%s) -lt $pg_deadline ]]; do
    if docker exec "$SMOKE_DB_NAME" psql -U preflight -d preflight -c 'SELECT 1' >/dev/null 2>&1; then
      pg_ready=1; break
    fi
    sleep 2
  done
  if [[ $pg_ready -ne 1 ]]; then
    log "Postgres did not pass 'SELECT 1' within 60s. Container logs:"
    docker logs --tail 80 "$SMOKE_DB_NAME" >&2 || true
    abort "Postgres did not become ready"
  fi

  log "Starting api container — migrations run during startup..."
  # The api runs as the image's nonroot user and MkdirAll's STORAGE_DIR at boot,
  # so back it with a world-writable tmpfs rather than relying on the distroless
  # base's /tmp permissions.
  if ! docker run -d --name "$SMOKE_APP_NAME" --network "$SMOKE_NET" \
        --tmpfs /tmp/uploads:rw,mode=1777 \
        -e DATABASE_URL="postgres://preflight:preflight@${SMOKE_DB_NAME}:5432/preflight?sslmode=disable" \
        -e AUTH_SECRET="preflight-smoke-secret-not-real-32chars" \
        -e API_ADDR=":8080" \
        -e STORAGE_DRIVER="filesystem" \
        -e STORAGE_DIR="/tmp/uploads" \
        -e RECOGNIZER="fake" \
        "${IMAGE_API}:latest" >/dev/null 2>&1; then
    abort "Failed to start api container — is ${IMAGE_API}:latest present? (step 0a should build it)"
  fi

  # Poll /healthz via the binary's own probe — same one prod's HEALTHCHECK uses.
  log "Polling /healthz (timeout 90s — migrations apply during this window)..."
  local app_deadline=$(( $(date +%s) + 90 ))
  local ok=0
  while [[ $(date +%s) -lt $app_deadline ]]; do
    if [[ "$(docker inspect -f '{{.State.Status}}' "$SMOKE_APP_NAME" 2>/dev/null)" != "running" ]]; then
      log "Api container exited unexpectedly. Logs:"
      docker logs --tail 120 "$SMOKE_APP_NAME" >&2 || true
      abort "Api container exited during smoke — common cause: failing migration or missing env"
    fi
    if docker exec "$SMOKE_APP_NAME" /app/api -healthcheck >/dev/null 2>&1; then
      ok=1; break
    fi
    sleep 3
  done

  if [[ $ok -ne 1 ]]; then
    log "Api did not pass /healthz within 90s. Logs:"
    docker logs --tail 120 "$SMOKE_APP_NAME" >&2 || true
    abort "Local runtime smoke failed — /healthz never responded ok"
  fi
  log "Runtime smoke OK — container boots, migrations apply, /healthz responds."

  # Explicit success-path teardown; zero the globals so the EXIT trap is a no-op.
  docker rm -f "$SMOKE_APP_NAME" "$SMOKE_DB_NAME" >/dev/null 2>&1 || true
  docker network rm "$SMOKE_NET" >/dev/null 2>&1 || true
  SMOKE_APP_NAME=""; SMOKE_DB_NAME=""; SMOKE_NET=""

  confirm "Proceed with deploy of $DEPLOY_SHA?" || abort "Cancelled at preflight summary"
}

step_1_push() {
  step 1 "Push origin/$DEPLOY_BRANCH (off-host code backup)"
  if [[ "$(git rev-parse HEAD)" == "$(git rev-parse "origin/$DEPLOY_BRANCH" 2>/dev/null || echo none)" ]]; then
    log "origin/$DEPLOY_BRANCH is already at HEAD — nothing to push."
    return 0
  fi
  confirm "git push origin $DEPLOY_BRANCH?" || abort "Cancelled at push"
  git push origin "$DEPLOY_BRANCH"
}

step_2_capture_state() {
  step 2 "Capture pre-deploy state on prod"

  # Ensure git trusts the prod repo before any git call. A prior deploy's tar
  # extraction can leave the tree owned by the local build uid (e.g. 1000) while
  # git here runs as root, which aborts every git command with "dubious
  # ownership". Mark it safe up front (idempotent), not just after extraction.
  ssh_remote "git config --global --get-all safe.directory 2>/dev/null | grep -qxF '$REMOTE_PATH' || git config --global --add safe.directory '$REMOTE_PATH'"

  PREV_REMOTE_SHA=$(ssh_remote "cd '$REMOTE_PATH' && git rev-parse HEAD")
  PREV_REMOTE_SHA="${PREV_REMOTE_SHA//$'\r'/}"
  [[ -n "$PREV_REMOTE_SHA" ]] || abort "Empty git HEAD on prod — repo may be detached or broken"
  log "Prod is currently at $PREV_REMOTE_SHA"

  if [[ "$PREV_REMOTE_SHA" == "$DEPLOY_SHA" ]]; then
    abort "Prod is already at $DEPLOY_SHA — nothing to deploy. (Recovering from a partial deploy? Reset the prod git tree manually first.)"
  fi

  # Determine the recognizer backend from the prod .env so we know whether the
  # ollama (vlm) profile must be brought up and a model pulled in step 6.
  [[ -n "$(ssh_remote "test -f '$REMOTE_PATH/.env' && echo yes || true")" ]] \
    || abort "No .env on prod ($REMOTE_PATH/.env) — copy .env.example and fill in production secrets first"
  RECOGNIZER="$(remote_env_val RECOGNIZER)"; RECOGNIZER="${RECOGNIZER:-ollama}"
  log "Prod RECOGNIZER=$RECOGNIZER"
  case "$RECOGNIZER" in
    ollama)
      REMOTE_PROFILE_ARGS="--profile $COMPOSE_PROFILE --profile vlm"
      ;;
    gemini)
      [[ -n "$(remote_env_val GEMINI_API_KEY)" ]] \
        || abort "RECOGNIZER=gemini but GEMINI_API_KEY is empty in the prod .env"
      ;;
    fake)
      log "WARN: RECOGNIZER=fake on prod — the deterministic stub is not suitable for real traffic."
      ;;
    *)
      log "WARN: unrecognized RECOGNIZER=$RECOGNIZER; using prod-only profiles."
      ;;
  esac
  record_state RECOGNIZER "$RECOGNIZER"

  # Warn if prod has uncommitted local edits — tar extract would overwrite them.
  local remote_dirty
  remote_dirty=$(ssh_remote "cd '$REMOTE_PATH' && git status --porcelain")
  if [[ -n "$remote_dirty" ]]; then
    log "WARNING: prod working tree has local modifications:"
    echo "$remote_dirty" | sed 's/^/    /' >&2
    confirm "Discard prod-side changes by extracting over them?" --always-prompt \
      || abort "Cancelled at prod dirty-tree check"
  fi

  PREV_IMAGE_API_ID=$(ssh_remote "docker images -q '$IMAGE_API:latest'"); PREV_IMAGE_API_ID="${PREV_IMAGE_API_ID//$'\r'/}"
  PREV_IMAGE_WEB_ID=$(ssh_remote "docker images -q '$IMAGE_WEB:latest'"); PREV_IMAGE_WEB_ID="${PREV_IMAGE_WEB_ID//$'\r'/}"
  log "Current prod images: api=${PREV_IMAGE_API_ID:-<none>} web=${PREV_IMAGE_WEB_ID:-<none>}"
  [[ -n "$PREV_IMAGE_API_ID" && -n "$PREV_IMAGE_WEB_ID" ]] \
    || log "WARN: a prior image is missing — image (code-only) rollback may be incomplete."

  record_state PREV_REMOTE_SHA    "$PREV_REMOTE_SHA"
  record_state PREV_IMAGE_API_ID  "$PREV_IMAGE_API_ID"
  record_state PREV_IMAGE_WEB_ID  "$PREV_IMAGE_WEB_ID"
}

step_2a_resource_check() {
  step "2a" "Prod resource check (RAM / CPU / disk)"

  local lines
  lines=$(ssh_remote "bash -s" <<'REMOTE_EOF'
set -u
df --output=avail / 2>/dev/null | tail -n+2
df --output=avail /var/lib/docker 2>/dev/null | tail -n+2 || df --output=avail / 2>/dev/null | tail -n+2
awk '/MemAvailable/ {printf "%.0f\n", $2/1024; exit}' /proc/meminfo
awk '/MemTotal/    {printf "%.0f\n", $2/1024; exit}' /proc/meminfo
awk '{print $1; exit}' /proc/loadavg
nproc
REMOTE_EOF
  )

  local disk_root_kb="" disk_docker_kb="" mem_avail_mb="" mem_total_mb="" load_1m="" ncpu=""
  {
    read -r disk_root_kb || true
    read -r disk_docker_kb || true
    read -r mem_avail_mb || true
    read -r mem_total_mb || true
    read -r load_1m || true
    read -r ncpu || true
  } <<<"$lines"

  if [[ -z "$disk_root_kb" || -z "$disk_docker_kb" || -z "$mem_avail_mb" || -z "$mem_total_mb" || -z "$load_1m" || -z "$ncpu" ]]; then
    log "Resource probe returned incomplete data:"
    log "  disk_root_kb='$disk_root_kb' disk_docker_kb='$disk_docker_kb' mem_avail_mb='$mem_avail_mb' mem_total_mb='$mem_total_mb' load_1m='$load_1m' ncpu='$ncpu'"
    echo "$lines" | sed 's/^/    /' >&2
    abort "Could not parse all resource info from prod"
  fi

  local disk_root_gb disk_docker_gb load_per_cpu
  disk_root_gb=$(awk -v kb="$disk_root_kb" 'BEGIN{printf "%.1f", kb/1024/1024}')
  disk_docker_gb=$(awk -v kb="$disk_docker_kb" 'BEGIN{printf "%.1f", kb/1024/1024}')
  load_per_cpu=$(awk -v l="$load_1m" -v n="$ncpu" 'BEGIN{printf "%.2f", l/n}')

  log "  / free:           ${disk_root_gb} GB"
  log "  /var/lib/docker:  ${disk_docker_gb} GB"
  log "  MemAvailable:     ${mem_avail_mb} MB / ${mem_total_mb} MB total"
  log "  Load (1m / cpus): ${load_1m} / ${ncpu}  (per-cpu ${load_per_cpu})"

  log "  Live container stats (5s window):"
  ssh_remote "docker stats --no-stream --format 'table {{.Name}}\\t{{.CPUPerc}}\\t{{.MemUsage}}' 2>/dev/null | head -12" >&2 || true
  log "  backups dir size:"
  ssh_remote "du -sh '$REMOTE_PATH/backups' 2>/dev/null" >&2 || true

  local errors=() warnings=()

  if awk -v g="$disk_root_gb" -v t="$DISK_ROOT_ABORT_GB" 'BEGIN{exit !(g<t)}'; then
    errors+=("/ has only ${disk_root_gb} GB free (< ${DISK_ROOT_ABORT_GB} GB hard limit). Reclaim with 'journalctl --vacuum-time=2d' or 'apt clean'.")
  fi
  if awk -v g="$disk_docker_gb" -v t="$DISK_DOCKER_ABORT_GB" 'BEGIN{exit !(g<t)}'; then
    errors+=("/var/lib/docker has only ${disk_docker_gb} GB free (< ${DISK_DOCKER_ABORT_GB} GB hard limit). Reclaim with 'docker system prune -af'.")
  fi
  if awk -v m="$mem_avail_mb" -v t="$MEM_AVAIL_ABORT_MB" 'BEGIN{exit !(m<t)}'; then
    errors+=("MemAvailable ${mem_avail_mb} MB (< ${MEM_AVAIL_ABORT_MB} MB hard limit). The build will OOM. Free memory before deploying.")
  fi

  if [[ ${#errors[@]} -gt 0 ]]; then
    for e in "${errors[@]}"; do log "ERROR: $e"; done
    abort "Resource hard limits violated; cannot deploy safely"
  fi

  if awk -v g="$disk_root_gb" -v t="$DISK_ROOT_WARN_GB" 'BEGIN{exit !(g<t)}'; then
    warnings+=("/ has ${disk_root_gb} GB free (< ${DISK_ROOT_WARN_GB} GB recommended).")
  fi
  if awk -v g="$disk_docker_gb" -v t="$DISK_DOCKER_WARN_GB" 'BEGIN{exit !(g<t)}'; then
    warnings+=("/var/lib/docker has ${disk_docker_gb} GB free (< ${DISK_DOCKER_WARN_GB} GB recommended).")
  fi
  if awk -v m="$mem_avail_mb" -v t="$MEM_AVAIL_WARN_MB" 'BEGIN{exit !(m<t)}'; then
    warnings+=("MemAvailable ${mem_avail_mb} MB (< ${MEM_AVAIL_WARN_MB} MB recommended).")
  fi
  if awk -v p="$load_per_cpu" -v t="$LOAD_PER_CPU_WARN" 'BEGIN{exit !(p>t)}'; then
    warnings+=("Per-cpu load ${load_per_cpu} (> ${LOAD_PER_CPU_WARN}). Build will be slow.")
  fi

  if [[ ${#warnings[@]} -gt 0 ]]; then
    log "Resource warnings:"
    for w in "${warnings[@]}"; do log "  WARN: $w"; done
    confirm "Continue despite warnings?" || abort "Cancelled at resource check"
  fi

  record_state PROD_DISK_ROOT_GB    "$disk_root_gb"
  record_state PROD_DISK_DOCKER_GB  "$disk_docker_gb"
  record_state PROD_MEM_AVAIL_MB    "$mem_avail_mb"
  record_state PROD_LOAD_1M         "$load_1m"
}

step_3_backup() {
  step 3 "DB backup on prod"

  # pgnize's db-backup container runs an inline daily loop rather than a
  # standalone script, so trigger a one-off dump explicitly. POSTGRES_* are
  # read from the container's own env (escaped so they expand container-side).
  BACKUP_FILE="pgnize-deploy-$(date +%Y%m%d-%H%M%S).sql.gz"
  log "About to run pg_dump inside the $DB_BACKUP_SERVICE container -> /backups/$BACKUP_FILE"
  log "(Lands in $REMOTE_PATH/backups, subject to the container's 30-day retention.)"
  confirm "Proceed with backup?" || abort "Cancelled at backup"

  log "Running backup..."
  remote_compose "exec -T $DB_BACKUP_SERVICE sh -c 'PGPASSWORD=\"\$POSTGRES_PASSWORD\" pg_dump -h $DB_SERVICE -U \"\$POSTGRES_USER\" \"\$POSTGRES_DB\" | gzip > \"/backups/$BACKUP_FILE\"'" \
    || abort "pg_dump failed"

  # Verify on the host (./backups is a bind mount, so the file is directly visible).
  if ! ssh_remote "test -s '$REMOTE_PATH/backups/$BACKUP_FILE'"; then
    abort "Backup file $REMOTE_PATH/backups/$BACKUP_FILE is empty or missing"
  fi
  # Integrity: a truncated/corrupt gzip is worse than no backup.
  ssh_remote "gunzip -t '$REMOTE_PATH/backups/$BACKUP_FILE'" || abort "Backup gzip failed integrity check"
  local size
  size=$(ssh_remote "du -h '$REMOTE_PATH/backups/$BACKUP_FILE' | cut -f1"); size="${size//$'\r'/}"
  log "Backup OK: /backups/$BACKUP_FILE ($size)"

  record_state BACKUP_FILE "$BACKUP_FILE"
}

step_4_transfer() {
  step 4 "Code transfer (tarball + scp + extract)"

  TARBALL="/tmp/pgnize-deploy-${DEPLOY_SHA:0:12}.tar.gz"
  log "Building tarball at $TARBALL ..."
  ( cd "$LOCAL_REPO" && tar czf "$TARBALL" \
      --exclude='./node_modules' \
      --exclude='./web/node_modules' \
      --exclude='./e2e/node_modules' \
      --exclude='./web/.next' \
      --exclude='./web/out' \
      --exclude='./bin' \
      --exclude='./data' \
      --exclude='./backups' \
      --exclude='./coverage' \
      --exclude='./.claude/worktrees' \
      --exclude='./.env' \
      --exclude='./.env.*' \
      --exclude='./e2e/playwright-report' \
      --exclude='./e2e/test-results' \
      . )
  local size
  size=$(du -h "$TARBALL" | awk '{print $1}')
  log "Tarball: $TARBALL ($size)"
  confirm "scp tarball to $REMOTE_HOST:/tmp/?" || abort "Cancelled at transfer"

  scp_remote "$TARBALL" "$REMOTE_HOST:/tmp/"

  log "Backing up prod .env defensively (.env.deploy-backup)..."
  ssh_remote "test -f '$REMOTE_PATH/.env' && cp '$REMOTE_PATH/.env' '$REMOTE_PATH/.env.deploy-backup' || true"

  log "Extracting on prod (.env preserved by tarball excludes)..."
  ssh_remote "tar xzf '/tmp/$(basename "$TARBALL")' -C '$REMOTE_PATH'"

  # The tarball is built locally (uid 1000) and tar restores that uid on prod, so
  # after extraction the repo (and .git) is owned by 1000 while git here runs as
  # root -> "dubious ownership", which aborts every later git call. Mark the path
  # safe (idempotent: only add when missing, so repeated deploys don't duplicate).
  log "Marking prod repo as a git safe.directory ..."
  ssh_remote "git config --global --get-all safe.directory 2>/dev/null | grep -qxF '$REMOTE_PATH' || git config --global --add safe.directory '$REMOTE_PATH'"

  # `git reset --hard HEAD` repairs anything tar missed by re-checking out from
  # the just-extracted objects. `git clean -fd` then removes untracked files
  # that lingered from the previous commit. CRUCIAL: exclude ./backups — the DB
  # dumps there are untracked (bind mount, NOT gitignored) and git clean would
  # otherwise wipe them, including the dump we just made in step 3. Gitignored
  # files (.env, web/.next, node_modules, bin, data/uploads) are kept by default.
  log "Repairing working tree to match HEAD ..."
  ssh_remote "cd '$REMOTE_PATH' && git reset --hard HEAD"
  log "Cleaning untracked files on prod (backups + gitignored files kept)..."
  ssh_remote "cd '$REMOTE_PATH' && git clean -fd -e backups"

  local remote_sha
  remote_sha=$(ssh_remote "cd '$REMOTE_PATH' && git rev-parse HEAD"); remote_sha="${remote_sha//$'\r'/}"
  if [[ "$remote_sha" != "$DEPLOY_SHA" ]]; then
    abort "Post-transfer prod HEAD is $remote_sha, expected $DEPLOY_SHA"
  fi
  log "Prod git HEAD now $remote_sha — matches local"

  local post_dirty
  post_dirty=$(ssh_remote "cd '$REMOTE_PATH' && git status --porcelain")
  if [[ -n "$post_dirty" ]]; then
    log "WARNING: prod working tree differs from $DEPLOY_SHA after extract:"
    echo "$post_dirty" | sed 's/^/    /' >&2
    confirm "Continue anyway? (build will use the extracted files as-is)" \
      || abort "Cancelled at post-extract diff check"
  fi

  log "Cleaning up tarball..."
  ssh_remote "rm -f '/tmp/$(basename "$TARBALL")'"
  rm -f "$TARBALL"; TARBALL=""
}

# Tag one current image as a timestamped rollback tag. Echoes the tag on stdout.
tag_one_image() {
  local image="$1" prev_id="$2"
  if [[ -z "$prev_id" ]]; then
    log "No previous $image image — skipping its rollback tag."
    return 0
  fi
  local tag="${image}:rollback-$(date +%Y%m%d-%H%M%S)-${PREV_REMOTE_SHA:0:8}"
  log "Tagging $prev_id as $tag ..."
  ssh_remote "docker tag '$prev_id' '$tag'"
  ssh_remote "docker image inspect '$tag' >/dev/null 2>&1" \
    || abort "docker tag claimed success but $tag is not present"
  echo "$tag"
}

step_5_tag_image() {
  step 5 "Tag current images for rollback (api + web)"

  if [[ -z "$PREV_IMAGE_API_ID" && -z "$PREV_IMAGE_WEB_ID" ]]; then
    log "No previous images — skipping tag. Code-only rollback unavailable."
    return 0
  fi
  confirm "Tag the current api/web images for rollback?" || abort "Cancelled at image tag"
  ROLLBACK_TAG_API=$(tag_one_image "$IMAGE_API" "$PREV_IMAGE_API_ID")
  ROLLBACK_TAG_WEB=$(tag_one_image "$IMAGE_WEB" "$PREV_IMAGE_WEB_ID")
  [[ -n "${ROLLBACK_TAG_API:-}" ]] && record_state ROLLBACK_TAG_API "$ROLLBACK_TAG_API"
  [[ -n "${ROLLBACK_TAG_WEB:-}" ]] && record_state ROLLBACK_TAG_WEB "$ROLLBACK_TAG_WEB"
}

step_6_build_and_roll() {
  step 6 "Rebuild and roll the app"

  confirm "Build new images on prod (docker compose build $APP_SERVICE $WEB_SERVICE)?" || abort "Cancelled before build"
  if ! remote_compose "build $APP_SERVICE $WEB_SERVICE"; then
    log "Build failed. The previous containers are still running unchanged."
    [[ -n "${ROLLBACK_TAG_API:-}" ]] && log "Rollback tags preserved — '$0 --rollback' is available."
    abort "Build failed"
  fi

  # The caddy service joins the shared external 'edge' network so the
  # swiss-manager edge Caddy can reverse-proxy into this stack. Create it if the
  # host doesn't have it yet (idempotent; the swiss-manager stack joins the same
  # network).
  log "Ensuring the shared 'edge' docker network exists on prod..."
  ssh_remote "docker network inspect edge >/dev/null 2>&1 || docker network create edge" >/dev/null

  # Bring up data services first, then (optionally) ollama + model, then the app.
  log "Starting data services (db, minio)..."
  remote_compose "up -d $DB_SERVICE minio"

  if [[ "$RECOGNIZER" == "ollama" ]]; then
    log "Starting ollama (vlm profile)..."
    remote_compose "up -d ollama"
    local model
    model="$(remote_env_val RECOGNIZER_MODEL)"; model="${model:-minicpm-v}"
    if ! remote_compose "exec -T ollama ollama list" 2>/dev/null | grep -q "$model"; then
      log "Pulling ollama model '$model' (may download several GB)..."
      remote_compose "exec -T ollama ollama pull '$model'" || abort "Failed to pull ollama model '$model'"
    else
      log "ollama model '$model' already present."
    fi
  fi

  confirm "Roll the app (docker compose up -d)? Migrations run on api start." || abort "Cancelled before roll"
  # Roll the full stack so caddy/web/db-backup pick up any env or image changes.
  remote_compose "up -d"

  if ! wait_for_health; then
    log "Container did not reach healthy state."
    if confirm "Rollback now (code-only)?" --always-prompt; then
      rollback_code_only
      exit 1
    fi
    abort "Healthcheck timeout; rollback declined"
  fi
}

step_7_health_and_smoke() {
  step 7 "Health checks + smoke prompt"

  if ! check_health_internal; then
    log "Internal health check failed."
    if confirm "Rollback now (code-only)?" --always-prompt; then
      rollback_code_only
      exit 1
    fi
    abort "Internal /healthz failed; rollback declined"
  fi
  log "Internal /healthz OK"

  if ! check_health_external; then
    log "External health check failed (could be DNS/Caddy/TLS, not just the app)."
    confirm "Continue anyway? (the app may still be reachable directly)" --always-prompt || {
      if confirm "Rollback now (code-only)?" --always-prompt; then
        rollback_code_only
        exit 1
      fi
      abort "External health failed; rollback declined"
    }
  else
    log "External $EXTERNAL_HEALTH_URL OK"
  fi

  cat >&2 <<CHECKLIST

────────────────────────────────────────────────────────────────
  MANUAL SMOKE TEST — do these in your browser at https://$DOMAIN:
    1. Log in (or register a test account).
    2. Upload a score-sheet image.
    3. Trigger recognition and wait for the job to finish.
    4. Open the recognized game and verify moves / confidence render.
    5. Export the PGN and confirm it downloads.
────────────────────────────────────────────────────────────────

Choose:
  [c]ontinue    deploy succeeded
  [r]ollback    code-only rollback (DB stays on new schema)
  [a]bort       leave new code running, exit script

CHECKLIST
  local choice=""
  if [[ $ASSUME_YES -eq 1 ]]; then
    log "AUTO-YES: smoke checklist (treating as [c]ontinue)."
    choice="c"
  else
    read -r -p "Choice: " choice || choice=""
  fi
  case "$choice" in
    c|C)
      record_state DEPLOY_RESULT success
      log "Deploy successful: $DEPLOY_SHA"
      ;;
    r|R)
      rollback_code_only
      if confirm "Also restore DB from $BACKUP_FILE? (DESTRUCTIVE)" --always-prompt; then
        rollback_code_and_db
      fi
      exit 0
      ;;
    a|A)
      log "Aborted at smoke prompt — new code is running. Use '$0 --rollback' if needed."
      exit 0
      ;;
    *)
      log "Unrecognized choice; treating as abort. New code is running. Use '$0 --rollback' if needed."
      exit 0
      ;;
  esac
}

step_8_post_deploy_cleanup() {
  step 8 "Post-deploy cleanup (rollback tags + build cache)"

  # Only reached on the [c]ontinue path in step 7, so we never delete an image
  # we are about to need. Keeps the $KEEP_ROLLBACK_TAGS most recent rollback
  # tags PER image (api + web); prunes dangling images and bounds the build
  # cache. Volumes (pg_data, minio_data, ollama_models) are NEVER touched.
  log "Disk + docker usage BEFORE cleanup:"
  ssh_remote "df -h / | tail -1; docker system df" >&2 || true

  local image to_delete
  for image in "$IMAGE_API" "$IMAGE_WEB"; do
    to_delete=$(ssh_remote "docker images '$image' --filter 'reference=$image:rollback-*' --format '{{.CreatedAt}}\t{{.Repository}}:{{.Tag}}' | sort -r | tail -n +$((KEEP_ROLLBACK_TAGS + 1)) | cut -f2")
    to_delete="${to_delete//$'\r'/}"
    if [[ -n "$to_delete" ]]; then
      log "$image rollback tags to remove (keeping $KEEP_ROLLBACK_TAGS most recent):"
      echo "$to_delete" | sed 's/^/    /' >&2
    fi
  done
  log "Build cache will be trimmed to at most ${BUILD_CACHE_MAX_GB}GB (least-recently-used evicted first)."

  if ! confirm "Proceed with cleanup?"; then
    log "Skipped post-deploy cleanup. Run '$0 --cleanup' later to reclaim disk."
    return 0
  fi

  for image in "$IMAGE_API" "$IMAGE_WEB"; do
    to_delete=$(ssh_remote "docker images '$image' --filter 'reference=$image:rollback-*' --format '{{.CreatedAt}}\t{{.Repository}}:{{.Tag}}' | sort -r | tail -n +$((KEEP_ROLLBACK_TAGS + 1)) | cut -f2")
    to_delete="${to_delete//$'\r'/}"
    if [[ -n "$to_delete" ]]; then
      printf '%s\n' "$to_delete" | ssh_remote "xargs -r -n1 docker rmi" >&2 \
        || log "WARN: one or more 'docker rmi' calls failed for $image; continuing."
    fi
  done

  ssh_remote "docker image prune -f" >&2 || true
  ssh_remote "docker builder prune -f --max-used-space '${BUILD_CACHE_MAX_GB}GB'" >&2 || true

  log "Disk + docker usage AFTER cleanup:"
  ssh_remote "df -h / | tail -1; docker system df" >&2 || true
}

# ─── Rollback ──

rollback_code_only() {
  step "R" "Rollback (code-only)"

  if [[ -z "${PREV_REMOTE_SHA:-}" ]]; then
    abort "Cannot run code-only rollback: missing PREV_REMOTE_SHA in state"
  fi
  if [[ -z "${ROLLBACK_TAG_API:-}" && -z "${ROLLBACK_TAG_WEB:-}" ]]; then
    abort "Cannot run code-only rollback: no rollback image tags in state"
  fi

  [[ -n "${ROLLBACK_TAG_API:-}" ]] && { log "Re-tagging $ROLLBACK_TAG_API as $IMAGE_API:latest ..."; ssh_remote "docker tag '$ROLLBACK_TAG_API' '$IMAGE_API:latest'"; }
  [[ -n "${ROLLBACK_TAG_WEB:-}" ]] && { log "Re-tagging $ROLLBACK_TAG_WEB as $IMAGE_WEB:latest ..."; ssh_remote "docker tag '$ROLLBACK_TAG_WEB' '$IMAGE_WEB:latest'"; }

  log "Checking out previous commit on prod ($PREV_REMOTE_SHA) ..."
  ssh_remote "cd '$REMOTE_PATH' && git checkout '$PREV_REMOTE_SHA'"

  log "Recreating app containers ..."
  remote_compose "up -d"

  if ! wait_for_health; then
    abort "Rollback container did not become healthy — manual intervention required"
  fi
  check_health_internal || abort "Rollback /healthz failed — manual intervention required"
  log "Rollback complete. Migrations from this deploy remain applied (additive — old code tolerates them)."
  log "To roll the schema back too, re-run '$0 --rollback' and answer yes to the DB restore prompt."
}

rollback_code_and_db() {
  step "R+DB" "Rollback (code + DB restore)"

  [[ -n "${BACKUP_FILE:-}" ]] || abort "BACKUP_FILE missing from state — cannot restore DB"

  log "DESTRUCTIVE: restoring DB from /backups/$BACKUP_FILE — all data created since deploy will be LOST."
  confirm_phrase "Proceed?" "restore" || abort "Cancelled at DB restore"

  log "Verifying gzip integrity of /backups/$BACKUP_FILE ..."
  if ! ssh_remote "gunzip -t '$REMOTE_PATH/backups/$BACKUP_FILE'"; then
    abort "Backup file is corrupt — refusing to drop schema. Pick another backup or restore manually."
  fi

  log "Stopping app ..."
  remote_compose "stop $APP_SERVICE"

  log "Dropping public schema ..."
  ssh_remote "set -e; cd '$REMOTE_PATH' && PGNIZE_DOMAIN='$DOMAIN' docker compose $COMPOSE_BASE $REMOTE_PROFILE_ARGS exec -T '$DB_SERVICE' sh -c 'psql -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"DROP SCHEMA public CASCADE; CREATE SCHEMA public;\"'"

  log "Restoring from $BACKUP_FILE ..."
  if ! ssh_remote "set -e -o pipefail; cd '$REMOTE_PATH' && PGNIZE_DOMAIN='$DOMAIN' docker compose $COMPOSE_BASE $REMOTE_PROFILE_ARGS exec -T '$DB_BACKUP_SERVICE' gunzip -c '/backups/$BACKUP_FILE' | PGNIZE_DOMAIN='$DOMAIN' docker compose $COMPOSE_BASE $REMOTE_PROFILE_ARGS exec -T '$DB_SERVICE' sh -c 'psql -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -1'"; then
    log "RESTORE FAILED. The DB may be in a partial state. Manual intervention required."
    log "  - Public schema was dropped above; the dump pipe failed during restore."
    log "  - Re-attempt by re-running '$0 --rollback' (it re-drops and re-restores)."
    abort "DB restore failed mid-pipe"
  fi

  # Sanity check: a well-known table exists post-restore.
  if ! ssh_remote "cd '$REMOTE_PATH' && PGNIZE_DOMAIN='$DOMAIN' docker compose $COMPOSE_BASE $REMOTE_PROFILE_ARGS exec -T '$DB_SERVICE' sh -c 'psql -tA -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"SELECT 1 FROM information_schema.tables WHERE table_schema='\\''public'\\'' AND table_name='\\''users'\\''\"' | grep -q 1"; then
    abort "Restore completed but the users table is not present — restore is suspect"
  fi
  log "DB restore verified (users table present)."

  [[ -n "${ROLLBACK_TAG_API:-}" ]] && { log "Re-tagging $ROLLBACK_TAG_API as $IMAGE_API:latest ..."; ssh_remote "docker tag '$ROLLBACK_TAG_API' '$IMAGE_API:latest'"; }
  [[ -n "${ROLLBACK_TAG_WEB:-}" ]] && { log "Re-tagging $ROLLBACK_TAG_WEB as $IMAGE_WEB:latest ..."; ssh_remote "docker tag '$ROLLBACK_TAG_WEB' '$IMAGE_WEB:latest'"; }
  if [[ -n "${PREV_REMOTE_SHA:-}" ]]; then
    log "Checking out previous commit on prod ($PREV_REMOTE_SHA) ..."
    ssh_remote "cd '$REMOTE_PATH' && git checkout '$PREV_REMOTE_SHA'"
  fi

  log "Starting app ..."
  remote_compose "up -d"

  if ! wait_for_health; then
    abort "Container did not become healthy after DB restore — manual intervention required"
  fi
  check_health_internal || abort "Internal /healthz failed after DB restore"
  log "Code + DB rollback complete."
}

# ─── Disk cleanup (standalone mode) ──

step_cleanup_disk() {
  step "C" "Reclaim disk on $REMOTE_HOST"

  log "Verifying ssh connectivity to $REMOTE_HOST ..."
  ssh_remote 'echo ok' >/dev/null || abort "Cannot reach $REMOTE_HOST via ssh"

  log "Disk + Docker usage BEFORE:"
  ssh_remote "df -h / && echo && docker system df" >&2 || true

  # docker system prune -a removes all stopped containers and all unused images.
  # Volumes are NOT touched, so pg_data/minio_data/ollama_models stay intact.
  confirm "Run 'docker system prune -a -f' on $REMOTE_HOST? (keeps volumes — DB safe)" --always-prompt \
    || abort "Cancelled at docker system prune"
  ssh_remote "docker system prune -a -f" >&2

  if confirm "Also run 'docker builder prune -a -f'? Next build will be slow once." --always-prompt; then
    ssh_remote "docker builder prune -a -f" >&2
  else
    log "Skipped builder prune."
  fi

  log "Disk + Docker usage AFTER:"
  ssh_remote "df -h / && echo && docker system df" >&2 || true

  log "Cleanup complete."
}

# ─── Entry ──

main() {
  acquire_lock

  if [[ $MODE == "cleanup" ]]; then
    step_cleanup_disk
    return 0
  fi

  if [[ $MODE == "rollback" ]]; then
    if [[ ! -f "$STATE_FILE" ]]; then
      abort "No state file at $STATE_FILE — manual rollback required (retag images, git checkout previous SHA, restart compose)"
    fi
    # shellcheck disable=SC1090
    source "$STATE_FILE"
    # Re-derive remote profiles from the recorded recognizer so remote_compose
    # targets the same services this deploy brought up.
    RECOGNIZER="${RECOGNIZER:-ollama}"
    [[ "$RECOGNIZER" == "ollama" ]] && REMOTE_PROFILE_ARGS="--profile $COMPOSE_PROFILE --profile vlm"
    log "Loaded state from $STATE_FILE"
    log "  PREV_REMOTE_SHA  = ${PREV_REMOTE_SHA:-<unset>}"
    log "  ROLLBACK_TAG_API = ${ROLLBACK_TAG_API:-<unset>}"
    log "  ROLLBACK_TAG_WEB = ${ROLLBACK_TAG_WEB:-<unset>}"
    log "  BACKUP_FILE      = ${BACKUP_FILE:-<unset>}"
    log "  DEPLOY_SHA       = ${DEPLOY_SHA:-<unset>}"

    confirm_phrase "Rollback the prod stack to $PREV_REMOTE_SHA?" "rollback" || abort "Cancelled rollback"
    rollback_code_only
    if confirm "Also restore DB from $BACKUP_FILE? (DESTRUCTIVE)" --always-prompt; then
      rollback_code_and_db
    fi
    return 0
  fi

  # Fresh deploy: clear state file
  : > "$STATE_FILE"

  step_0_preflight
  step_0a_docker_preflight
  step_0b_runtime_smoke
  step_1_push
  step_2_capture_state
  step_2a_resource_check
  step_3_backup
  step_4_transfer
  step_5_tag_image
  step_6_build_and_roll
  step_7_health_and_smoke
  step_8_post_deploy_cleanup
}

main "$@"
