# Offline GeoIP databases for thism

thism resolves node country codes from **local offline files**.
Nothing is queried online at request time.

Supported formats:

- IP2Location `.BIN` (recommended default)
- MaxMind GeoLite2 `.mmdb`

## Why GitHub clone is not enough

The git repo ships **code**, not the vendor database files.

- Database binaries are large
- Redistribution is restricted by vendor licenses
- Download credentials must stay private

A fresh host therefore needs a one-time database acquisition step, either from the Settings UI or with the fetch script below. The server still starts when no database is installed.

## Settings UI

Admin settings → **Monitoring** includes an **IP Geolocation Source** card:

1. Default provider is **MaxMind**
2. Switch to **IP2Location**
3. Paste your own download token
4. Click **Update database** to refresh the offline file in place

Secrets are stored in SQLite `app_settings` and never returned in API responses
(only `*_set` booleans). Runtime lookups remain fully offline after download.

## Recommended bootstrap on a new server

```bash
git clone https://github.com/<you>/thism.git
cd thism

# secrets via env (do not commit)
export IP2LOCATION_TOKEN='...'
export MAXMIND_LICENSE_KEY='...'   # from https://www.maxmind.com/en/geolite2/signup

# optional:
# export GEOIP_DIR=/var/lib/thism/geo
# export FETCH_IP2LOCATION=1
# export FETCH_MAXMIND=1

sudo -E ./deploy/fetch-geoip-dbs.sh
```

This installs:

- `$GEOIP_DIR/IP2LOCATION-LITE-DB1.IPV6.BIN`
- `$GEOIP_DIR/GeoIP.mmdb`

Default `GEOIP_DIR` is `/var/lib/thism/geo`.

## Runtime wiring

For Settings-managed provider selection and database updates, set the writable directory instead of individual database paths:

| Variable / flag | Purpose |
|---|---|
| `THISM_GEOIP_DIR` / `--geoip-dir` | Writable directory for provider-managed databases |

The legacy fixed-path mode remains available for existing deployments:

| Variable / flag | Purpose |
|---|---|
| `THISM_GEOIP_DB` / `--geoip-db` | Primary database path |
| `THISM_GEOIP_DB_FALLBACK` / `--geoip-db-fallback` | Optional second database |

Setting either legacy path override bypasses the Settings-managed provider and updater.

Behavior:

- Managed mode loads the selected provider from `THISM_GEOIP_DIR`. With the default `/var/lib/thism/geo` directory, a missing selected database falls back to the matching file under `/opt/1panel/geo` for compatibility. Database updates still write to the new managed directory.
- Legacy fixed-path mode loads both readable paths, checks the primary first, and tries the fallback when the primary returns no country code.

Missing databases only disable country enrichment; the server still starts.

## Docker Compose note

Mount a host geo directory into the container and point env vars at it, for example:

```yaml
volumes:
  - /var/lib/thism/geo:/geo
environment:
  THISM_GEOIP_DIR: /geo
```

The mount must be writable if database updates will be triggered from the settings page. The one-command `install-compose.sh` installer detects the image UID/GID and prepares `THISM_GEOIP_HOST_DIR` automatically. For a manual Compose deployment, prepare the directory for the official image user before startup:

```bash
IMAGE=ghcr.io/jsllxx77/thism:latest
GEO_UID="$(docker run --rm --entrypoint id "$IMAGE" -u)"
GEO_GID="$(docker run --rm --entrypoint id "$IMAGE" -g)"
sudo install -d -m 0755 -o "$GEO_UID" -g "$GEO_GID" /var/lib/thism/geo
```

You can prefetch the files with `deploy/fetch-geoip-dbs.sh`, or enter provider credentials in Settings → Monitoring and update them from the UI after startup.

## License reminders

- IP2Location LITE requires attribution:
  “uses the IP2Location LITE database for IP geolocation”
- MaxMind GeoLite2 requires a license key and compliance with MaxMind terms
- Never commit tokens/license keys or raw database files to git
