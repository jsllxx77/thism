#!/usr/bin/env bash
# Download offline GeoIP databases for thism.
#
# Supports:
#   - IP2Location LITE DB1 (IPv4+IPv6 BIN)
#   - MaxMind GeoLite2-City (MMDB)
#
# Credentials stay out of git:
#   IP2LOCATION_TOKEN or IP2LOCATION_TOKEN_FILE
#   MAXMIND_LICENSE_KEY or MAXMIND_LICENSE_KEY_FILE
#
# Example (fresh host after cloning thism):
#   export IP2LOCATION_TOKEN=...
#   export MAXMIND_LICENSE_KEY=...
#   sudo ./deploy/fetch-geoip-dbs.sh
#
# Then start thism-server. It will auto-detect both files under GEOIP_DIR.

set -euo pipefail

GEOIP_DIR="${GEOIP_DIR:-/var/lib/thism/geo}"
IP2LOCATION_CODE="${IP2LOCATION_CODE:-DB1LITEBINIPV6}"
MAXMIND_EDITION="${MAXMIND_EDITION:-GeoLite2-City}"
FETCH_IP2LOCATION="${FETCH_IP2LOCATION:-1}"
FETCH_MAXMIND="${FETCH_MAXMIND:-1}"
KEEP_DOWNLOADS="${KEEP_DOWNLOADS:-0}"

log() { printf '[fetch-geoip] %s\n' "$*"; }
die() { printf '[fetch-geoip] ERROR: %s\n' "$*" >&2; exit 1; }

read_secret() {
  local env_name="$1"
  local file_env_name="$2"
  local value="${!env_name:-}"
  if [[ -n "$value" ]]; then
    printf '%s' "$value"
    return 0
  fi
  local file_path="${!file_env_name:-}"
  if [[ -n "$file_path" ]]; then
    [[ -f "$file_path" ]] || die "$file_env_name points to missing file: $file_path"
    tr -d '\r\n' <"$file_path"
    return 0
  fi
  printf ''
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

download() {
  local url="$1"
  local out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 15 --max-time 300 -o "$out" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$out" "$url"
  else
    die "need curl or wget"
  fi
}

install_file() {
  local src="$1"
  local dest="$2"
  mkdir -p "$(dirname "$dest")"
  local tmp="${dest}.tmp.$$"
  cp -f "$src" "$tmp"
  chmod 644 "$tmp"
  mv -f "$tmp" "$dest"
  log "installed $dest ($(wc -c <"$dest" | tr -d ' ') bytes)"
}

fetch_ip2location() {
  local token
  token="$(read_secret IP2LOCATION_TOKEN IP2LOCATION_TOKEN_FILE)"
  [[ -n "$token" ]] || die "IP2LOCATION_TOKEN or IP2LOCATION_TOKEN_FILE is required to fetch IP2Location"

  local url="https://www.ip2location.com/download?token=${token}&file=${IP2LOCATION_CODE}"
  local zip_path="${WORKDIR}/ip2location.zip"
  log "downloading IP2Location ${IP2LOCATION_CODE}"
  download "$url" "$zip_path"

  # Reject HTML error pages pretending to be zips.
  if file "$zip_path" | grep -qi 'html\|text'; then
    die "IP2Location download did not return a zip (check token/code). First bytes: $(head -c 120 "$zip_path" | tr '\n' ' ')"
  fi

  need_cmd unzip
  local extract_dir="${WORKDIR}/ip2location"
  mkdir -p "$extract_dir"
  unzip -qo "$zip_path" -d "$extract_dir"

  local bin
  bin="$(find "$extract_dir" -type f \( -iname '*.IPV6.BIN' -o -iname '*DB1*.BIN' -o -iname '*.BIN' \) | head -n 1 || true)"
  [[ -n "$bin" ]] || die "no .BIN found inside IP2Location archive"
  install_file "$bin" "${GEOIP_DIR}/IP2LOCATION-LITE-DB1.IPV6.BIN"

  # Keep license/readme next to the DB for attribution.
  local readme license
  readme="$(find "$extract_dir" -type f -iname 'README*.TXT' | head -n 1 || true)"
  license="$(find "$extract_dir" -type f -iname 'LICENSE*.TXT' | head -n 1 || true)"
  [[ -n "$readme" ]] && install_file "$readme" "${GEOIP_DIR}/IP2LOCATION-README_LITE.TXT"
  [[ -n "$license" ]] && install_file "$license" "${GEOIP_DIR}/IP2LOCATION-LICENSE-CC-BY-SA-4.0.TXT"
}

fetch_maxmind() {
  local key
  key="$(read_secret MAXMIND_LICENSE_KEY MAXMIND_LICENSE_KEY_FILE)"
  [[ -n "$key" ]] || die "MAXMIND_LICENSE_KEY or MAXMIND_LICENSE_KEY_FILE is required to fetch MaxMind GeoLite2"

  local url="https://download.maxmind.com/app/geoip_download?edition_id=${MAXMIND_EDITION}&license_key=${key}&suffix=tar.gz"
  local tarball="${WORKDIR}/maxmind.tar.gz"
  log "downloading MaxMind ${MAXMIND_EDITION}"
  download "$url" "$tarball"

  if file "$tarball" | grep -qi 'html\|text\|zip'; then
    # MaxMind returns plain text errors sometimes.
    if ! file "$tarball" | grep -qi 'gzip\|tar'; then
      die "MaxMind download failed (check license key/edition). Body: $(head -c 200 "$tarball" | tr '\n' ' ')"
    fi
  fi

  need_cmd tar
  local extract_dir="${WORKDIR}/maxmind"
  mkdir -p "$extract_dir"
  tar -xzf "$tarball" -C "$extract_dir"

  local mmdb
  mmdb="$(find "$extract_dir" -type f -name "*.mmdb" | head -n 1 || true)"
  [[ -n "$mmdb" ]] || die "no .mmdb found inside MaxMind archive"
  install_file "$mmdb" "${GEOIP_DIR}/GeoIP.mmdb"
}

main() {
  need_cmd mkdir
  need_cmd cp
  need_cmd mv
  need_cmd chmod
  command -v file >/dev/null 2>&1 || log "warning: 'file' command missing; content-type checks will be weaker"

  mkdir -p "$GEOIP_DIR"
  WORKDIR="$(mktemp -d /tmp/thism-geoip.XXXXXX)"
  cleanup() {
    if [[ "$KEEP_DOWNLOADS" == "1" ]]; then
      log "kept downloads in $WORKDIR"
    else
      rm -rf "$WORKDIR"
    fi
  }
  trap cleanup EXIT

  local fetched=0
  if [[ "$FETCH_IP2LOCATION" == "1" ]]; then
    fetch_ip2location
    fetched=$((fetched + 1))
  else
    log "skip IP2Location (FETCH_IP2LOCATION=0)"
  fi
  if [[ "$FETCH_MAXMIND" == "1" ]]; then
    fetch_maxmind
    fetched=$((fetched + 1))
  else
    log "skip MaxMind (FETCH_MAXMIND=0)"
  fi

  [[ "$fetched" -gt 0 ]] || die "nothing to fetch; enable FETCH_IP2LOCATION and/or FETCH_MAXMIND"

  log "done. databases in $GEOIP_DIR:"
  ls -lh "$GEOIP_DIR" | sed 's/^/[fetch-geoip]   /'
  cat <<EOF
[fetch-geoip] thism will auto-detect:
[fetch-geoip]   primary default: ${GEOIP_DIR}/IP2LOCATION-LITE-DB1.IPV6.BIN
[fetch-geoip]   fallback default: ${GEOIP_DIR}/GeoIP.mmdb
[fetch-geoip] override with:
[fetch-geoip]   THISM_GEOIP_DB=...
[fetch-geoip]   THISM_GEOIP_DB_FALLBACK=...
EOF
}

main "$@"
