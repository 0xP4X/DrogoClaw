#!/usr/bin/env bash
# start.sh — Start OWASP Juice Shop for benchmarking
# Usage: ./benchmarks/juice_shop/start.sh
# Stops and removes any existing container, then starts a fresh one.

set -euo pipefail

NAME="juice-shop-bench"
PORT="${JUICE_SHOP_PORT:-3000}"

echo "[*] Stopping existing Juice Shop container (if any)..."
docker rm -f "$NAME" 2>/dev/null || true

echo "[*] Starting Juice Shop on port $PORT..."
docker run -d \
  --name "$NAME" \
  -p "127.0.0.1:$PORT:3000" \
  --restart unless-stopped \
  bkimminich/juice-shop

echo "[*] Waiting for Juice Shop to be ready..."
for i in $(seq 1 30); do
  if curl -sf "http://localhost:$PORT" >/dev/null 2>&1; then
    echo "[+] Juice Shop is ready at http://localhost:$PORT"
    echo "[+] Score Board: http://localhost:$PORT/#/score-board"
    echo "[+] API Docs:   http://localhost:$PORT/api-docs"
    exit 0
  fi
  sleep 1
done

echo "[-] Juice Shop did not start within 30 seconds."
echo "    Check: docker logs $NAME"
exit 1
