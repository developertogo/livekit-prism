#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"
echo "[Prism] Starting FastAPI WebRTC Token Server and UI Harness on http://localhost:3000..."
exec uv run uvicorn server:app --reload --port 3000
