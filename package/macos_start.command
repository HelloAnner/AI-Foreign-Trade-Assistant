#!/bin/bash
set -e

# Run the packaged macOS binary from the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

if [ ! -f "./AI_Trade_Assistant_macOS_arm64" ]; then
  echo "[ERROR] Missing AI_Trade_Assistant_macOS_arm64 in ${SCRIPT_DIR}"
  exit 1
fi

chmod +x ./AI_Trade_Assistant_macOS_arm64
exec "./AI_Trade_Assistant_macOS_arm64"
