#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${THISM_INSTALL_DIR:-$HOME/thism-deploy}"
SERVICE_NAME="${THISM_SERVER_SERVICE_NAME:-thism-server}"
SERVICE_FILE="${THISM_SERVER_SERVICE_FILE:-/etc/systemd/system/${SERVICE_NAME}.service}"
ENV_FILE="${THISM_SERVER_ENV_FILE:-/etc/default/thism-server}"
TARGET_BIN="${THISM_SERVER_BIN:-/usr/local/bin/thism-server}"
DATA_DIR="${THISM_SERVER_DATA_DIR:-/var/lib/thism}"
REMOVE_DATA="${THISM_REMOVE_DATA:-0}"

is_truthy() {
  case "${1:-}" in
    1|true|TRUE|yes|YES|y|Y) return 0 ;;
    *) return 1 ;;
  esac
}

if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  echo "This script must be run as root." >&2
  exit 1
fi

compose_ran=0
if command -v docker >/dev/null 2>&1 && [ -f "${INSTALL_DIR}/compose.yaml" ]; then
  (
    cd "$INSTALL_DIR"
    if is_truthy "$REMOVE_DATA"; then
      docker compose down --volumes --remove-orphans
    else
      docker compose down --remove-orphans
    fi
  )
  compose_ran=1
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now "$SERVICE_NAME" >/dev/null 2>&1 || true
fi

rm -f "$SERVICE_FILE"
rm -f "$ENV_FILE"
rm -f "$TARGET_BIN"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || true
  systemctl reset-failed "$SERVICE_NAME" >/dev/null 2>&1 || true
fi

if is_truthy "$REMOVE_DATA"; then
  if [ -d "$INSTALL_DIR" ]; then
    rm -rf "$INSTALL_DIR"
  fi
  if [ -d "$DATA_DIR" ]; then
    rm -rf "$DATA_DIR"
  fi
else
  if [ "$compose_ran" -eq 1 ]; then
    echo "Preserved Docker volume data. Re-run with THISM_REMOVE_DATA=1 to remove compose volumes and ${INSTALL_DIR}."
  fi
  if [ -d "$DATA_DIR" ]; then
    echo "Preserved ${DATA_DIR}. Re-run with THISM_REMOVE_DATA=1 to remove it."
  fi
fi

echo "ThisM server has been removed from this host."
echo
echo "Important: agents installed on other hosts are not removed by this script."
echo "Run the agent uninstall script on each monitored host if you want to remove those agents too."
