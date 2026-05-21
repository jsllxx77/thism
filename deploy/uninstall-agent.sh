#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="${THISM_AGENT_SERVICE_NAME:-thism-agent}"
SERVICE_FILE="${THISM_AGENT_SERVICE_FILE:-/etc/systemd/system/${SERVICE_NAME}.service}"
ENV_FILE="${THISM_AGENT_ENV_FILE:-/etc/default/thism-agent}"
TARGET_BIN="${THISM_AGENT_BIN:-/usr/local/bin/thism-agent}"
VERSION_FILE="${THISM_AGENT_VERSION_FILE:-/usr/local/bin/.thism-agent.version}"

if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  echo "This script must be run as root." >&2
  exit 1
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now "$SERVICE_NAME" >/dev/null 2>&1 || true
fi

rm -f "$SERVICE_FILE"
rm -f "$ENV_FILE"
rm -f "$TARGET_BIN"
rm -f "$VERSION_FILE"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || true
  systemctl reset-failed "$SERVICE_NAME" >/dev/null 2>&1 || true
fi

echo "ThisM agent has been removed from this host."
echo
echo "Important: this only removes the local agent files and systemd service."
echo "Open the ThisM web panel and delete this node from Settings -> Node Management if you no longer want it listed there."
