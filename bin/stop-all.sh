#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "[Prism] Stopping running processes..."
pkill -f "prism-gateway" || true
pkill -f "uvicorn server:app" || true
pkill -f "python agent.py" || true

cd "${ROOT_DIR}"
if command -v docker &> /dev/null; then
    echo "[Prism] Stopping docker compose containers..."
    docker compose down || true
fi

echo "[Prism] All services stopped."
