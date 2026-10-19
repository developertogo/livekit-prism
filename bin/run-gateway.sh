#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BIN_PATH="${SCRIPT_DIR}/prism-gateway"

if [ ! -f "${BIN_PATH}" ]; then
    echo "[Prism] Building prism-gateway binary..."
    cd "${ROOT_DIR}"
    go build -o "${BIN_PATH}" ./cmd/prism-gateway
fi

echo "[Prism] Starting Prism Go Edge Gateway..."
cd "${ROOT_DIR}"
exec "${BIN_PATH}"
