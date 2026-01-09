# roost-dev

After making changes that affect runtime behavior, rebuild and restart:

```bash
GOPATH=/Users/anthony/go go build -o ~/go/bin/roost-dev ./cmd/roost-dev && launchctl kickstart -k gui/$(id -u)/com.roost-dev
```

## Code patterns

- **Use non-blocking operations in HTTP handlers.** For process management, prefer `StartAsync()` over `Start()` in API handlers so responses return immediately. The dashboard polls for status updates.
- **Avoid holding mutexes while waiting.** Release locks before any operation that could block (network calls, waiting for ports, etc.).
