#!/bin/bash
# Rebuild and restart fireup

set -e

cd "$(dirname "$0")/.."

echo "Building fireup..."
unset GOPATH
go install ./cmd/fireup/

echo "Stopping any running fireup..."
# Use SIGTERM first to allow graceful shutdown (kills child processes)
pkill -TERM fireup 2>/dev/null || true
sleep 2
# Then SIGKILL any stragglers
pkill -9 fireup 2>/dev/null || true
launchctl bootout gui/$(id -u)/com.fireup 2>/dev/null || true
sleep 1

# Reinstall service (regenerates plist with current PATH, HOME, etc.)
echo "Installing service..."
FIREUP_YES=1 fireup service install

echo "Waiting for server to start..."
sleep 1

if pgrep -q fireup; then
    echo "fireup is running (PID $(pgrep fireup))"
    echo "Dashboard: http://fireup.test"
else
    echo "Error: fireup failed to start"
    echo "Check logs: tail ~/Library/Logs/fireup/stderr.log"
    exit 1
fi
