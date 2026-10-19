#!/usr/bin/env bash
set -euo pipefail

if ! command -v livekit-server &> /dev/null; then
    echo "[Prism] livekit-server not found in PATH."
    echo "[Prism] Install natively via Homebrew: brew install livekit-cli livekit"
    exit 1
fi

echo "[Prism] Starting LiveKit SFU in development mode..."
exec livekit-server --dev
