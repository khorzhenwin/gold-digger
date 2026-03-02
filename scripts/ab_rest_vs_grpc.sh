#!/usr/bin/env bash

set -euo pipefail

REST_BASE_URL="${REST_BASE_URL:-http://localhost:8080}"
GRPC_ADDR="${GRPC_ADDR:-localhost:9090}"
SYMBOL="${SYMBOL:-AAPL}"
WATCHLIST_ID="${WATCHLIST_ID:-1}"

require_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing dependency: $1"
    exit 1
  fi
}

require_bin curl
require_bin grpcurl

echo "== Read path: ticker price =="
echo "-- REST"
time curl -sS "${REST_BASE_URL}/api/v1/ticker-price/${SYMBOL}" >/tmp/rest_ticker.json
echo "-- gRPC"
time grpcurl -plaintext -d "{\"ticker\":\"${SYMBOL}\"}" \
  "${GRPC_ADDR}" golddigger.v1.TickerPriceService/GetTickerPrice >/tmp/grpc_ticker.json
echo "REST response:"
cat /tmp/rest_ticker.json
echo
echo "gRPC response:"
cat /tmp/grpc_ticker.json
echo

echo "== Write path: delete watchlist item =="
echo "-- REST"
time curl -sS -o /dev/null -w "HTTP %{http_code}\n" -X DELETE \
  "${REST_BASE_URL}/api/v1/watchlist/${WATCHLIST_ID}"
echo "-- gRPC"
time grpcurl -plaintext -d "{\"id\":${WATCHLIST_ID}}" \
  "${GRPC_ADDR}" golddigger.v1.WatchlistService/DeleteWatchlistItem

echo
echo "A/B run complete."
