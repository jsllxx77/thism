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
EOF
mv "$tmp_env" .env
trap - EXIT

docker compose pull
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
echo "Runtime settings are stored in: $THISM_INSTALL_DIR/.env"
