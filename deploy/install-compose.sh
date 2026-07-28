#!/usr/bin/env bash
set -euo pipefail

THISM_GITHUB_REPO="${THISM_GITHUB_REPO:-jsllxx77/thism}"
THISM_REF="${THISM_REF:-main}"
THISM_IMAGE="${THISM_IMAGE:-ghcr.io/jsllxx77/thism:latest}"
THISM_INSTALL_DIR="${THISM_INSTALL_DIR:-$HOME/thism-deploy}"
THISM_PORT="${THISM_PORT:-8080}"
THISM_ADMIN_USER="${THISM_ADMIN_USER:-admin}"
THISM_TOKEN="${THISM_TOKEN:-}"
THISM_ADMIN_PASS="${THISM_ADMIN_PASS:-}"
THISM_TOKEN_SECRET_FILE="${THISM_TOKEN_SECRET_FILE:-./secrets/thism_token}"
THISM_ADMIN_USER_SECRET_FILE="${THISM_ADMIN_USER_SECRET_FILE:-./secrets/thism_admin_user}"
THISM_ADMIN_PASS_SECRET_FILE="${THISM_ADMIN_PASS_SECRET_FILE:-./secrets/thism_admin_pass}"
THISM_GEOIP_HOST_DIR="${THISM_GEOIP_HOST_DIR:-/var/lib/thism/geo}"

RAW_BASE="https://raw.githubusercontent.com/${THISM_GITHUB_REPO}/${THISM_REF}/deploy"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

random_secret() {
  od -An -N16 -tx1 /dev/urandom | tr -d ' \n'
}

write_secret_file() {
  local path="$1"
  local value="$2"
  local label="$3"
  local dir
  local created=0

  if [ -z "$path" ]; then
    echo "$label secret file path is required" >&2
    exit 1
  fi

  dir="$(dirname "$path")"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    chmod 700 "$dir"
  fi

  if [ ! -s "$path" ]; then
    printf '%s\n' "$value" >"$path"
    created=1
  fi

  case "$path" in
    /*)
      if [ "$created" -eq 1 ]; then
        chmod 444 "$path"
      fi
      ;;
    *) chmod 444 "$path" ;;
  esac
}

display_secret_path() {
  case "$1" in
    /*) printf '%s\n' "$1" ;;
    *) printf '%s/%s\n' "$THISM_INSTALL_DIR" "$1" ;;
  esac
}

resolve_host_path() {
  case "$1" in
    /*) printf '%s\n' "$1" ;;
    ./*) printf '%s/%s\n' "$THISM_INSTALL_DIR" "${1#./}" ;;
    *) printf '%s/%s\n' "$THISM_INSTALL_DIR" "$1" ;;
  esac
}

prepare_geoip_dir() {
  local configured_path="$1"
  local host_path
  local identity
  local container_uid
  local container_gid

  case "/$configured_path/" in
    *"/../"*) echo "Refusing GeoIP host directory with a parent traversal: $configured_path" >&2; exit 1 ;;
  esac

  host_path="$(resolve_host_path "$configured_path")"
  case "$host_path" in
    ""|/) echo "Refusing to use an unsafe GeoIP host directory: $host_path" >&2; exit 1 ;;
  esac

  identity="$(docker run --rm --entrypoint sh "$THISM_IMAGE" -c 'printf "%s:%s\n" "$(id -u)" "$(id -g)"')"
  case "$identity" in
    *:*) ;;
    *) echo "Unable to determine the runtime UID/GID for $THISM_IMAGE" >&2; exit 1 ;;
  esac
  container_uid="${identity%%:*}"
  container_gid="${identity##*:}"
  case "$container_uid" in
    ""|*[!0-9]*) echo "Invalid runtime UID reported by $THISM_IMAGE: $identity" >&2; exit 1 ;;
  esac
  case "$container_gid" in
    ""|*[!0-9]*) echo "Invalid runtime GID reported by $THISM_IMAGE: $identity" >&2; exit 1 ;;
  esac

  docker run --rm --user 0:0 --entrypoint sh \
    -v "$host_path:/geo" "$THISM_IMAGE" \
    -c "chown $container_uid:$container_gid /geo && chmod 0755 /geo"

  if ! docker run --rm --entrypoint sh -v "$host_path:/geo" "$THISM_IMAGE" -c 'test -w /geo'; then
    echo "GeoIP directory is not writable by the ThisM container: $host_path" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd docker
require_cmd mktemp

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose is required" >&2
  exit 1
fi

mkdir -p "$THISM_INSTALL_DIR"
chmod 700 "$THISM_INSTALL_DIR"
cd "$THISM_INSTALL_DIR"
THISM_INSTALL_DIR="$(pwd -P)"
umask 077

tmp_env="$(mktemp)"
trap 'rm -f "$tmp_env"' EXIT

curl -fsSL "$RAW_BASE/docker-compose.yml" -o compose.yaml
curl -fsSL "$RAW_BASE/.env.example" -o .env.example

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

THISM_IMAGE="${THISM_IMAGE:-ghcr.io/jsllxx77/thism:latest}"
THISM_PORT="${THISM_PORT:-8080}"
THISM_ADMIN_USER="${THISM_ADMIN_USER:-admin}"
THISM_TOKEN_SECRET_FILE="${THISM_TOKEN_SECRET_FILE:-./secrets/thism_token}"
THISM_ADMIN_USER_SECRET_FILE="${THISM_ADMIN_USER_SECRET_FILE:-./secrets/thism_admin_user}"
THISM_ADMIN_PASS_SECRET_FILE="${THISM_ADMIN_PASS_SECRET_FILE:-./secrets/thism_admin_pass}"
THISM_GEOIP_HOST_DIR="${THISM_GEOIP_HOST_DIR:-/var/lib/thism/geo}"

if [ -z "$THISM_TOKEN" ] && [ ! -s "$THISM_TOKEN_SECRET_FILE" ]; then
  THISM_TOKEN="$(random_secret)"
fi

if [ -z "$THISM_ADMIN_USER" ] && [ ! -s "$THISM_ADMIN_USER_SECRET_FILE" ]; then
  THISM_ADMIN_USER="admin"
fi

if [ -z "$THISM_ADMIN_PASS" ] && [ ! -s "$THISM_ADMIN_PASS_SECRET_FILE" ]; then
  THISM_ADMIN_PASS="$(random_secret)"
fi

write_secret_file "$THISM_TOKEN_SECRET_FILE" "$THISM_TOKEN" "THISM_TOKEN"
write_secret_file "$THISM_ADMIN_USER_SECRET_FILE" "$THISM_ADMIN_USER" "THISM_ADMIN_USER"
write_secret_file "$THISM_ADMIN_PASS_SECRET_FILE" "$THISM_ADMIN_PASS" "THISM_ADMIN_PASS"

cat >"$tmp_env" <<EOF
THISM_IMAGE=$THISM_IMAGE
THISM_PORT=$THISM_PORT
THISM_TOKEN_SECRET_FILE=$THISM_TOKEN_SECRET_FILE
THISM_ADMIN_USER_SECRET_FILE=$THISM_ADMIN_USER_SECRET_FILE
THISM_ADMIN_PASS_SECRET_FILE=$THISM_ADMIN_PASS_SECRET_FILE
THISM_GEOIP_HOST_DIR=$THISM_GEOIP_HOST_DIR
EOF
mv "$tmp_env" .env
trap - EXIT

docker compose pull
prepare_geoip_dir "$THISM_GEOIP_HOST_DIR"
docker compose up -d

echo
echo "ThisM has been deployed."
echo "Directory: $THISM_INSTALL_DIR"
echo "URL: http://localhost:${THISM_PORT}"
echo "Credentials are stored as Docker secret files:"
echo "  Admin user: $(display_secret_path "$THISM_ADMIN_USER_SECRET_FILE")"
echo "  Admin password: $(display_secret_path "$THISM_ADMIN_PASS_SECRET_FILE")"
echo "  API token: $(display_secret_path "$THISM_TOKEN_SECRET_FILE")"
echo
echo "GeoIP data directory: $(resolve_host_path "$THISM_GEOIP_HOST_DIR")"
echo "Runtime settings are stored in: $THISM_INSTALL_DIR/.env"
