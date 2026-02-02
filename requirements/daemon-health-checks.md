# Daemon Health Checks

## Problem

Services without HTTP endpoints (daemons, background workers, collectors) show as yellow in the dashboard because fireup can't confirm they're "ready" - it only knows the process is alive.

## Current Behavior

- Web services: Green when HTTP port responds
- Daemons: Yellow (running but unconfirmed ready)

## Proposed Solution

Allow services to specify a custom health check mechanism:

```yaml
services:
  collector:
    cmd: python collector.py
    health:
      type: file  # or 'tcp', 'exec', 'none'
      path: /tmp/collector-ready  # touch this file when ready
      # or: port: 9999  # for tcp
      # or: cmd: "pgrep -f collector"  # for exec
```

### Health Check Types

1. **file** - Service touches a file when ready, fireup watches for it
2. **tcp** - Service listens on a port (doesn't need HTTP, just accepts connection)
3. **exec** - Run a command, success = healthy
4. **none** - Explicitly opt out, stay yellow (this is fine)

## Use Cases

- Python firehose collectors
- Background job workers
- Message queue consumers
- Any long-running daemon without HTTP

## Notes

- Default behavior should remain unchanged (HTTP check for services with ports)
- Yellow is acceptable for daemons - this is a nice-to-have for visual consistency
