# fireup

## Managing fireup during development

There are TWO versions of fireup:

1. **Installed version**: `/Users/anthony/go/bin/fireup` (managed by launchd)
2. **Local build**: `/Users/anthony/Documents/dev/fireup/fireup`

### Use restart script

```bash
# Rebuild and restart (uses launchd)
./scripts/restart.sh
```

### Service management commands

```bash
# Stop the background service
launchctl bootout gui/$(id -u)/com.fireup

# Reinstall the service (writes new plist and loads it)
./fireup service install

# Or if using installed version:
fireup service install
```

The `service install` command captures your current `PATH`, `HOME`, `USER`, etc. and writes them to the LaunchAgent plist. This ensures spawned processes have access to tools like nvm, rbenv, etc.

### Setup/teardown wizards

```bash
# Interactive setup (port forwarding, CA cert, background service)
fireup setup

# Interactive teardown (reverse of setup)
fireup teardown
```

Both wizards prompt for confirmation before each step and show which steps are already done.

### Debug request handling:

```bash
# View server request logs
curl -s "http://fireup.test/api/server-logs" | jq -r '.[]'
```

## Code patterns

- **Use non-blocking operations in HTTP handlers.** For process management, prefer `StartAsync()` over `Start()` in API handlers so responses return immediately. The dashboard polls for status updates.
- **Avoid holding mutexes while waiting.** Release locks before any operation that could block (network calls, waiting for ports, etc.).
- **Always background server processes.** When starting fireup from bash, use `run_in_background: true` or append `&` to avoid blocking the conversation. Use `tee` to capture output: `/path/to/fireup 2>&1 | tee ./tmp/fireup.log &`

## Useful URLs

- **Dashboard**: http://fireup.test
- **Icon test page**: http://fireup.test/icons (for previewing icon options)
- **ASCII logo generator**: https://patorjk.com/software/taag/#p=testall&t=fireup (for CLI logo)

## UI patterns

- **Use CSS tooltips, not title attributes.** For tooltips, use `data-tooltip="..."` instead of `title="..."`. CSS tooltips appear instantly on hover, while native title tooltips have a ~500ms delay. The CSS is already set up: any element with `data-tooltip` will show the tooltip on hover.
- **Icon buttons must have hover tooltips.** Any button that uses an icon (instead of or in addition to text) must have a `data-tooltip` attribute providing a descriptive tooltip explaining what the button does.
