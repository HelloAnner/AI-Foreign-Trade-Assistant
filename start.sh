#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

APP_NAME="ai-foreign-trade-assistant"
IMAGE_NAME="${APP_NAME}:latest"
CONTAINER_NAME="${APP_NAME}-runner"
HOST_PORT="${APP_PORT:-25000}"
HOST_HOME="${HOST_HOME_OVERRIDE:-$HOME}"
DATA_DIR="${FOREIGN_TRADE_DATA_DIR:-$HOST_HOME/.foreign_trade}"
CONTAINER_DATA_ROOT="/data/.foreign_trade"
HEALTH_URL="http://127.0.0.1:${HOST_PORT}/api/health"

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
    -v "$DATA_DIR:${CONTAINER_DATA_ROOT}" \
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

main() {
  cleanup_stack
  build_image
  CONTAINER_ID=$(start_container)
  log "容器 ID: $CONTAINER_ID"
  wait_for_health
  log "🎉 部署完成，可访问 http://127.0.0.1:${HOST_PORT}"
}

main "$@"
