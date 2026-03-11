#!/usr/bin/env bash
set -euo pipefail

# GateForge Load Test Script
# Requires: hey (go install github.com/rakyll/hey@latest) or wrk

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
DURATION="${DURATION:-60}"
CONCURRENCY="${CONCURRENCY:-50}"
RESULTS_FILE="scripts/loadtest_results.txt"

echo "========================================="
echo " GateForge Load Test"
echo "========================================="
echo "Gateway:     $GATEWAY_URL"
echo "Duration:    ${DURATION}s"
echo "Concurrency: $CONCURRENCY"
echo "========================================="
echo ""

# Detect available tool
if command -v hey &>/dev/null; then
    TOOL="hey"
elif command -v wrk &>/dev/null; then
    TOOL="wrk"
else
    echo "ERROR: Neither 'hey' nor 'wrk' found."
    echo "Install hey: go install github.com/rakyll/hey@latest"
    echo "Install wrk: https://github.com/wg/wrk"
    exit 1
fi

echo "Using: $TOOL"
echo ""

run_hey() {
    local url="$1"
    local label="$2"
    echo "--- $label ---"
    echo "URL: $url"
    hey -z "${DURATION}s" -c "$CONCURRENCY" -t 30 "$url" 2>&1
    echo ""
}

run_wrk() {
    local url="$1"
    local label="$2"
    echo "--- $label ---"
    echo "URL: $url"
    wrk -t4 -c"$CONCURRENCY" -d"${DURATION}s" --latency "$url" 2>&1
    echo ""
}

{
    echo "GateForge Load Test Results"
    echo "Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "Tool: $TOOL"
    echo "Duration: ${DURATION}s | Concurrency: $CONCURRENCY"
    echo ""

    ROUTES=(
        "$GATEWAY_URL/api/users|Users API"
        "$GATEWAY_URL/api/orders|Orders API"
        "$GATEWAY_URL/api/products|Products API"
    )

    for route_info in "${ROUTES[@]}"; do
        IFS='|' read -r url label <<< "$route_info"
        if [ "$TOOL" = "hey" ]; then
            run_hey "$url" "$label"
        else
            run_wrk "$url" "$label"
        fi
    done

    echo "========================================="
    echo " Load test completed"
    echo "========================================="
} 2>&1 | tee "$RESULTS_FILE"

echo ""
echo "Results saved to: $RESULTS_FILE"
