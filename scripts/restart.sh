#!/bin/bash
# Rebuild and restart the launchd-managed roost-dev server

set -e

cd "$(dirname "$0")/.."

echo "Building roost-dev..."
unset GOPATH
go install ./cmd/roost-dev/

echo "Stopping any running roost-dev..."
# Use SIGTERM first to allow graceful shutdown (kills child processes)
pkill -TERM roost-dev 2>/dev/null || true
sleep 2
# Then SIGKILL any stragglers
pkill -9 roost-dev 2>/dev/null || true
launchctl bootout gui/$(id -u)/com.roost-dev 2>/dev/null || true
sleep 1
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.roost-dev.plist

echo "Waiting for server to start..."
sleep 1

if pgrep -q roost-dev; then
    echo "roost-dev is running (PID $(pgrep roost-dev))"
    echo "Dashboard: http://roost-dev.test"
else
    echo "Error: roost-dev failed to start"
    echo "Check logs: tail ~/Library/Logs/roost-dev/stderr.log"
    exit 1
fi
