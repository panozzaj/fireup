# roost-dev

A local development proxy for all your projects. Like [puma-dev](https://github.com/puma/puma-dev), but language-agnostic.

## Features

- Works with any web server (Node, Ruby, Python, Elixir, Go, Rust, etc.)
- Dynamic port allocation - no more port conflicts
- On-demand startup - services start when you access them
- Subdomain support - `admin.myapp.localhost` passes through to your app
- Web dashboard at `roost-dev.localhost`

## Quick Start

```bash
# One-time setup (forwards port 80 to roost-dev)
sudo ./roost-dev --setup

# Run without sudo
./roost-dev

# Create a config
echo "npm run dev" > ~/.config/roost-dev/myapp

# Visit http://myapp.localhost
```

## Configuration

Place config files in `~/.config/roost-dev/`. The filename becomes the app name.

### Port proxy

```bash
echo "3000" > ~/.config/roost-dev/myapp
# Proxies http://myapp.localhost to localhost:3000
```

### Command

```bash
echo "npm run dev" > ~/.config/roost-dev/myapp
# Starts command with PORT env var, proxies to it
```

### Static files

```bash
ln -s ~/projects/my-site ~/.config/roost-dev/mysite
# Serves files from the directory
```

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
# Setup with .test TLD
sudo ./roost-dev --setup --tld test

# Run
./roost-dev --tld test

# Visit http://myapp.test
```

## CLI Options

```
--dir <path>          Config directory (default: ~/.config/roost-dev)
--http-port <n>       HTTP port (default: 9080)
--advertise-port <n>  Port for URLs (default: 80)
--dns-port <n>        DNS server port (default: 9053)
--tld <domain>        Top-level domain (default: localhost)
--setup               Setup pf rules and DNS (requires sudo)
--cleanup             Remove pf rules and DNS (requires sudo)
```

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
    <key>Program</key>
    <string>/Users/YOUR_USERNAME/go/bin/roost-dev</string>
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
