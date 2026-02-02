# fireup tailscale command

## Goal

Add a `fireup tailscale` subcommand to simplify exposing fireup via Tailscale Serve.

## Commands

### `fireup tailscale serve`

Start Tailscale Serve to expose fireup to your tailnet.

```bash
fireup tailscale serve
# Runs: tailscale serve http://127.0.0.1:9280

fireup tailscale serve --bg
# Runs in background: tailscale serve --bg http://127.0.0.1:9280
```

Output should show:

- The Tailscale URL to access fireup
- Example paths for accessing apps (e.g., `/blog/`, `/api-myapp/`)

### `fireup tailscale status`

Show current Tailscale Serve configuration.

```bash
fireup tailscale status
# Runs: tailscale serve status
```

### `fireup tailscale stop`

Stop Tailscale Serve.

```bash
fireup tailscale stop
# Runs: tailscale serve reset
```

## Implementation notes

- Shell out to `tailscale` CLI (don't try to use Tailscale's Go library)
- Check if `tailscale` is installed and in PATH
- Get machine name from `tailscale status --json` for displaying the URL
- Use port 9280 (fireup's default HTTP port)
