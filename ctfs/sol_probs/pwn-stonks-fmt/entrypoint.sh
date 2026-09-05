#!/bin/sh
set -e

HTTP_PORT="${PORT:-8080}"
export HTTP_PORT

echo "[*] Starting Socat TCP Challenge listener on port 5000..."
socat TCP-LISTEN:5000,reuseaddr,fork,nodelay EXEC:/home/ctf/challenge,pty,stderr,echo=0 &

echo "[*] Starting Caddy File Server on port ${HTTP_PORT}..."
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
