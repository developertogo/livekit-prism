#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=========================================="
echo " Running Prism Unit Test Suite"
echo "=========================================="

echo ""
echo "[1/2] Running Go Edge Gateway Tests..."
cd "${ROOT_DIR}"
go test -v ./cmd/prism-gateway/...

echo ""
echo "[2/2] Running Python Agent Plugin & Server Tests..."
cd "${ROOT_DIR}"
uv run pytest -v

echo ""
echo "=========================================="
echo " All Unit Tests Passed Successfully!"
echo "=========================================="
