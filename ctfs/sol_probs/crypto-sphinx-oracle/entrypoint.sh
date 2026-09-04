#!/bin/sh
set -e

HTTP_PORT="${PORT:-8080}"
export HTTP_PORT

echo "[*] Starting Caddy File Server on port ${HTTP_PORT}..."
caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &

echo "[*] Starting Socat TCP Sphinx Oracle listener on port 5000..."
exec socat TCP-LISTEN:5000,reuseaddr,fork,nodelay EXEC:"/home/ctf/challenge",pty,stderr,echo=0
