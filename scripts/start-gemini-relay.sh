#!/usr/bin/env bash
# Keep Gemini reachable from the RU VPS by relaying through this machine's egress.
# Usage:
#   ./scripts/start-gemini-relay.sh root@5.63.152.227
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VPS_HOST="${1:-root@5.63.152.227}"
LOCAL_PORT="${LOCAL_PORT:-8787}"

if ! curl -fsS "http://127.0.0.1:${LOCAL_PORT}/" >/dev/null 2>&1; then
  :
fi

if ! pgrep -f "gemini_relay.py ${LOCAL_PORT}" >/dev/null 2>&1; then
  nohup python3 "$ROOT/scripts/gemini_relay.py" "$LOCAL_PORT" > /tmp/gemini-relay.log 2>&1 &
  sleep 1
fi

pkill -f "ssh.*-R.*${LOCAL_PORT}:127.0.0.1:${LOCAL_PORT}.*${VPS_HOST#*@}" 2>/dev/null || true
ssh -fN \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -i "${SSH_IDENTITY:-$HOME/.ssh/id_ed25519}" \
  -R "0.0.0.0:${LOCAL_PORT}:127.0.0.1:${LOCAL_PORT}" \
  "$VPS_HOST"

echo "Gemini relay up on localhost:${LOCAL_PORT}, forwarded to ${VPS_HOST}:${LOCAL_PORT}"
echo "VPS should use GEMINI_API_BASE=http://gemini-relay:8787 (compose service) or http://gemini-relay-bridge:8787"
