# fireup

A local development proxy for all your projects.

## Key Features

- Works with any web server (Node, Ruby, Python, Elixir, Go, Rust, etc.)
- Dynamic port allocation - no more port conflicts
- On-demand startup - services start when you access them
- Single or multi-service projects
- Subdomain support - `admin.myapp.test` passes through to your app
- Wildcard domains - `*.myapp.test` works too
- Static file serving
- HTTPS support with locally-trusted certificates (automatic CA generation)
- Web dashboard
- CLI for managing apps and services

## Video demo

Here's a 5 minute demo of fireup in action:

https://github.com/user-attachments/assets/c292a101-04b7-4fa7-80b9-abe0ac859d1b

## Installation

Requires Go 1.21+.

```bash
go install github.com/panozzaj/fireup/cmd/fireup@latest
```

This installs the `fireup` binary to your `$GOPATH/bin` (usually `~/go/bin`). Make sure this is in your PATH.

(I'm not super familiar with the Golang ecosystem, so if there's a better way to distribute this package or you run into issues, please open an issue or PR.)

## Quick Start

```bash
fireup setup
```

Then visit **http://fireup.test** to see the dashboard.

## Configuration

Place service config files in `~/.config/fireup/`. Unless specified otherwise, the filename (without extension) is the app name and domain.

### YAML config (recommended)

```yaml
# ~/.config/fireup/myproject.yml
root: ~/projects/myproject
cmd: bin/rails server -p $PORT -b 127.0.0.1
```

Your command receives the port via `$PORT` environment variable. fireup dynamically assigns ports, avoiding conflicts between apps.

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
# ~/.config/fireup/blog.yml
name: blog
root: ~/projects/blog
cmd: bundle exec jekyll serve --port $PORT --host 127.0.0.1 --livereload-port $((PORT + 1)) --watch
```

Note: Port numbers must be under 65535, so keep offsets small when fireup assigns high ports (50000+).

We may add first-class multi-port feature later.

### Static files

For serving static files, use a symlink to the directory:

```bash
ln -s ~/projects/my-site ~/.config/fireup/mysite
# Serves files from ~/projects/my-site at http://mysite.test
```

Or in YAML:

```yaml
# ~/.config/fireup/mysite.yml
root: ~/projects/my-site
static: true
```

### Fixed port proxy

If you're already running a server on a fixed port:

```bash
echo "3000" > ~/.config/fireup/myapp
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
sudo fireup install --tld dev
fireup serve --tld dev
# Visit http://myapp.dev
```

## CLI Commands

See `fireup --help` for a list of commands.

Run `fireup <command> --help` for command-specific options.

## Known Issues

> [!WARNING]
> This is a development tool only. Do not use in production.

### Docker Desktop

> [!CAUTION]
> Docker Desktop on macOS is not currently compatible with fireup's port forwarding.

Both Docker Desktop and fireup use macOS's `pf` (packet filter) for port redirection. When Docker starts or restarts, it clears fireup's pf rules, breaking access to `*.test` domains.

**Workaround:** After starting Docker Desktop, re-run:

```bash
fireup ports install
```

**Alternative:** Access apps directly via `localhost:9280` instead of `*.test` domains.

Pull requests to improve Docker compatibility are welcome.

## Claude Code Integration

If you use [Claude Code](https://github.com/anthropics/claude-code), you can install the `/fireup` slash command to help set up fireup configurations for your projects:

```bash
# Create the commands directory if it doesn't exist
mkdir -p ~/.claude/commands

# Symlink the command (adjust the path to where you cloned fireup)
ln -s /path/to/fireup/.claude/commands/fireup.md ~/.claude/commands/fireup.md
```

Then in any project, run `/fireup` in Claude Code to get help setting up fireup for that project.

## Status

**Version 0.9.0** - Pre-release

This tool is under active development. The setup and operation work on my Mac and meet my current needs across a variety of stacks. I plan to bump to version 1.0 once I get feedback that it works reliably on other machines and that setup is straightforward.

If you run into issues or have improvements, I'm open to PRs. For larger changes, consider opening an issue first to discuss the approach. I prefer smaller incremental PRs over large ones.

## License

MIT
