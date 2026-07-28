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

A fresh host therefore needs a one-time (or periodic) fetch step.

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

`thism-server` accepts:

| Variable / flag | Purpose |
|---|---|
| `THISM_GEOIP_DB` / `--geoip-db` | Primary database path |
| `THISM_GEOIP_DB_FALLBACK` / `--geoip-db-fallback` | Optional second database |

Behavior:

1. If both paths are set and readable, both are loaded
2. Lookups use the primary first
3. If primary returns empty, fallback is tried
4. If flags/env are empty, common paths are auto-detected:
   - `/var/lib/thism/geo/...`
   - `/opt/1panel/geo/...` (legacy compatibility)
   - `./geo/...`

Missing databases only disable country enrichment; the server still starts.

## Docker Compose note

Mount a host geo directory into the container and point env vars at it, for example:

```yaml
volumes:
  - /var/lib/thism/geo:/geo:ro
environment:
  THISM_GEOIP_DB: /geo/IP2LOCATION-LITE-DB1.IPV6.BIN
  THISM_GEOIP_DB_FALLBACK: /geo/GeoIP.mmdb
```

Fetch the files on the host with `deploy/fetch-geoip-dbs.sh` before `docker compose up`.

## License reminders

- IP2Location LITE requires attribution:
  “uses the IP2Location LITE database for IP geolocation”
- MaxMind GeoLite2 requires a license key and compliance with MaxMind terms
- Never commit tokens/license keys or raw database files to git
