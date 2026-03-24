#!/usr/bin/env bash
# =========================================
#  ANTIGRAVITY — Sample Run Script
# =========================================
set -euo pipefail

echo "=== Building ANTIGRAVITY ==="
cd "$(dirname "$0")/.."
go build -o antigravity ./cmd/server

echo "=== Starting server on http://localhost:3600 ==="
./antigravity &
SERVER_PID=$!
sleep 2

echo ""
echo "=== Health Check ==="
curl -s http://localhost:3600/health | python3 -m json.tool

echo ""
echo "=== Starting Index (Go docs, depth 1) ==="
curl -s -X POST http://localhost:3600/index \
  -H "Content-Type: application/json" \
  -d '{"origin": "https://go.dev", "maxDepth": 1}' | python3 -m json.tool

echo ""
echo "=== Waiting 15 seconds for crawl ==="
sleep 15

echo ""
echo "=== System State ==="
curl -s http://localhost:3600/api/state | python3 -m json.tool

echo ""
echo "=== Search: 'go' ==="
curl -s "http://localhost:3600/search?query=go&sortBy=relevance" | python3 -m json.tool

echo ""
echo "=== First 10 lines of p.data ==="
head -n 10 data/storage/p.data

echo ""
echo "=== Stopping server ==="
kill $SERVER_PID 2>/dev/null || true
echo "Done."
