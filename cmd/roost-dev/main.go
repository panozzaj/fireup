package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/dns"
	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/server"
)

// GlobalConfig stores persistent settings
type GlobalConfig struct {
	TLD           string        `json:"tld"`
	Ollama        *OllamaConfig `json:"ollama,omitempty"`
	ClaudeCommand string        `json:"claude_command,omitempty"` // Command to run Claude Code (default: "claude")
}

// OllamaConfig stores settings for local LLM error analysis
type OllamaConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`   // e.g., "http://localhost:11434"
	Model   string `json:"model"` // e.g., "llama3.2"
}

var (
	version = "dev"
)

func printLogo() {
	if os.Getenv("CLAUDECODE") != "1" {
		fmt.Println(logo.CLI())
	}
}

func main() {
	// Handle no args or help
	if len(os.Args) < 2 {
		printMainUsage()
		os.Exit(0)
	}

	// Handle global flags before command
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printMainUsage()
		os.Exit(0)
	}
	if os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version" {
		fmt.Printf("roost-dev %s\n", version)
		os.Exit(0)
	}

	// Route to command
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "serve":
		cmdServe(args)
	case "list", "ls":
		cmdList(args)
	case "start":
		cmdAppControl("start", args)
	case "stop":
		cmdAppControl("stop", args)
	case "restart":
		cmdAppControl("restart", args)
	case "install":
		cmdInstall(args)
	case "uninstall":
		cmdUninstall(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\nRun 'roost-dev help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func printMainUsage() {
	printLogo()
	fmt.Println(`
roost-dev - Local development proxy for all your projects

USAGE:
    roost-dev <command> [options]

COMMANDS:
    serve             Start the roost-dev server
    list, ls          List configured apps and their status
    start <app>       Start an app
    stop <app>        Stop an app
    restart <app>     Restart an app
    install           Setup port forwarding (requires sudo)
    uninstall         Remove port forwarding config (requires sudo)
    help              Show this help
    version           Show version

Run 'roost-dev <command> --help' for command-specific options.

QUICK START:
    sudo roost-dev install        # One-time setup
    roost-dev serve               # Start the server
    echo "npm start" > ~/.config/roost-dev/myapp
    # Visit http://myapp.localhost`)
}

// cmdServe handles the 'serve' command
func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	var (
		configDir     string
		httpPort      int
		httpsPort     int
		advertisePort int
		dnsPort       int
		tld           string
	)

	fs.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")
	fs.IntVar(&httpPort, "http-port", 9280, "HTTP port to listen on")
	fs.IntVar(&httpsPort, "https-port", 9443, "HTTPS port to listen on")
	fs.IntVar(&advertisePort, "advertise-port", 80, "Port to use in URLs (0 = same as http-port)")
	fs.IntVar(&dnsPort, "dns-port", 9053, "DNS server port")
	fs.StringVar(&tld, "tld", "", "Top-level domain (default: from config or 'localhost')")

	fs.Usage = func() {
		fmt.Println(`roost-dev serve - Start the roost-dev server

USAGE:
    roost-dev serve [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
CONFIGURATION:
    Place config files in ~/.config/roost-dev/

    Command (recommended):
        echo "npm run dev" > ~/.config/roost-dev/myapp
        # roost-dev assigns a dynamic PORT, avoiding conflicts

    Static site (symlink to directory):
        ln -s ~/projects/my-site ~/.config/roost-dev/mysite
        # Directory must contain index.html

    Fixed port proxy (not recommended):
        echo "3000" > ~/.config/roost-dev/myapp
        # Fixed ports can conflict; prefer commands with $PORT

    YAML config (for multi-service projects):
        name: myproject
        root: ~/projects/myproject
        services:
          backend:
            cmd: mix phx.server -p $PORT
          frontend:
            cmd: npm start
            env:
              API_URL: http://backend-myproject.localhost

    Commands receive the port via the $PORT environment variable.
    Your command should listen on this port (e.g., "rails server -p $PORT").`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	// Load saved config for TLD default
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		log.Printf("Warning: could not load config: %v", err)
		globalCfg = &GlobalConfig{TLD: "localhost"}
	}
	if tld == "" {
		tld = globalCfg.TLD
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

	// Convert Ollama config
	var ollamaCfg *config.OllamaConfig
	if globalCfg.Ollama != nil && globalCfg.Ollama.Enabled {
		ollamaCfg = &config.OllamaConfig{
			Enabled: globalCfg.Ollama.Enabled,
			URL:     globalCfg.Ollama.URL,
			Model:   globalCfg.Ollama.Model,
		}
	}

	// Get Claude command with default
	claudeCmd := globalCfg.ClaudeCommand
	if claudeCmd == "" {
		claudeCmd = "claude"
	}

	cfg := &config.Config{
		Dir:           configDir,
		HTTPPort:      httpPort,
		HTTPSPort:     httpsPort,
		URLPort:       urlPort,
		TLD:           tld,
		Ollama:        ollamaCfg,
		ClaudeCommand: claudeCmd,
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
			fmt.Println("  roost-dev is running on port 9280, but your browser will")
			fmt.Println("  try port 80. Run this once to set up the redirect:")
			fmt.Println("")
			fmt.Println("    sudo roost-dev install")
			fmt.Println(reset)
		}
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// cmdList handles the 'list' command
func cmdList(args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev list - List configured apps and their status

USAGE:
    roost-dev list

Shows all configured apps, their running status, and URLs.
If the server is not running, shows config files only.`)
			os.Exit(0)
		}
	}

	if err := runList(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdAppControl handles start/stop/restart commands
func cmdAppControl(action string, args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Printf(`roost-dev %s - %s an app

USAGE:
    roost-dev %s <app-name>

Requires the roost-dev server to be running.
`, action, strings.Title(action), action)
			os.Exit(0)
		}
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: roost-dev %s <app-name>\n", action)
		os.Exit(1)
	}

	if err := runCommand(action, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdInstall handles the 'install' command
func cmdInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	var (
		tld       string
		configDir string
	)

	fs.StringVar(&tld, "tld", "localhost", "Top-level domain to configure")
	fs.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`roost-dev install - Setup port forwarding for roost-dev

USAGE:
    sudo roost-dev install [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Sets up macOS pf (packet filter) rules to forward port 80 to roost-dev.
    This allows accessing apps at http://myapp.localhost without specifying a port.

    For custom TLDs like .test, also creates a DNS resolver file.

EXAMPLES:
    sudo roost-dev install              # Setup for .localhost
    sudo roost-dev install --tld test   # Setup for .test TLD`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	if err := runSetup(configDir, 9280, 9053, tld); err != nil {
		log.Fatalf("Install failed: %v", err)
	}
}

// cmdUninstall handles the 'uninstall' command
func cmdUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)

	var tld string
	fs.StringVar(&tld, "tld", "localhost", "Top-level domain to remove")

	fs.Usage = func() {
		fmt.Println(`roost-dev uninstall - Remove roost-dev port forwarding config

USAGE:
    sudo roost-dev uninstall [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Removes the pf anchor file and DNS resolver (if using custom TLD).
    Does not modify /etc/pf.conf - you may want to manually remove
    the roost-dev lines or restore from the backup.

EXAMPLES:
    sudo roost-dev uninstall              # Remove .localhost config
    sudo roost-dev uninstall --tld test   # Remove .test TLD config`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	if err := runCleanup(tld); err != nil {
		log.Fatalf("Uninstall failed: %v", err)
	}
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

func checkInstallConflicts(tld string) error {
	fmt.Println("Checking for conflicts...")
	var warnings []string

	// Check for puma-dev
	if _, err := os.Stat("/etc/resolver/dev"); err == nil {
		warnings = append(warnings, "puma-dev resolver found at /etc/resolver/dev")
	}
	if _, err := os.Stat("/etc/pf.anchors/com.apple.puma-dev"); err == nil {
		warnings = append(warnings, "puma-dev pf anchor found at /etc/pf.anchors/com.apple.puma-dev")
	}

	// Check if something is listening on port 80
	conn, err := net.DialTimeout("tcp", "127.0.0.1:80", 500*time.Millisecond)
	if err == nil {
		conn.Close()
		warnings = append(warnings, "something is already listening on port 80")
	}

	// Check for existing resolver that might conflict
	resolverPath := fmt.Sprintf("/etc/resolver/%s", tld)
	if _, err := os.Stat(resolverPath); err == nil {
		// Read it to see if it's ours
		data, _ := os.ReadFile(resolverPath)
		if !strings.Contains(string(data), "roost-dev") {
			warnings = append(warnings, fmt.Sprintf("existing resolver at %s (not from roost-dev)", resolverPath))
		}
	}

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
		fmt.Println("\nThese may conflict with roost-dev. Consider removing them first.")
		fmt.Print("Continue anyway? [y/N]: ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("installation cancelled")
		}
	}

	return nil
}

func runSetup(configDir string, targetPort, dnsPort int, tld string) error {
	fmt.Println("Installing roost-dev...")

	// Check for conflicts first (before requiring sudo)
	if err := checkInstallConflicts(tld); err != nil {
		return err
	}

	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("install requires root privileges. Run with: sudo roost-dev install")
	}

	// Save TLD to config so we don't need --tld flag every time
	if err := saveGlobalConfig(configDir, &GlobalConfig{TLD: tld}); err != nil {
		fmt.Printf("Warning: could not save config: %v\n", err)
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
		resolverContent := fmt.Sprintf("# Generated by roost-dev\nnameserver 127.0.0.1\nport %d\n", dnsPort)
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
	fmt.Println("    roost-dev serve")
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

// AppStatus represents the status of a single app from the API
type AppStatus struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	URL         string      `json:"url"`
	Aliases     []string    `json:"aliases,omitempty"`
	Description string      `json:"description,omitempty"`
	Running     bool        `json:"running,omitempty"`
	Port        int         `json:"port,omitempty"`
	Uptime      string      `json:"uptime,omitempty"`
	Services    []SvcStatus `json:"services,omitempty"`
}

// SvcStatus represents the status of a service within a multi-service app
type SvcStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Port    int    `json:"port,omitempty"`
	Uptime  string `json:"uptime,omitempty"`
	URL     string `json:"url"`
	Default bool   `json:"default,omitempty"`
}

func runList() error {
	// Load config to get TLD
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		globalCfg = &GlobalConfig{TLD: "localhost"}
	}

	// Try to get status from running server
	url := fmt.Sprintf("http://roost-dev.%s/api/status", globalCfg.TLD)
	resp, err := http.Get(url)
	if err != nil {
		// Server not running - fall back to listing config files
		return listConfigFiles(configDir, globalCfg.TLD)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return listConfigFiles(configDir, globalCfg.TLD)
	}

	var apps []AppStatus
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return fmt.Errorf("failed to parse status: %v", err)
	}

	if len(apps) == 0 {
		fmt.Println("No apps configured.")
		fmt.Printf("Add configs to %s\n", configDir)
		return nil
	}

	// Print header
	fmt.Printf("%-25s %-10s %s\n", "APP", "STATUS", "URL")
	fmt.Printf("%-25s %-10s %s\n", strings.Repeat("-", 25), strings.Repeat("-", 10), strings.Repeat("-", 30))

	for _, app := range apps {
		var status string
		if app.Type == "multi-service" {
			// For multi-service apps, show how many services are running
			runningCount := 0
			for _, svc := range app.Services {
				if svc.Running {
					runningCount++
				}
			}
			if runningCount == 0 {
				status = "stopped"
			} else if runningCount == len(app.Services) {
				status = "running"
			} else {
				status = fmt.Sprintf("%d/%d", runningCount, len(app.Services))
			}
		} else {
			if app.Running {
				status = "running"
			} else {
				status = "stopped"
			}
		}

		// Pad status first, then add color codes (so ANSI codes don't affect width)
		paddedStatus := fmt.Sprintf("%-10s", status)
		switch {
		case status == "running":
			paddedStatus = "\033[32m" + paddedStatus + "\033[0m" // green
		case status == "stopped":
			paddedStatus = "\033[90m" + paddedStatus + "\033[0m" // gray
		case strings.Contains(status, "/"):
			paddedStatus = "\033[33m" + paddedStatus + "\033[0m" // yellow for partial
		}

		name := app.Name
		if len(app.Aliases) > 0 {
			name = fmt.Sprintf("%s (%s)", app.Name, strings.Join(app.Aliases, ", "))
		}
		fmt.Printf("%-25s %s %s\n", name, paddedStatus, app.URL)
	}

	return nil
}

func listConfigFiles(configDir, tld string) error {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No apps configured.")
			fmt.Printf("Add configs to %s\n", configDir)
			return nil
		}
		return err
	}

	var apps []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files and config.json
		if strings.HasPrefix(name, ".") || name == "config.json" {
			continue
		}
		// Remove .yml/.yaml extension for display
		name = strings.TrimSuffix(name, ".yml")
		name = strings.TrimSuffix(name, ".yaml")
		apps = append(apps, name)
	}

	if len(apps) == 0 {
		fmt.Println("No apps configured.")
		fmt.Printf("Add configs to %s\n", configDir)
		return nil
	}

	fmt.Println("Configured apps (server not running):")
	fmt.Printf("%-20s %s\n", "APP", "URL")
	fmt.Printf("%-20s %s\n", "---", "---")
	for _, app := range apps {
		url := fmt.Sprintf("http://%s.%s", app, tld)
		fmt.Printf("%-20s %s\n", app, url)
	}
	fmt.Println("\nStart the server with: roost-dev serve")

	return nil
}

func runCleanup(tld string) error {
	fmt.Println("Uninstalling roost-dev configuration...")

	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstall requires root privileges. Run with: sudo roost-dev uninstall")
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
