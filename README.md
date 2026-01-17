# roost-dev

A local development proxy for all your projects. Like [puma-dev](https://github.com/puma/puma-dev), but language-agnostic.

> [!WARNING]
> This is a development tool only. Do not use in production.

## Key Features

- Works with any web server (Node, Ruby, Python, Elixir, Go, Rust, etc.)
- Dynamic port allocation - no more port conflicts
- On-demand startup - services start when you access them
- Subdomain support - `admin.myapp.test` passes through to your app
- Wildcard domains - `*.myapp.test` works too
- Static file serving
- HTTPS support with locally-trusted certificates (automatic CA generation)
- Web dashboard
- CLI for managing apps and services

## Installation

Requires Go 1.21+.

```bash
go install github.com/panozzaj/roost-dev/cmd/roost-dev@latest
```

This installs the `roost-dev` binary to your `$GOPATH/bin` (usually `~/go/bin`). Make sure this is in your PATH.

## Quick Start

```bash
# One-time setup
sudo roost-dev setup
```

Then visit **http://roost-dev.test** to see the dashboard.

Create configs in `~/.config/roost-dev/` for your apps (see Configuration below).

## Configuration

Place config files in `~/.config/roost-dev/`. The filename becomes the app name.

### YAML config (recommended)

```yaml
# ~/.config/roost-dev/myproject.yml
root: ~/projects/myproject
cmd: bin/rails server -p $PORT -b 127.0.0.1
```

Your command receives the port via `$PORT` environment variable. roost-dev dynamically assigns ports, avoiding conflicts between apps.

Optional fields:

```yaml
name: myproject # defaults to filename without extension
description: My App # shown on dashboard
```

### Multi-service projects

For projects with multiple services:

```yaml
name: myproject
root: ~/projects/myproject
services:
    backend:
        cmd: bin/rails server -p $PORT
    frontend:
        cmd: npm start
        depends_on: [backend]
        env:
            API_URL: http://backend-myproject.test
```

Access at `http://frontend-myproject.test` and `http://backend-myproject.test`.

Services with `depends_on` will automatically start their dependencies first.

### Multiple ports

Some tools need multiple ports (e.g., Jekyll with livereload). Use shell arithmetic on `$PORT`:

```yaml
# ~/.config/roost-dev/blog.yml
name: blog
root: ~/projects/blog
cmd: bundle exec jekyll serve --port $PORT --host 127.0.0.1 --livereload-port $((PORT + 1)) --watch
```

Note: Port numbers must be under 65535, so keep offsets small when roost-dev assigns high ports (50000+).

### Static files

For serving static files, use a symlink to the directory:

```bash
ln -s ~/projects/my-site ~/.config/roost-dev/mysite
# Serves files from ~/projects/my-site at http://mysite.test
```

Or in YAML:

```yaml
# ~/.config/roost-dev/mysite.yml
root: ~/projects/my-site
static: true
```

Static sites show an HTML5 icon on the dashboard (since they're always available).

### Fixed port proxy

If you're already running a server on a fixed port:

```bash
echo "3000" > ~/.config/roost-dev/myapp
# Proxies http://myapp.test to localhost:3000
```

> **Note**: Fixed ports can conflict between apps. Prefer YAML with `$PORT` when possible.

## Subdomains

Subdomains are passed through to your app:

```
admin.myapp.test -> myapp (Host header: admin.myapp.test)
```

Your app reads the `Host` header to determine the subdomain.

## Using a Different TLD

The default TLD is `.test`. To use a different one (e.g., `.dev`):

```bash
sudo roost-dev install --tld dev
roost-dev serve --tld dev
# Visit http://myapp.dev
```

## CLI Commands

See `roost-dev --help` for a list of commands.

Run `roost-dev <command> --help` for command-specific options.

## HTTPS Support

roost-dev supports HTTPS with automatic certificate generation for any domain.

```bash
# Generate CA and trust it (one-time, prompts for password)
roost-dev cert install

# Restart roost-dev to enable HTTPS
roost-dev service uninstall && roost-dev service install

# Restart your browser to pick up the new CA
```

Now you can access your apps via HTTPS:

- **https://myapp.test** - Your app with HTTPS
- **https://roost-dev.test** - Dashboard with HTTPS
- **https://anyapp.test** - Any domain works automatically

Certificates are generated on-demand for each domain. Both HTTP and HTTPS work simultaneously.

To check certificate status:

```bash
roost-dev cert status
```

## Running as a Background Service (macOS)

To have roost-dev start automatically on login and stay running:

```bash
roost-dev service install
```

This creates a LaunchAgent that runs `roost-dev serve` automatically. The service will restart if it crashes.

To manage the service:

```bash
roost-dev service status      # Check if running
roost-dev service uninstall   # Stop and remove
```

Logs are written to `~/Library/Logs/roost-dev/`.

## Known Issues

### Docker Desktop

> [!CAUTION]
> Docker Desktop on macOS is not currently compatible with roost-dev's port forwarding.

Both Docker Desktop and roost-dev use macOS's `pf` (packet filter) for port redirection. When Docker starts or restarts, it clears roost-dev's pf rules, breaking access to `*.test` domains.

**Workaround:** After starting Docker Desktop, re-run:

```bash
roost-dev ports install
```

**Alternative:** Access apps directly via `localhost:9280` instead of `*.test` domains.

Pull requests to improve Docker compatibility are welcome.

## License

MIT
