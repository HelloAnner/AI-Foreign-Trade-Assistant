#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "$PROJECT_ROOT"

APP_NAME="ai-foreign-trade-assistant"
IMAGE_NAME="${APP_NAME}:latest"
CONTAINER_NAME="${APP_NAME}-runner"
HOST_HOME="${HOST_HOME_OVERRIDE:-$HOME}"
DATA_DIR="${FOREIGN_TRADE_DATA_DIR:-$HOST_HOME/.foreign_trade}"
CONTAINER_DATA_ROOT="/data/.foreign_trade"
DEFAULT_HTTP_PORT=25000

if [ -n "${APP_HTTP_ADDR:-}" ]; then
  case "$APP_HTTP_ADDR" in
    *:*) HOST_PORT="${APP_HTTP_ADDR##*:}" ;;
    *) HOST_PORT="$DEFAULT_HTTP_PORT" ;;
  esac
elif [ -n "${APP_PORT:-}" ]; then
  HOST_PORT="$APP_PORT"
else
  HOST_PORT="$DEFAULT_HTTP_PORT"
fi

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

  if "${DOCKER[@]}" image inspect "${IMAGE_NAME}" >/dev/null 2>&1; then
    log "删除旧镜像 ${IMAGE_NAME}"
    "${DOCKER[@]}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi
}

build_image() {
  log "构建 ${IMAGE_NAME} 镜像..."
  "${DOCKER[@]}" build -t "$IMAGE_NAME" .
}

start_container() {
  log "启动容器 ${CONTAINER_NAME}..."
  mkdir -p "$DATA_DIR"
  local docker_env=(-e "TZ=${TZ:-Asia/Shanghai}")
  if [ -n "${APP_PORT:-}" ]; then
    docker_env+=(-e "APP_PORT=${APP_PORT}")
  fi
  if [ -n "${APP_HTTP_ADDR:-}" ]; then
    docker_env+=(-e "APP_HTTP_ADDR=${APP_HTTP_ADDR}")
  fi
  docker_env+=(-e "FTA_DATA_DIR=${CONTAINER_DATA_ROOT}")
  "${DOCKER[@]}" run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    --network host \
    -v "$DATA_DIR:${CONTAINER_DATA_ROOT}" \
    "${docker_env[@]}" \
    "$IMAGE_NAME"
}

main() {
  cleanup_stack
  build_image
  CONTAINER_ID=$(start_container)
  log "容器 ID: $CONTAINER_ID"
  log "🎉 部署完成，可访问 http://127.0.0.1:${HOST_PORT}"
}

main "$@"
