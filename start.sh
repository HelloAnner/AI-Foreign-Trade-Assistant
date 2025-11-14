#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

APP_NAME="ai-foreign-trade-assistant"
IMAGE_NAME="${APP_NAME}:latest"
CONTAINER_NAME="${APP_NAME}-runner"
HOST_PORT="${APP_PORT:-7860}"
DATA_DIR="$ROOT_DIR/data"
HEALTH_URL="http://127.0.0.1:${HOST_PORT}/api/health"
SEARCH_URL="http://127.0.0.1:${HOST_PORT}/api/settings/test-search"

log() {
  printf '[start] %s\n' "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "缺少依赖: $1"
    exit 1
  fi
}

require_cmd docker
require_cmd curl

DOCKER=(docker)
if [ -n "${DOCKER_CONTEXT:-}" ]; then
  log "使用 docker context: ${DOCKER_CONTEXT}"
  DOCKER=(docker --context "$DOCKER_CONTEXT")
fi

if ! "${DOCKER[@]}" info >/dev/null 2>&1; then
  log "Docker 守护进程未运行，请先启动 Docker"
  exit 1
fi

cleanup_stack() {
  if "${DOCKER[@]}" ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    log "停止已有容器 ${CONTAINER_NAME}"
    "${DOCKER[@]}" stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    "${DOCKER[@]}" rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi
}

build_image() {
  log "构建 ${IMAGE_NAME} 镜像..."
  "${DOCKER[@]}" build -t "$IMAGE_NAME" .
}

start_container() {
  log "启动容器 ${CONTAINER_NAME}..."
  mkdir -p "$DATA_DIR"
  "${DOCKER[@]}" run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "${HOST_PORT}:7860" \
    -v "$DATA_DIR:/data" \
    -e TZ="${TZ:-Asia/Shanghai}" \
    "$IMAGE_NAME"
}

wait_for_health() {
  log "等待服务健康就绪..."
  local retries=60
  local delay=2
  for ((i=1; i<=retries; i++)); do
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
      log "服务健康检查通过"
      return 0
    fi
    sleep "$delay"
  done
  log "健康检查超时，打印容器日志"
  "${DOCKER[@]}" logs "$CONTAINER_NAME"
  exit 1
}

run_search_smoke() {
  log "触发 Playwright 搜索自检..."
  local payload='{}'
  local tmpfile="$(mktemp)"
  local http_code
  if http_code=$(curl -sS -w '%{http_code}' -o "$tmpfile" -X POST "$SEARCH_URL" -H 'Content-Type: application/json' -d "$payload"); then
    if [ "$http_code" = "200" ]; then
      log "搜索 API 返回成功"
      cat "$tmpfile"
      rm -f "$tmpfile"
      return 0
    fi
    log "搜索 API 返回 HTTP $http_code"
  else
    log "搜索 API 调用异常"
  fi
  cat "$tmpfile" 2>/dev/null || true
  rm -f "$tmpfile"
  "${DOCKER[@]}" logs "$CONTAINER_NAME" | tail -n 200
  exit 1
}

main() {
  cleanup_stack
  build_image
  CONTAINER_ID=$(start_container)
  log "容器 ID: $CONTAINER_ID"
  wait_for_health
  run_search_smoke
  log "🎉 部署完成，可访问 http://127.0.0.1:${HOST_PORT}"
}

main "$@"
