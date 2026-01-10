# roost-dev

## Managing roost-dev during development

There are TWO versions of roost-dev:
1. **Installed version**: `/Users/anthony/go/bin/roost-dev` (managed by launchd)
2. **Local build**: `/Users/anthony/Documents/dev/roost-dev/roost-dev`

### Option 1: Use air for hot-reloading (recommended)

[air](https://github.com/air-verse/air) watches for file changes and auto-rebuilds. Config is in `.air.toml`.

```bash
# Install air (one-time)
go install github.com/air-verse/air@latest

# Stop the launchd-managed version and any running instances
launchctl bootout gui/$(id -u)/com.roost-dev 2>/dev/null || true
pkill -9 roost-dev 2>/dev/null || true

# Run with air (auto-rebuilds on file changes)
air
```

### Option 2: Manual build

```bash
# Stop the launchd-managed version and any running instances
launchctl bootout gui/$(id -u)/com.roost-dev 2>/dev/null || true
pkill -9 roost-dev 2>/dev/null || true

# Build the local version
unset GOPATH && go build -o /Users/anthony/Documents/dev/roost-dev/roost-dev ./cmd/roost-dev/

# Start the local version
/Users/anthony/Documents/dev/roost-dev/roost-dev &
```

### When done with development, restore launchd version:

```bash
pkill -9 roost-dev 2>/dev/null || true
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.roost-dev.plist
```

### Debug request handling:

```bash
# View server request logs
curl -s "http://roost-dev.test/api/server-logs" | jq -r '.[]'
```

## Code patterns

- **Use non-blocking operations in HTTP handlers.** For process management, prefer `StartAsync()` over `Start()` in API handlers so responses return immediately. The dashboard polls for status updates.
- **Avoid holding mutexes while waiting.** Release locks before any operation that could block (network calls, waiting for ports, etc.).
- **Always background server processes.** When starting roost-dev from bash, use `run_in_background: true` or append `&` to avoid blocking the conversation. Use `tee` to capture output: `/path/to/roost-dev 2>&1 | tee ./tmp/roost-dev.log &`
