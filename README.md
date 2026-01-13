# roost-dev

A local development proxy for all your projects. Like [puma-dev](https://github.com/puma/puma-dev), but language-agnostic.

> **Warning**: This is a development tool only. Do not use in production.

## Features

- Works with any web server (Node, Ruby, Python, Elixir, Go, Rust, etc.)
- Dynamic port allocation - no more port conflicts
- On-demand startup - services start when you access them
- Subdomain support - `admin.myapp.test` passes through to your app
- HTTPS support with locally-trusted certificates (automatic CA generation)
- Web dashboard at `roost-dev.test`

## Quick Start

```bash
# One-time install (forwards port 80 to roost-dev)
sudo roost-dev install

# Start the server (or use: roost-dev service install)
roost-dev serve
```

Then visit:
- **http://roost-test.test** - Verify it's working
- **http://roost-dev.test** - Open the dashboard

Create configs in `~/.config/roost-dev/` for your apps (see Configuration below).

## Configuration

Place config files in `~/.config/roost-dev/`. The filename becomes the app name.

### Command (recommended)

```bash
echo "npm run dev" > ~/.config/roost-dev/myapp
# roost-dev assigns a PORT, starts the command, and proxies to it
```

Your command receives the port via `$PORT` environment variable. This is the preferred method because roost-dev dynamically assigns ports, avoiding conflicts between apps.

### Static files

```bash
ln -s ~/projects/my-site ~/.config/roost-dev/mysite
# Serves files from the directory (must contain index.html)
```

### Fixed port proxy

```bash
echo "3000" > ~/.config/roost-dev/myapp
# Proxies http://myapp.test to localhost:3000
```

> **Note**: Fixed ports can conflict with other apps. Prefer using a command with `$PORT` when possible.

### YAML config

```yaml
# ~/.config/roost-dev/myproject.yml
name: myproject
description: My Project
root: ~/projects/myproject
cmd: bin/rails server -p $PORT -b 127.0.0.1
```

For multi-service projects:

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

```
roost-dev serve           Start the server
roost-dev list            List configured apps and their status
roost-dev start <app>     Start an app
roost-dev stop <app>      Stop an app
roost-dev restart <app>   Restart an app
roost-dev install         Setup port forwarding (requires sudo)
roost-dev uninstall       Remove port forwarding (requires sudo)
roost-dev service         Manage background service (install/uninstall/status)
roost-dev cert            Manage HTTPS certificates (install/uninstall/status)
```

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

## License

MIT
