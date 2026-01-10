package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/dns"
	"github.com/panozzaj/roost-dev/internal/server"
)

// GlobalConfig stores persistent settings
type GlobalConfig struct {
	TLD string `json:"tld"`
}

var (
	version = "dev"
)

const asciiLogo = `                 __            __
   _________  ____  _____/ /_      ____/ /__ _   __
  / ___/ __ \/ __ \/ ___/ __/_____/ __  / _ \ | / /
 / /  / /_/ / /_/ (__  ) /_/_____/ /_/ /  __/ |/ /
/_/   \____/\____/____/\__/      \__,_/\___/|___/`

func printLogo() {
	if os.Getenv("CLAUDECODE") != "1" {
		fmt.Println(asciiLogo)
	}
}

func main() {
	// Check for subcommands first (anything that doesn't start with -)
	if len(os.Args) >= 2 && !strings.HasPrefix(os.Args[1], "-") {
		switch os.Args[1] {
		case "start", "stop", "restart":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: roost-dev %s <app-name>\n", os.Args[1])
				os.Exit(1)
			}
			if err := runCommand(os.Args[1], os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "help":
			printUsage()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}

	// Flags
	var (
		configDir     string
		httpPort      int
		httpsPort     int
		advertisePort int
		dnsPort       int
		tld           string
		showHelp      bool
		showVer       bool
		doSetup       bool
		doCleanup     bool
	)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	flag.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")
	flag.IntVar(&httpPort, "http-port", 9280, "HTTP port to listen on")
	flag.IntVar(&httpsPort, "https-port", 9443, "HTTPS port to listen on")
	flag.IntVar(&advertisePort, "advertise-port", 80, "Port to use in URLs (0 = same as http-port)")
	flag.IntVar(&dnsPort, "dns-port", 9053, "DNS server port")
	flag.StringVar(&tld, "tld", "", "Top-level domain to use (default: from config or 'localhost')")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVer, "version", false, "Show version")
	flag.BoolVar(&doSetup, "setup", false, "Setup pf rules for port forwarding (requires sudo)")
	flag.BoolVar(&doCleanup, "cleanup", false, "Remove pf rules (requires sudo)")

	// Use our custom usage function so flag errors show subcommands
	flag.Usage = printUsage
	flag.Parse()

	// Load saved config for TLD default
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		log.Printf("Warning: could not load config: %v", err)
		globalCfg = &GlobalConfig{TLD: "localhost"}
	}
	if tld == "" {
		tld = globalCfg.TLD
	}

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	if showVer {
		fmt.Printf("roost-dev %s\n", version)
		os.Exit(0)
	}

	if doSetup {
		if err := runSetup(configDir, httpPort, dnsPort, tld); err != nil {
			log.Fatalf("Setup failed: %v", err)
		}
		os.Exit(0)
	}

	if doCleanup {
		if err := runCleanup(tld); err != nil {
			log.Fatalf("Cleanup failed: %v", err)
		}
		os.Exit(0)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Load configuration
	urlPort := advertisePort
	if urlPort == 0 {
		urlPort = httpPort
	}
	cfg := &config.Config{
		Dir:       configDir,
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		URLPort:   urlPort,
		TLD:       tld,
	}

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		srv.Shutdown()
		os.Exit(0)
	}()

	printLogo()
	fmt.Printf("roost-dev %s\n", version)
	fmt.Printf("Configuration directory: %s\n", configDir)
	fmt.Printf("Listening on http://127.0.0.1:%d\n", httpPort)
	if urlPort == 80 {
		fmt.Printf("Dashboard at http://roost-dev.%s\n", tld)
	} else {
		fmt.Printf("Dashboard at http://roost-dev.%s:%d\n", tld, urlPort)
	}

	// Start DNS server for custom TLDs
	if tld != "localhost" {
		dnsServer := dns.New(dnsPort, tld)
		go func() {
			if err := dnsServer.Start(); err != nil {
				log.Printf("DNS server error: %v", err)
			}
		}()
		fmt.Printf("DNS server on 127.0.0.1:%d for *.%s\n", dnsPort, tld)
	}
	fmt.Println()

	// Warn if pf rules aren't set up but we're using port forwarding defaults
	if httpPort != urlPort && urlPort == 80 {
		if _, err := os.Stat(pfAnchorPath); os.IsNotExist(err) {
			yellow := "\033[33m"
			reset := "\033[0m"
			fmt.Println(yellow + "WARNING: URLs like http://myapp.localhost won't work yet.")
			fmt.Println("")
			fmt.Println("  roost-dev is running on port 9080, but your browser will")
			fmt.Println("  try port 80. Run this once to set up the redirect:")
			fmt.Println("")
			fmt.Println("    sudo roost-dev --setup")
			fmt.Println(reset)
		}
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func printUsage() {
	printLogo()
	fmt.Println(`
roost-dev - Local development proxy for all your projects

USAGE:
    roost-dev [OPTIONS]
    roost-dev <command> <app-name>

COMMANDS:
    start <app>     Start an app
    stop <app>      Stop an app
    restart <app>   Restart an app

OPTIONS:
    --dir <path>          Configuration directory (default: ~/.config/roost-dev)
    --http-port <n>       HTTP port to listen on (default: 9080)
    --https-port <n>      HTTPS port to listen on (default: 9443)
    --advertise-port <n>  Port to use in URLs (default: 80)
    --dns-port <n>        DNS server port (default: 9053)
    --tld <domain>        Top-level domain to use (default: localhost)
    --setup               Setup pf/DNS for the specified TLD (requires sudo)
    --cleanup             Remove pf/DNS configuration (requires sudo)
    --help                Show this help
    --version             Show version

CONFIGURATION:
    Place config files in ~/.config/roost-dev/

    Simple port file:
        echo "3000" > ~/.config/roost-dev/myapp
        # Access at http://myapp.localhost

    Command file:
        echo "npm run dev" > ~/.config/roost-dev/myapp
        # roost-dev starts the command with PORT env var

    Static file path:
        echo "/path/to/index.html" > ~/.config/roost-dev/mysite
        # Serves static files

    YAML config (for multi-service projects):
        # ~/.config/roost-dev/myproject.yml
        name: myproject
        root: ~/projects/myproject
        services:
          backend:
            dir: backend
            cmd: mix phx.server
          frontend:
            cmd: npm start
            env:
              API_URL: http://backend-myproject.localhost

        # Access at http://frontend-myproject.localhost
        # Access at http://backend-myproject.localhost

SETUP (recommended):
    # One-time setup for .localhost (pf rules only)
    sudo roost-dev --setup

    # Or setup for .test TLD (pf rules + DNS resolver)
    sudo roost-dev --setup --tld test

    # Then just run roost-dev (no sudo needed)
    roost-dev              # for .localhost
    roost-dev --tld test   # for .test

    # Remove configuration
    sudo roost-dev --cleanup
    sudo roost-dev --cleanup --tld test

EXAMPLES:
    # After running --setup, just start roost-dev
    roost-dev

    # Use .test TLD (requires setup with --tld test first)
    roost-dev --tld test

    # Or run with sudo on port 80 directly (no setup needed)
    sudo roost-dev --http-port 80 --advertise-port 80`)
}

const (
	pfAnchorPath     = "/etc/pf.anchors/roost-dev"
	launchdPlistPath = "/Library/LaunchDaemons/dev.roost.pfctl.plist"
	globalConfigName = "config.json"
)

func loadGlobalConfig(configDir string) (*GlobalConfig, error) {
	path := filepath.Join(configDir, globalConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{TLD: "localhost"}, nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveGlobalConfig(configDir string, cfg *GlobalConfig) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(configDir, globalConfigName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func runSetup(configDir string, targetPort, dnsPort int, tld string) error {
	fmt.Println("Setting up roost-dev...")

	// Save TLD to config so we don't need --tld flag every time
	if err := saveGlobalConfig(configDir, &GlobalConfig{TLD: tld}); err != nil {
		fmt.Printf("Warning: could not save config: %v\n", err)
	}

	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("setup requires root privileges. Run with sudo")
	}

	// Create the pf anchor file
	anchorContent := `# roost-dev port forwarding rules
# Forward port 80 to 9280 for roost-dev
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 9280
`
	fmt.Printf("Creating %s...\n", pfAnchorPath)
	if err := os.WriteFile(pfAnchorPath, []byte(anchorContent), 0644); err != nil {
		return fmt.Errorf("writing anchor file: %w", err)
	}

	// Check if pf.conf needs to be updated
	pfConf, err := os.ReadFile("/etc/pf.conf")
	if err != nil {
		return fmt.Errorf("reading /etc/pf.conf: %w", err)
	}

	pfConfStr := string(pfConf)
	needsUpdate := false

	// Check if our anchor is already referenced
	if !strings.Contains(pfConfStr, "roost-dev") {
		needsUpdate = true

		// Create backup before modifying
		backupPath := "/etc/pf.conf.roost-dev-backup"
		fmt.Printf("Backing up /etc/pf.conf to %s...\n", backupPath)
		if err := os.WriteFile(backupPath, pfConf, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}

		fmt.Println("Updating /etc/pf.conf to include roost-dev anchor...")

		// Add anchor reference after the com.apple anchor line
		lines := strings.Split(pfConfStr, "\n")
		var newLines []string
		for _, line := range lines {
			newLines = append(newLines, line)
			if strings.Contains(line, `rdr-anchor "com.apple/*"`) {
				newLines = append(newLines, `rdr-anchor "roost-dev"`)
			}
			if strings.Contains(line, `load anchor "com.apple" from "/etc/pf.anchors/com.apple"`) {
				newLines = append(newLines, `load anchor "roost-dev" from "/etc/pf.anchors/roost-dev"`)
			}
		}

		if err := os.WriteFile("/etc/pf.conf", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return fmt.Errorf("writing /etc/pf.conf: %w", err)
		}
	}

	// Create launchd plist for loading pf rules on boot
	launchdContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.roost.pfctl</string>
    <key>ProgramArguments</key>
    <array>
        <string>/sbin/pfctl</string>
        <string>-e</string>
        <string>-f</string>
        <string>/etc/pf.conf</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`
	fmt.Printf("Creating %s...\n", launchdPlistPath)
	if err := os.WriteFile(launchdPlistPath, []byte(launchdContent), 0644); err != nil {
		return fmt.Errorf("writing launchd plist: %w", err)
	}

	// Enable pf and load the rules now
	fmt.Println("Enabling pf and loading rules...")
	cmd := exec.Command("/sbin/pfctl", "-e", "-f", "/etc/pf.conf")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// pfctl returns error if pf is already enabled, which is fine
		fmt.Println("Note: pf may already be enabled")
	}

	// Load the anchor specifically
	cmd = exec.Command("/sbin/pfctl", "-a", "roost-dev", "-f", pfAnchorPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("loading anchor: %w", err)
	}

	// Create DNS resolver file for custom TLD
	if tld != "localhost" {
		resolverDir := "/etc/resolver"
		if err := os.MkdirAll(resolverDir, 0755); err != nil {
			return fmt.Errorf("creating resolver directory: %w", err)
		}

		resolverPath := fmt.Sprintf("%s/%s", resolverDir, tld)
		resolverContent := fmt.Sprintf("nameserver 127.0.0.1\nport %d\n", dnsPort)
		fmt.Printf("Creating %s...\n", resolverPath)
		if err := os.WriteFile(resolverPath, []byte(resolverContent), 0644); err != nil {
			return fmt.Errorf("writing resolver file: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Port 80 is now forwarded to port 9280.")
	fmt.Printf("TLD '%s' saved to config.\n", tld)
	fmt.Println()
	fmt.Println("You can now run roost-dev without sudo:")
	fmt.Println()
	fmt.Println("    roost-dev")
	fmt.Println()
	fmt.Printf("Then access your apps at http://appname.%s\n", tld)
	if needsUpdate {
		fmt.Println()
		fmt.Println("Note: /etc/pf.conf was modified. Backup saved to /etc/pf.conf.roost-dev-backup")
	}

	return nil
}

func runCommand(cmd, appName string) error {
	// Load config to get TLD
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		globalCfg = &GlobalConfig{TLD: "localhost"}
	}

	// Show action in progress
	switch cmd {
	case "start":
		fmt.Printf("Starting %s...\n", appName)
	case "stop":
		fmt.Printf("Stopping %s...\n", appName)
	case "restart":
		fmt.Printf("Restarting %s...\n", appName)
	}

	// Make request to roost-dev API
	url := fmt.Sprintf("http://roost-dev.%s/api/%s?name=%s", globalCfg.TLD, cmd, appName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to roost-dev: %v (is it running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Show completion message
	switch cmd {
	case "start":
		fmt.Printf("%s started\n", appName)
	case "stop":
		fmt.Printf("%s stopped\n", appName)
	case "restart":
		fmt.Printf("%s restarted\n", appName)
	}
	return nil
}

func runCleanup(tld string) error {
	fmt.Println("Removing roost-dev configuration...")

	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("cleanup requires root privileges. Run with sudo")
	}

	// Flush our anchor
	fmt.Println("Flushing roost-dev anchor...")
	cmd := exec.Command("/sbin/pfctl", "-a", "roost-dev", "-F", "all")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors if anchor doesn't exist

	// Remove the anchor file
	if _, err := os.Stat(pfAnchorPath); err == nil {
		fmt.Printf("Removing %s...\n", pfAnchorPath)
		os.Remove(pfAnchorPath)
	}

	// Remove the launchd plist
	if _, err := os.Stat(launchdPlistPath); err == nil {
		fmt.Printf("Removing %s...\n", launchdPlistPath)
		// Unload the launchd job first
		exec.Command("/bin/launchctl", "unload", launchdPlistPath).Run()
		os.Remove(launchdPlistPath)
	}

	// Remove DNS resolver file for custom TLD
	if tld != "localhost" {
		resolverPath := fmt.Sprintf("/etc/resolver/%s", tld)
		if _, err := os.Stat(resolverPath); err == nil {
			fmt.Printf("Removing %s...\n", resolverPath)
			os.Remove(resolverPath)
		}
	}

	fmt.Println()
	fmt.Println("Cleanup complete!")
	fmt.Println()
	fmt.Println("Note: /etc/pf.conf was not modified. To fully remove roost-dev:")
	fmt.Println("  1. Remove the roost-dev anchor lines from /etc/pf.conf")
	fmt.Println("  2. Or restore from backup: sudo cp /etc/pf.conf.roost-dev-backup /etc/pf.conf")

	return nil
}
