#!/usr/bin/env sh
set -e

log() {
  printf '[entrypoint] %s\n' "$*"
}

# Default envs mirroring find_jobs approach
: "${PLAYWRIGHT_DRIVER_PATH:=/opt/playwright/driver}"
: "${PLAYWRIGHT_BROWSERS_PATH:=/opt/playwright/browsers}"
: "${HOME:=/data}"
: "${FOREIGN_TRADE_NO_BROWSER:=1}"
export PLAYWRIGHT_DRIVER_PATH PLAYWRIGHT_BROWSERS_PATH HOME FOREIGN_TRADE_NO_BROWSER
export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS="${PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS:-true}"
export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD="${PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD:-1}"

mkdir -p "$HOME"

check_driver_file() {
  local rel="$1"
  local abs="$PLAYWRIGHT_DRIVER_PATH/$rel"
  if [ ! -e "$abs" ]; then
    log "缺少 Playwright driver 文件: $abs"
    exit 1
  fi
}

if [ ! -d "$PLAYWRIGHT_DRIVER_PATH" ]; then
  log "未找到 Playwright driver 目录: $PLAYWRIGHT_DRIVER_PATH"
  exit 1
fi

check_driver_file node
check_driver_file package/package.json
check_driver_file package/index.js
check_driver_file package/lib/server/index.js

if [ ! -d "$PLAYWRIGHT_BROWSERS_PATH" ]; then
  log "未找到 Playwright 浏览器缓存: $PLAYWRIGHT_BROWSERS_PATH"
  exit 1
fi

if ! find "$PLAYWRIGHT_BROWSERS_PATH" -maxdepth 1 -type d -name 'chromium-*' | grep -q .; then
  log "浏览器缓存中缺少 chromium-* 目录"
  exit 1
fi

mkdir -p "$HOME/.foreign_trade/logs"
mkdir -p "$HOME/.foreign_trade/cache"

log "环境检查通过，启动主进程"
exec "$@"
