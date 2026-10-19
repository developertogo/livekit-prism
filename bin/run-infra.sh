#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"
echo "[Prism] Starting ClickHouse and NATS via docker compose..."
docker compose up -d
echo "[Prism] Infrastructure containers started."
