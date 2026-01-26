# roost-dev tailscale command

## Goal

Add a `roost-dev tailscale` subcommand to simplify exposing roost-dev via Tailscale Serve.

## Commands

### `roost-dev tailscale serve`

Start Tailscale Serve to expose roost-dev to your tailnet.

```bash
roost-dev tailscale serve
# Runs: tailscale serve http://127.0.0.1:9280

roost-dev tailscale serve --bg
# Runs in background: tailscale serve --bg http://127.0.0.1:9280
```

Output should show:

- The Tailscale URL to access roost-dev
- Example paths for accessing apps (e.g., `/blog/`, `/api-myapp/`)

### `roost-dev tailscale status`

Show current Tailscale Serve configuration.

```bash
roost-dev tailscale status
# Runs: tailscale serve status
```

### `roost-dev tailscale stop`

Stop Tailscale Serve.

```bash
roost-dev tailscale stop
# Runs: tailscale serve reset
```

## Implementation notes

- Shell out to `tailscale` CLI (don't try to use Tailscale's Go library)
- Check if `tailscale` is installed and in PATH
- Get machine name from `tailscale status --json` for displaying the URL
- Use port 9280 (roost-dev's default HTTP port)
