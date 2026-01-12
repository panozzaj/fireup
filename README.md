# roost-dev

A local development proxy for all your projects. Like [puma-dev](https://github.com/puma/puma-dev), but language-agnostic.

> **Warning**: This is a development tool only. Do not use in production.

## Features

- Works with any web server (Node, Ruby, Python, Elixir, Go, Rust, etc.)
- Dynamic port allocation - no more port conflicts
- On-demand startup - services start when you access them
- Subdomain support - `admin.myapp.localhost` passes through to your app
- Web dashboard at `roost-dev.localhost`

## Quick Start

```bash
# One-time install (forwards port 80 to roost-dev)
sudo roost-dev install

# Start the server
roost-dev serve

# Verify it's working
open http://roost-test.localhost

# Open the dashboard
open http://roost-dev.localhost
```

Then create configs in `~/.config/roost-dev/` for your apps (see Configuration below).

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
# Proxies http://myapp.localhost to localhost:3000
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
      API_URL: http://backend-myproject.localhost
```

Access at `http://frontend-myproject.localhost` and `http://backend-myproject.localhost`.

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
admin.myapp.localhost -> myapp (Host header: admin.myapp.localhost)
```

Your app reads the `Host` header to determine the subdomain.

## Using .test TLD

roost-dev includes a DNS server for custom TLDs:

```bash
# Install with .test TLD
sudo roost-dev install --tld test

# Run
roost-dev serve --tld test

# Visit http://myapp.test
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
```

Run `roost-dev <command> --help` for command-specific options.

## Running as a Background Service (macOS)

To have roost-dev start automatically on login and stay running:

```bash
# Build and install the binary
go build -o ~/go/bin/roost-dev ./cmd/roost-dev

# Create the LaunchAgent
cat > ~/Library/LaunchAgents/com.roost-dev.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.roost-dev</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/YOUR_USERNAME/go/bin/roost-dev</string>
        <string>serve</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/roost-dev/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/roost-dev/stderr.log</string>
</dict>
</plist>
EOF

# Replace YOUR_USERNAME with your actual username
sed -i '' "s/YOUR_USERNAME/$USER/g" ~/Library/LaunchAgents/com.roost-dev.plist

# Create logs directory
mkdir -p ~/Library/Logs/roost-dev

# Load the agent
launchctl load ~/Library/LaunchAgents/com.roost-dev.plist
```

To manage the service:

```bash
# Stop
launchctl unload ~/Library/LaunchAgents/com.roost-dev.plist

# Start
launchctl load ~/Library/LaunchAgents/com.roost-dev.plist

# View logs
tail -f ~/Library/Logs/roost-dev/stdout.log
```

## License

MIT
