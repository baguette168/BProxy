#!/bin/bash

set -e

echo "=========================================="
echo "BProxy Demo Test Script"
echo "=========================================="
echo ""

cd /workspace/bproxy

echo "[1/5] Building BProxy..."
make clean > /dev/null 2>&1
make build

echo "[2/5] Starting Admin Server in background..."
./bin/admin -addr 0.0.0.0:8443 > admin.log 2>&1 &
ADMIN_PID=$!
echo "Admin PID: $ADMIN_PID"
sleep 2

echo "[3/5] Starting Agent 1..."
./bin/agent -admin 127.0.0.1:8443 > agent1.log 2>&1 &
AGENT1_PID=$!
echo "Agent 1 PID: $AGENT1_PID"
sleep 2

echo "[4/5] Starting Agent 2..."
./bin/agent -admin 127.0.0.1:8443 > agent2.log 2>&1 &
AGENT2_PID=$!
echo "Agent 2 PID: $AGENT2_PID"
sleep 2

echo "[5/5] Checking logs..."
echo ""
echo "=== Admin Log (last 10 lines) ==="
tail -10 admin.log
echo ""
echo "=== Agent 1 Log (last 5 lines) ==="
tail -5 agent1.log
echo ""
echo "=== Agent 2 Log (last 5 lines) ==="
tail -5 agent2.log
echo ""

echo "=========================================="
echo "Demo running! Processes:"
echo "  Admin:   PID $ADMIN_PID"
echo "  Agent 1: PID $AGENT1_PID"
echo "  Agent 2: PID $AGENT2_PID"
echo ""
echo "Logs available in:"
echo "  - admin.log"
echo "  - agent1.log"
echo "  - agent2.log"
echo ""
echo "Press Ctrl+C to stop all processes..."
echo "=========================================="

cleanup() {
    echo ""
    echo "Stopping all processes..."
    kill $ADMIN_PID $AGENT1_PID $AGENT2_PID 2>/dev/null || true
    echo "Cleanup complete!"
    exit 0
}

trap cleanup INT TERM

wait