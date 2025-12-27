#!/usr/bin/env sh
set -e

log() { printf '[boss-entrypoint] %s\n' "$*"; }

: "${PLAYWRIGHT_DRIVER_PATH:=/opt/playwright/driver}"
: "${PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS:=true}"
: "${PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD:=1}"
: "${FOREIGN_TRADE_NO_BROWSER:=1}"
export PLAYWRIGHT_DRIVER_PATH PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD FOREIGN_TRADE_NO_BROWSER

if [ -n "${BROWSERLESS_URL:-}" ]; then
  log "Using remote Browserless: $BROWSERLESS_URL (mode=${BROWSERLESS_MODE:-playwright})"
fi

exec "$@"

