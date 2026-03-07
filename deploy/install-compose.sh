#!/usr/bin/env bash
set -euo pipefail

THISM_GITHUB_REPO="${THISM_GITHUB_REPO:-thism-dev/thism}"
THISM_REF="${THISM_REF:-main}"
THISM_IMAGE="${THISM_IMAGE:-ghcr.io/thism-dev/thism:latest}"
THISM_INSTALL_DIR="${THISM_INSTALL_DIR:-$HOME/thism-deploy}"
THISM_PORT="${THISM_PORT:-8080}"
THISM_ADMIN_USER="${THISM_ADMIN_USER:-admin}"
THISM_TOKEN="${THISM_TOKEN:-}"
THISM_ADMIN_PASS="${THISM_ADMIN_PASS:-}"

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

require_cmd curl
require_cmd docker
require_cmd mktemp

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose is required" >&2
  exit 1
fi

mkdir -p "$THISM_INSTALL_DIR"
cd "$THISM_INSTALL_DIR"
umask 077

tmp_env="$(mktemp)"
trap 'rm -f "$tmp_env"' EXIT

curl -fsSL "$RAW_BASE/docker-compose.yml" -o compose.yaml
curl -fsSL "$RAW_BASE/.env.example" -o .env.example

if [ ! -f .env ]; then
  if [ -z "$THISM_TOKEN" ]; then
    THISM_TOKEN="$(random_secret)"
  fi

  if [ -z "$THISM_ADMIN_PASS" ]; then
    THISM_ADMIN_PASS="$(random_secret)"
  fi

  cat >"$tmp_env" <<EOF
THISM_IMAGE=$THISM_IMAGE
THISM_PORT=$THISM_PORT
THISM_TOKEN=$THISM_TOKEN
THISM_ADMIN_USER=$THISM_ADMIN_USER
THISM_ADMIN_PASS=$THISM_ADMIN_PASS
EOF
  mv "$tmp_env" .env
  trap - EXIT
else
  rm -f "$tmp_env"
  trap - EXIT
fi

set -a
. ./.env
set +a

docker compose pull
docker compose up -d

echo
echo "ThisM has been deployed."
echo "Directory: $THISM_INSTALL_DIR"
echo "URL: http://localhost:${THISM_PORT}"
echo "Admin user: $THISM_ADMIN_USER"
echo "Admin password: $THISM_ADMIN_PASS"
echo "API token: $THISM_TOKEN"
echo
echo "Credentials are stored in: $THISM_INSTALL_DIR/.env"
