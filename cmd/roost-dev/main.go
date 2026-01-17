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

	"golang.org/x/term"

	"github.com/panozzaj/roost-dev/internal/certs"
	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/dns"
	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/server"
	"github.com/panozzaj/roost-dev/internal/setup"
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

// confirmStep prompts the user for y/N confirmation and returns true if they confirm
func confirmStep(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// setupChecker is used to check installation status
var setupChecker = setup.NewChecker()

// isPortForwardingInstalled checks if port forwarding appears to be set up
func isPortForwardingInstalled(tld string) bool {
	return setupChecker.IsPortForwardingInstalled(tld)
}

// isCertInstalled checks if certificates appear to be set up
func isCertInstalled(configDir string) bool {
	return setupChecker.IsCertInstalled(configDir)
}

// isServiceInstalled checks if the background service appears to be set up
// Returns (installed, running)
func isServiceInstalled() (bool, bool) {
	homeDir, _ := os.UserHomeDir()
	installed := setupChecker.IsServiceInstalled(homeDir)
	if !installed {
		return false, false
	}
	// Check if it's running (requires exec, not abstracted)
	cmd := exec.Command("launchctl", "list", "com.roost-dev")
	if err := cmd.Run(); err != nil {
		return true, false
	}
	return true, true
}

// isPfPlistOutdated checks if the pf LaunchDaemon plist differs from expected.
func isPfPlistOutdated() bool {
	content, err := os.ReadFile(launchdPlistPath)
	if err != nil {
		return false // File doesn't exist or can't be read
	}
	return string(content) != expectedPfPlistContent
}

func main() {
	// Handle no args or help
	if len(os.Args) < 2 {
		printMainUsage()
		os.Exit(0)
	}

	// Handle global flags before command
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		// Check if asking for help on a specific command: roost-dev help serve
		if len(os.Args) >= 3 {
			os.Args = []string{os.Args[0], os.Args[2], "--help"}
		} else {
			printMainUsage()
			os.Exit(0)
		}
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
	case "setup":
		cmdSetup(args)
	case "teardown":
		cmdTeardown(args)
	case "status":
		cmdStatus(args)
	case "ports":
		cmdPorts(args)
	case "install":
		// Legacy: redirect to ports install
		cmdPortsInstall(args)
	case "uninstall":
		// Legacy: redirect to ports uninstall
		cmdPortsUninstall(args)
	case "service":
		cmdService(args)
	case "cert":
		cmdCert(args)
	case "docs":
		cmdDocs(args)
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

GETTING STARTED:
    setup             Interactive setup wizard (ports + cert + service)
    teardown          Remove all roost-dev configuration
    status            Show status of all components

APP MANAGEMENT:
    serve             Start the roost-dev server
    list, ls          List configured apps and their status
    start <app>       Start an app
    stop <app>        Stop an app
    restart <app>     Restart an app

COMPONENT MANAGEMENT:
    ports             Manage port forwarding (install/uninstall)
    cert              Manage HTTPS certificates (install/uninstall)
    service           Manage background service (install/uninstall)

HELP:
    docs              Full documentation (config, troubleshooting)
    <command> --help  Command-specific options

QUICK START:
    roost-dev setup               # Interactive setup wizard
    # Then visit http://roost-dev.test`)
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
		globalCfg = &GlobalConfig{TLD: "test"}
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

// cmdPortsInstall handles the 'ports install' command (also legacy 'install')
func cmdPortsInstall(args []string) {
	fs := flag.NewFlagSet("ports install", flag.ExitOnError)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	var (
		tld       string
		configDir string
	)

	fs.StringVar(&tld, "tld", "test", "Top-level domain to configure")
	fs.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`roost-dev ports install - Setup port forwarding

USAGE:
    roost-dev ports install [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Sets up macOS pf (packet filter) rules to forward ports to roost-dev,
    and creates a DNS resolver for the TLD. This allows accessing apps at
    http://myapp.test without specifying a port.

    Port forwarding:
      - Port 80  → 9280 (HTTP)
      - Port 443 → 9443 (HTTPS)

    Requires sudo for system configuration.

EXAMPLES:
    roost-dev ports install              # Setup for .test (default)
    roost-dev ports install --tld dev    # Setup for .dev TLD`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	if err := runPortsInstall(configDir, tld); err != nil {
		log.Fatalf("Port forwarding install failed: %v", err)
	}
}

// cmdPortsUninstall handles the 'ports uninstall' command (also legacy 'uninstall')
func cmdPortsUninstall(args []string) {
	fs := flag.NewFlagSet("ports uninstall", flag.ExitOnError)

	var tld string
	fs.StringVar(&tld, "tld", "test", "Top-level domain to remove")

	fs.Usage = func() {
		fmt.Println(`roost-dev ports uninstall - Remove port forwarding config

USAGE:
    roost-dev ports uninstall [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Removes the pf anchor file and DNS resolver (if using custom TLD).
    Does not modify /etc/pf.conf - you may want to manually remove
    the roost-dev lines or restore from the backup.

    Requires sudo for system configuration.

EXAMPLES:
    roost-dev ports uninstall              # Remove .test config (default)
    roost-dev ports uninstall --tld dev    # Remove .dev TLD config`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	if err := runPortsUninstall(tld); err != nil {
		log.Fatalf("Port forwarding uninstall failed: %v", err)
	}
}

// cmdPorts handles the 'ports' command for managing port forwarding
func cmdPorts(args []string) {
	if len(args) == 0 {
		printPortsUsage()
		os.Exit(0)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "install":
		cmdPortsInstall(subargs)
	case "uninstall":
		cmdPortsUninstall(subargs)
	case "-h", "--help", "help":
		printPortsUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown ports command: %s\n\n", subcmd)
		printPortsUsage()
		os.Exit(1)
	}
}

func printPortsUsage() {
	fmt.Println(`roost-dev ports - Manage port forwarding

USAGE:
    roost-dev ports <command>

COMMANDS:
    install     Setup port forwarding (80→9280, 443→9443)
    uninstall   Remove port forwarding configuration

Use 'roost-dev status' to check port forwarding status.

DESCRIPTION:
    Manages macOS pf (packet filter) rules that forward ports 80 and 443
    to roost-dev, allowing you to access apps at http://myapp.test without
    specifying a port number.`)
}

// cmdSetup is the interactive setup wizard
func cmdSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	var (
		tld       string
		configDir string
	)

	fs.StringVar(&tld, "tld", "test", "Top-level domain to configure")
	fs.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`roost-dev setup - Interactive setup wizard

USAGE:
    roost-dev setup [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Sets up roost-dev with all recommended components:

    1. Port forwarding - Forward ports 80/443 to roost-dev
       Lets you use http://myapp.test instead of http://localhost:9280

    2. HTTPS certificates - Generate a trusted local CA
       Enables https://myapp.test with no browser warnings

    3. Background service - Start roost-dev automatically on login
       roost-dev runs in the background so your apps are always ready

    The wizard explains each step before asking for your password.
    You can also run each step individually with the ports/cert/service commands.`)
	}

	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	runSetupWizard(configDir, tld)
}

// cmdTeardown removes all roost-dev configuration
func cmdTeardown(args []string) {
	fs := flag.NewFlagSet("teardown", flag.ExitOnError)

	var tld string
	fs.StringVar(&tld, "tld", "test", "Top-level domain to remove")

	fs.Usage = func() {
		fmt.Println(`roost-dev teardown - Remove all roost-dev configuration

USAGE:
    roost-dev teardown [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Removes all roost-dev components:
    - Stops and removes the background service
    - Removes HTTPS certificates and CA from trust store
    - Removes port forwarding rules

    The wizard explains each step and asks for confirmation.`)
	}

	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	runTeardownWizard(tld)
}

// cmdStatus shows overall status of all components
func cmdStatus(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev status - Show status of all components

USAGE:
    roost-dev status

Shows the status of:
- Port forwarding (ports 80/443)
- HTTPS certificates
- Background service`)
			os.Exit(0)
		}
	}

	runOverallStatus()
}

// cmdService handles the 'service' command for managing roost-dev as a background service
func cmdService(args []string) {
	if len(args) == 0 {
		printServiceUsage()
		os.Exit(0)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "install":
		cmdServiceInstall(subargs)
	case "uninstall":
		cmdServiceUninstall(subargs)
	case "status":
		cmdServiceStatus(subargs)
	case "-h", "--help", "help":
		printServiceUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown service command: %s\n\n", subcmd)
		printServiceUsage()
		os.Exit(1)
	}
}

func printServiceUsage() {
	fmt.Println(`roost-dev service - Manage roost-dev as a background service

USAGE:
    roost-dev service <command>

COMMANDS:
    install     Install and start roost-dev as a LaunchAgent (runs on login)
    uninstall   Stop and remove the LaunchAgent
    status      Show service status

DESCRIPTION:
    Sets up roost-dev to run automatically in the background via macOS LaunchAgent.
    The service will start on login and restart automatically if it crashes.

EXAMPLES:
    roost-dev service install     # Start running in background
    roost-dev service status      # Check if it's running
    roost-dev service uninstall   # Stop background service`)
}

func cmdServiceInstall(args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev service install - Install roost-dev as a background service

USAGE:
    roost-dev service install

Installs a LaunchAgent that runs 'roost-dev serve' automatically on login.
Logs are written to ~/Library/Logs/roost-dev/`)
			os.Exit(0)
		}
	}

	if err := runServiceInstall(); err != nil {
		log.Fatalf("Service install failed: %v", err)
	}
}

func cmdServiceUninstall(args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev service uninstall - Remove roost-dev background service

USAGE:
    roost-dev service uninstall

Stops roost-dev and removes the LaunchAgent.`)
			os.Exit(0)
		}
	}

	if err := runServiceUninstall(); err != nil {
		log.Fatalf("Service uninstall failed: %v", err)
	}
}

func cmdServiceStatus(args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev service status - Show service status

USAGE:
    roost-dev service status

Shows whether the LaunchAgent is installed and running.`)
			os.Exit(0)
		}
	}

	runServiceStatus()
}

func getUserLaunchAgentPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "LaunchAgents", "com.roost-dev.plist")
}

func runServiceInstall() error {
	// LaunchAgents must be installed as the user, not root
	// Even with SUDO_UID, root can't bootstrap into another user's GUI domain
	if os.Geteuid() == 0 {
		return fmt.Errorf("cannot install user LaunchAgent as root; run 'roost-dev service install' without sudo")
	}

	homeDir, _ := os.UserHomeDir()
	plistPath := getUserLaunchAgentPath()
	logsDir := filepath.Join(homeDir, "Library", "Logs", "roost-dev")

	// Find roost-dev binary
	binaryPath, err := exec.LookPath("roost-dev")
	if err != nil {
		// Fall back to go/bin
		binaryPath = filepath.Join(homeDir, "go", "bin", "roost-dev")
	}

	// Ensure logs directory exists
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("creating logs directory: %w", err)
	}

	// Create LaunchAgent directory if needed
	launchAgentDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentDir, 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}

	// Build environment variables section
	// Capture key environment variables from current session so spawned processes
	// have access to user's PATH (with nvm, rbenv, etc.), HOME, and other essentials
	envVars := []struct{ key, fallback string }{
		{"HOME", os.Getenv("HOME")},
		{"USER", os.Getenv("USER")},
		{"PATH", os.Getenv("PATH")},
		{"SHELL", "/bin/zsh"},
		{"LANG", "en_US.UTF-8"},
	}

	var envSection strings.Builder
	envSection.WriteString("    <key>EnvironmentVariables</key>\n")
	envSection.WriteString("    <dict>\n")
	for _, ev := range envVars {
		val := ev.fallback
		if val == "" {
			continue
		}
		// Escape XML special characters
		val = strings.ReplaceAll(val, "&", "&amp;")
		val = strings.ReplaceAll(val, "<", "&lt;")
		val = strings.ReplaceAll(val, ">", "&gt;")
		envSection.WriteString(fmt.Sprintf("        <key>%s</key>\n", ev.key))
		envSection.WriteString(fmt.Sprintf("        <string>%s</string>\n", val))
	}
	envSection.WriteString("    </dict>\n")

	// Generate plist content
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.roost-dev</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>serve</string>
    </array>
%s    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>%s/stderr.log</string>
</dict>
</plist>
`, binaryPath, envSection.String(), logsDir, logsDir)

	fmt.Printf("Creating %s...\n", plistPath)
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	// Unload if already loaded (ignore errors)
	exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.roost-dev", os.Getuid())).Run()

	// Load the agent
	fmt.Println("Loading LaunchAgent...")
	cmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("loading LaunchAgent: %s (%w)", string(output), err)
	}

	fmt.Println()
	fmt.Println("Service installed successfully!")
	fmt.Println()
	fmt.Printf("roost-dev is now running in the background.\n")
	fmt.Printf("Logs: %s/\n", logsDir)
	fmt.Println()
	fmt.Println("The service will start automatically on login.")

	return nil
}

func runServiceUninstall() error {
	green := "\033[32m"
	reset := "\033[0m"

	plistPath := getUserLaunchAgentPath()

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Printf("%s✓ Service is not installed%s\n", green, reset)
		return nil
	}

	// Unload the agent
	exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.roost-dev", os.Getuid())).Run()

	// Remove plist
	fmt.Printf("  Removing %s\n", plistPath)
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}

	return nil
}

func runServiceStatus() {
	plistPath := getUserLaunchAgentPath()

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("Service: not installed")
		fmt.Println()
		fmt.Println("Run 'roost-dev service install' to set up background service.")
		return
	}

	fmt.Println("Service: installed")
	fmt.Printf("Plist: %s\n", plistPath)

	// Check if running via launchctl
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/com.roost-dev", os.Getuid()))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Status: not running")
	} else {
		// Parse PID from output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "pid = ") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					pid := strings.TrimSpace(parts[1])
					fmt.Printf("Status: running (PID %s)\n", pid)
					break
				}
			}
		}
	}

	homeDir, _ := os.UserHomeDir()
	logsDir := filepath.Join(homeDir, "Library", "Logs", "roost-dev")
	fmt.Printf("Logs: %s/\n", logsDir)
}

// cmdCert handles the 'cert' command for managing HTTPS certificates
func cmdCert(args []string) {
	if len(args) == 0 {
		printCertUsage()
		os.Exit(0)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "install":
		cmdCertInstall(subargs)
	case "uninstall":
		cmdCertUninstall(subargs)
	case "status":
		cmdCertStatus(subargs)
	case "-h", "--help", "help":
		printCertUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown cert command: %s\n\n", subcmd)
		printCertUsage()
		os.Exit(1)
	}
}

func printCertUsage() {
	fmt.Println(`roost-dev cert - Manage HTTPS certificates

USAGE:
    roost-dev cert <command>

COMMANDS:
    install     Generate CA and trust it (requires sudo for trust)
    uninstall   Remove CA and certificates
    status      Show certificate status

DESCRIPTION:
    Generates a local Certificate Authority (CA) that roost-dev uses to
    dynamically create certificates for any domain. After running 'cert install':

    - https://myapp.test will work (any domain!)
    - Certificates are generated on-demand
    - No browser warnings

EXAMPLES:
    roost-dev cert install    # Generate CA and enable HTTPS
    roost-dev cert status     # Check certificate status`)
}

func cmdCertInstall(args []string) {
	fs := flag.NewFlagSet("cert install", flag.ExitOnError)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := filepath.Join(homeDir, ".config", "roost-dev")

	var tld string
	var configDir string

	fs.StringVar(&tld, "tld", "test", "TLD for certificate (e.g., test)")
	fs.StringVar(&configDir, "dir", defaultConfigDir, "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`roost-dev cert install - Generate and trust the roost-dev CA

USAGE:
    roost-dev cert install [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Generates a local Certificate Authority (CA) and installs it into
    your system trust store. roost-dev then uses this CA to dynamically
    generate certificates for any domain on-the-fly.

    After running this:
    - https://myapp.test will work in browsers
    - https://anyother.test will also work
    - No certificate warnings
    - No need to regenerate certs when adding new apps`)
	}

	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	// Load saved config for TLD
	globalCfg, _ := loadGlobalConfig(configDir)
	if globalCfg != nil && tld == "test" {
		tld = globalCfg.TLD
	}

	if err := runCertInstall(configDir, tld); err != nil {
		log.Fatalf("Certificate install failed: %v", err)
	}
}

func cmdCertUninstall(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev cert uninstall - Remove HTTPS certificates

USAGE:
    roost-dev cert uninstall

Removes the roost-dev CA and certificates from the config directory.
Also removes the CA from the system trust store (requires sudo).`)
			os.Exit(0)
		}
	}

	if err := runCertUninstall(); err != nil {
		log.Fatalf("Certificate uninstall failed: %v", err)
	}
}

func cmdCertStatus(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev cert status - Show certificate status

USAGE:
    roost-dev cert status

Shows whether certificates are installed and their details.`)
			os.Exit(0)
		}
	}

	runCertStatus()
}

func getCertsDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "roost-dev", "certs")
}

func runCertInstall(configDir, tld string) error {
	certsDir := getCertsDir()

	// Check if CA already exists
	caPath := filepath.Join(certsDir, "ca.pem")
	if _, err := os.Stat(caPath); err == nil {
		fmt.Printf("CA already exists: %s\n", caPath)
		fmt.Println("To regenerate, run 'roost-dev cert uninstall' first.")
		return nil
	}

	// Generate CA
	fmt.Println("Generating roost-dev CA...")
	if err := certs.GenerateCA(certsDir); err != nil {
		return fmt.Errorf("generating CA: %w", err)
	}

	fmt.Println()
	fmt.Printf("CA certificate: %s\n", caPath)
	fmt.Println()

	// Install CA into system trust store (macOS)
	fmt.Println("Installing CA into system trust store...")
	fmt.Println("(You may be prompted for your password)")
	fmt.Println()

	cmd := exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", caPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println()
		fmt.Println("To install manually:")
		fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caPath)
		fmt.Println()
		fmt.Println("Or double-click the CA file and trust it in Keychain Access.")
		return fmt.Errorf("failed to install CA into system trust store")
	}

	fmt.Println()
	fmt.Println("CA installed successfully!")
	fmt.Println()
	fmt.Println("HTTPS is now enabled with dynamic certificate generation.")
	fmt.Println("Any *.test domain will automatically get a valid certificate.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Restart roost-dev:")
	fmt.Println("     roost-dev service uninstall && roost-dev service install")
	fmt.Println()
	fmt.Println("  2. Restart your browser (quit fully and reopen)")
	fmt.Println("     This is needed for browsers to trust the new CA.")
	fmt.Println()
	fmt.Printf("Then visit: https://roost-dev.%s\n", tld)

	return nil
}

func runCertUninstall() error {
	green := "\033[32m"
	reset := "\033[0m"

	certsDir := getCertsDir()

	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		fmt.Printf("%s✓ No certificates installed%s\n", green, reset)
		return nil
	}

	fmt.Printf("  Removing %s\n", certsDir)
	if err := os.RemoveAll(certsDir); err != nil {
		return fmt.Errorf("removing certs directory: %w", err)
	}

	return nil
}

func runCertStatus() {
	certsDir := getCertsDir()
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")

	// Load TLD from config
	globalCfg, _ := loadGlobalConfig(configDir)
	tld := "test"
	if globalCfg != nil {
		tld = globalCfg.TLD
	}

	caFile := filepath.Join(certsDir, "ca.pem")
	keyFile := filepath.Join(certsDir, "ca-key.pem")

	fmt.Printf("TLD: .%s\n", tld)
	fmt.Printf("Certs directory: %s\n", certsDir)
	fmt.Println()

	caExists := false

	if info, err := os.Stat(caFile); err == nil {
		caExists = true
		fmt.Printf("CA Certificate: %s\n", caFile)
		fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("CA Certificate: not found")
	}

	if _, err := os.Stat(keyFile); err == nil {
		fmt.Printf("CA Key: %s\n", keyFile)
	} else {
		fmt.Println("CA Key: not found")
	}

	fmt.Println()
	if caExists {
		fmt.Println("Status: HTTPS enabled (dynamic certificate generation)")
		fmt.Printf("  https://myapp.%s will work\n", tld)
		fmt.Printf("  https://anyapp.%s will work\n", tld)
		fmt.Println()
		fmt.Println("Certificates are generated on-demand for each domain.")
	} else {
		fmt.Println("Status: HTTPS not configured")
		fmt.Println("  Run 'roost-dev cert install' to enable HTTPS")
	}
}

const (
	pfAnchorPath     = "/etc/pf.anchors/roost-dev"
	launchdPlistPath = "/Library/LaunchDaemons/dev.roost.pfctl.plist"
	globalConfigName = "config.json"

	// expectedPfPlistContent is the expected content of the pf LaunchDaemon plist.
	// Used by both isPfPlistOutdated() and runPortsInstall() to stay in sync.
	expectedPfPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
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
)

func loadGlobalConfig(configDir string) (*GlobalConfig, error) {
	path := filepath.Join(configDir, globalConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{TLD: "test"}, nil
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

// getProcessOnPort returns the process name listening on a port, or empty string if unknown
func getProcessOnPort(port int) string {
	// Try lsof to find the process (works without sudo for processes we own)
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse lsof output - format: COMMAND PID USER ...
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] { // Skip header
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			return fields[0] // Return command name
		}
	}
	return ""
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
		if proc := getProcessOnPort(80); proc != "" {
			warnings = append(warnings, fmt.Sprintf("%s is listening on port 80", proc))
		} else {
			warnings = append(warnings, "something is listening on port 80")
		}
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

func runPortsInstall(configDir, tld string) error {
	// If not running as root, check for conflicts and re-invoke with sudo
	if os.Geteuid() != 0 {
		// Check for conflicts before asking for sudo
		if err := checkInstallConflicts(tld); err != nil {
			return err
		}

		fmt.Println("Port forwarding requires administrator privileges.")
		fmt.Println("You'll be prompted for your password.")
		fmt.Println()

		// Find our binary
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding executable: %w", err)
		}

		// Re-invoke with sudo
		args := []string{exe, "ports", "install", "--tld", tld, "--dir", configDir}
		cmd := exec.Command("sudo", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	// Running as root - do the actual install
	fmt.Println("Installing port forwarding...")

	// Save TLD to config so we don't need --tld flag every time
	if err := saveGlobalConfig(configDir, &GlobalConfig{TLD: tld}); err != nil {
		fmt.Printf("Warning: could not save config: %v\n", err)
	}

	// Create the pf anchor file
	anchorContent := `# roost-dev port forwarding rules
# Forward port 80 to 9280 for roost-dev HTTP (IPv4)
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 9280
# Forward port 443 to 9443 for roost-dev HTTPS (IPv4)
rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port 9443
# Forward port 80 to 9280 for roost-dev HTTP (IPv6)
rdr pass on lo0 inet6 proto tcp from any to any port 80 -> ::1 port 9280
# Forward port 443 to 9443 for roost-dev HTTPS (IPv6)
rdr pass on lo0 inet6 proto tcp from any to any port 443 -> ::1 port 9443
`
	// Create pf anchor file (no message - reduce verbosity)
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

	// Create launchd plist (see expectedPfPlistContent for architecture notes)
	if err := os.WriteFile(launchdPlistPath, []byte(expectedPfPlistContent), 0644); err != nil {
		return fmt.Errorf("writing launchd plist: %w", err)
	}

	// Load the LaunchDaemon so pf rules persist across reboots
	// First unload if already loaded (ignore errors)
	exec.Command("/bin/launchctl", "bootout", "system/dev.roost.pfctl").Run()
	// Then bootstrap the daemon
	if err := exec.Command("/bin/launchctl", "bootstrap", "system", launchdPlistPath).Run(); err != nil {
		return fmt.Errorf("loading LaunchDaemon (are you running as root?): %w", err)
	}

	// Enable pf and load the rules now (suppress verbose pfctl output)
	cmd := exec.Command("/sbin/pfctl", "-e", "-f", "/etc/pf.conf")
	cmd.Run() // Ignore errors - pf may already be enabled

	// Load the anchor specifically
	cmd = exec.Command("/sbin/pfctl", "-a", "roost-dev", "-f", pfAnchorPath)
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
		resolverContent := "# Generated by roost-dev\nnameserver 127.0.0.1\nport 9053\n"
		// Create resolver file (no message - reduce verbosity)
		if err := os.WriteFile(resolverPath, []byte(resolverContent), 0644); err != nil {
			return fmt.Errorf("writing resolver file: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("Port forwarding installed!")
	fmt.Println("  - Port 80  → 9280 (HTTP)")
	fmt.Println("  - Port 443 → 9443 (HTTPS)")
	fmt.Printf("  - TLD: .%s\n", tld)
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
		globalCfg = &GlobalConfig{TLD: "test"}
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
		globalCfg = &GlobalConfig{TLD: "test"}
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

func runPortsUninstall(tld string) error {
	green := "\033[32m"
	reset := "\033[0m"

	// Check if already uninstalled (before prompting for sudo)
	if !isPortForwardingInstalled(tld) {
		fmt.Printf("%s✓ Port forwarding is not installed%s\n", green, reset)
		fmt.Println("  Not found: /etc/pf.anchors/roost-dev")
		fmt.Println("  Not found: /Library/LaunchDaemons/dev.roost.pfctl.plist")
		fmt.Printf("  Not found: /etc/resolver/%s\n", tld)
		return nil
	}

	// If not running as root, re-invoke with sudo
	if os.Geteuid() != 0 {
		fmt.Println("Removing port forwarding requires administrator privileges.")
		fmt.Println("You'll be prompted for your password.")
		fmt.Println()

		// Find our binary
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding executable: %w", err)
		}

		// Re-invoke with sudo
		args := []string{exe, "ports", "uninstall", "--tld", tld}
		cmd := exec.Command("sudo", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	fmt.Println("Removing port forwarding...")
	removedSomething := false

	// Flush our anchor (suppress output)
	exec.Command("/sbin/pfctl", "-a", "roost-dev", "-F", "all").Run()

	// Remove the anchor file
	if _, err := os.Stat(pfAnchorPath); err == nil {
		fmt.Printf("  Removing %s\n", pfAnchorPath)
		os.Remove(pfAnchorPath)
		removedSomething = true
	}

	// Remove the launchd plist
	if _, err := os.Stat(launchdPlistPath); err == nil {
		fmt.Printf("  Removing %s\n", launchdPlistPath)
		exec.Command("/bin/launchctl", "bootout", "system/dev.roost.pfctl").Run()
		os.Remove(launchdPlistPath)
		removedSomething = true
	}

	// Remove DNS resolver file for custom TLD
	if tld != "localhost" {
		resolverPath := fmt.Sprintf("/etc/resolver/%s", tld)
		if _, err := os.Stat(resolverPath); err == nil {
			fmt.Printf("  Removing %s\n", resolverPath)
			os.Remove(resolverPath)
			removedSomething = true
		}
	}

	if removedSomething {
		fmt.Printf("%s✓ Port forwarding removed%s\n", green, reset)
	}

	// Only mention backup if it exists
	backupPath := "/etc/pf.conf.roost-dev-backup"
	if _, err := os.Stat(backupPath); err == nil {
		fmt.Println()
		fmt.Println("Note: /etc/pf.conf still contains roost-dev references.")
		fmt.Println("To restore original pf.conf:")
		fmt.Printf("  sudo cp %s /etc/pf.conf\n", backupPath)
	}

	return nil
}

// runSetupWizard is the interactive setup wizard
func runSetupWizard(configDir, tld string) {
	red := "\033[31m"
	yellow := "\033[33m"
	green := "\033[32m"
	reset := "\033[0m"

	printLogo()
	fmt.Println()
	fmt.Println("Welcome to roost-dev setup!")
	fmt.Println()
	fmt.Println("This wizard will configure roost-dev with three components:")
	fmt.Println()
	fmt.Println("  1. PORT FORWARDING")
	fmt.Println("     Redirects ports 80 and 443 to roost-dev, so you can access")
	fmt.Println("     your apps at http://myapp.test instead of http://localhost:9280")
	fmt.Println("     Requires: sudo")
	fmt.Println("       - Writes to /etc/pf.anchors/roost-dev (firewall rules)")
	fmt.Printf("       - Writes to /etc/resolver/%s (DNS resolution)\n", tld)
	fmt.Println()
	fmt.Println("  2. HTTPS CERTIFICATES")
	fmt.Println("     Generates a local Certificate Authority so https://myapp.test")
	fmt.Println("     works in your browser with no warnings.")
	fmt.Println("     Requires: sudo")
	fmt.Println("       - Adds CA to /Library/Keychains/System.keychain")
	fmt.Println()
	fmt.Println("  3. BACKGROUND SERVICE")
	fmt.Println("     Installs a LaunchAgent so roost-dev starts automatically")
	fmt.Println("     when you log in. Your apps are always ready!")
	fmt.Println("     Requires: nothing (runs as your user)")
	fmt.Println("       - Writes to ~/Library/LaunchAgents/com.roost-dev.plist")
	fmt.Println()
	fmt.Println("Each step will ask for confirmation before making changes.")
	fmt.Println("Steps that need sudo will prompt for your password.")

	// Check for root AFTER showing overview so user understands the context
	if os.Geteuid() == 0 {
		fmt.Println()
		fmt.Printf("%sError: Do not run 'roost-dev setup' with sudo.%s\n", red, reset)
		fmt.Println()
		fmt.Println("Run it as your normal user instead:")
		fmt.Println("  roost-dev setup")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 1: Port forwarding
	fmt.Println()
	fmt.Println("Step 1/3: Port Forwarding")
	fmt.Println()
	if isPortForwardingInstalled(tld) {
		if isPfPlistOutdated() {
			fmt.Printf("%s⚠ Installed but config differs%s\n", yellow, reset)
			fmt.Println("  Found: /etc/pf.anchors/roost-dev")
			fmt.Println("  Found: /Library/LaunchDaemons/dev.roost.pfctl.plist (differs)")
			fmt.Printf("  Found: /etc/resolver/%s\n", tld)
			fmt.Println()
			fmt.Println("Requires: sudo (will prompt for password)")
			fmt.Println()
			if confirmStep("Update port forwarding configuration?") {
				if err := runPortsInstall(configDir, tld); err != nil {
					fmt.Printf("\n%s⚠ Update failed: %v%s\n", yellow, err, reset)
				} else {
					fmt.Printf("%s✓ Port forwarding updated%s\n", green, reset)
				}
			} else {
				fmt.Println("Skipped. You can update later with: roost-dev ports install")
			}
		} else {
			fmt.Printf("%s✓ Already installed%s\n", green, reset)
			fmt.Println("  Found: /etc/pf.anchors/roost-dev")
			fmt.Println("  Found: /Library/LaunchDaemons/dev.roost.pfctl.plist")
			fmt.Printf("  Found: /etc/resolver/%s\n", tld)
		}
	} else {
		fmt.Println("This step lets you access apps at http://myapp.test instead of")
		fmt.Println("http://localhost:9280. It configures macOS packet filter (pf) to")
		fmt.Println("redirect ports 80/443 to roost-dev.")
		fmt.Println()
		fmt.Println("What it does:")
		fmt.Println("  - Creates /etc/pf.anchors/roost-dev (firewall redirect rules)")
		fmt.Println("  - Adds anchor to /etc/pf.conf (backs up original first)")
		fmt.Println("  - Creates /Library/LaunchDaemons/dev.roost.pfctl.plist")
		fmt.Println("    (loads pf rules on boot)")
		fmt.Printf("  - Creates /etc/resolver/%s (DNS for .%s domain)\n", tld, tld)
		fmt.Println()
		fmt.Println("Requires: sudo (will prompt for password)")
		fmt.Println()
		if confirmStep("Install port forwarding?") {
			if err := runPortsInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Port forwarding failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev ports install")
			} else {
				fmt.Printf("%s✓ Port forwarding installed%s\n", green, reset)
			}
		} else {
			fmt.Println("Skipped. You can run this later with: roost-dev ports install")
		}
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 2: Certificates
	fmt.Println()
	fmt.Println("Step 2/3: HTTPS Certificates")
	fmt.Println()
	if isCertInstalled(configDir) {
		fmt.Printf("%s✓ Already installed%s\n", green, reset)
		fmt.Printf("  Found: %s/certs/ca-key.pem\n", configDir)
		fmt.Printf("  Found: %s/certs/ca.pem\n", configDir)
	} else {
		fmt.Println("This step enables https://myapp.test with no browser warnings.")
		fmt.Println("It creates a local Certificate Authority (CA) and adds it to your")
		fmt.Println("system keychain as a trusted root.")
		fmt.Println()
		fmt.Println("What it does:")
		fmt.Printf("  - Generates CA in %s/certs/\n", configDir)
		fmt.Println("  - Adds CA to /Library/Keychains/System.keychain (trusted root)")
		fmt.Println()
		fmt.Println("Requires: sudo (will prompt for password)")
		fmt.Println()
		if confirmStep("Install HTTPS certificates?") {
			if err := runCertInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Certificate setup failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev cert install")
			} else {
				fmt.Printf("%s✓ HTTPS certificates installed%s\n", green, reset)
			}
		} else {
			fmt.Println("Skipped. You can run this later with: roost-dev cert install")
		}
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 3: Background service
	fmt.Println()
	fmt.Println("Step 3/3: Background Service")
	fmt.Println()
	installed, running := isServiceInstalled()
	if installed {
		fmt.Printf("%s✓ Already installed%s\n", green, reset)
		fmt.Println("  Found: ~/Library/LaunchAgents/com.roost-dev.plist")
		if running {
			fmt.Println("  Status: running")
		} else {
			fmt.Printf("  %sStatus: not running%s\n", yellow, reset)
		}
	} else {
		fmt.Println("This step makes roost-dev start automatically when you log in,")
		fmt.Println("so your apps are always accessible.")
		fmt.Println()
		fmt.Println("What it does:")
		fmt.Println("  - Creates ~/Library/LaunchAgents/com.roost-dev.plist")
		fmt.Println("  - Starts roost-dev immediately")
		fmt.Println()
		fmt.Println("Requires: nothing (no sudo needed)")
		fmt.Println()
		if confirmStep("Install background service?") {
			if err := runServiceInstall(); err != nil {
				fmt.Printf("\n%s⚠ Service install failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev service install")
			} else {
				fmt.Printf("%s✓ Background service installed%s\n", green, reset)
			}
		} else {
			fmt.Println("Skipped. You can run this later with: roost-dev service install")
		}
	}
	fmt.Println()

	// Final summary
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Printf("  Dashboard:  http://roost-dev.%s\n", tld)
	fmt.Printf("  Dashboard:  https://roost-dev.%s (HTTPS)\n", tld)
	fmt.Println()
	fmt.Println("Create app configs in ~/.config/roost-dev/")
	fmt.Println("Run 'roost-dev status' to check component status.")
	fmt.Println()
	fmt.Println("Note: Restart your browser for HTTPS to work (quit fully and reopen).")
}

// runTeardownWizard removes all roost-dev configuration
func runTeardownWizard(tld string) {
	yellow := "\033[33m"
	green := "\033[32m"
	reset := "\033[0m"

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")

	fmt.Println("roost-dev teardown")
	fmt.Println()
	fmt.Println("This will remove all roost-dev components from your system.")
	fmt.Println("Each step will ask for confirmation before making changes.")
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 1: Background service
	fmt.Println()
	fmt.Println("Step 1/3: Background Service")
	fmt.Println()
	installed, running := isServiceInstalled()
	if !installed {
		fmt.Printf("%s✓ Already removed%s\n", green, reset)
		fmt.Println("  Not found: ~/Library/LaunchAgents/com.roost-dev.plist")
	} else {
		fmt.Println("Will remove:")
		fmt.Println("  - ~/Library/LaunchAgents/com.roost-dev.plist")
		if running {
			fmt.Println("  - Stop running service")
		}
		fmt.Println()
		if confirmStep("Remove background service?") {
			if err := runServiceUninstall(); err != nil {
				fmt.Printf("%s⚠ Service removal failed: %v%s\n", yellow, err, reset)
			} else {
				fmt.Printf("%s✓ Background service removed%s\n", green, reset)
			}
		} else {
			fmt.Println("Skipped.")
		}
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 2: Certificates
	fmt.Println()
	fmt.Println("Step 2/3: HTTPS Certificates")
	fmt.Println()
	if !isCertInstalled(configDir) {
		fmt.Printf("%s✓ Already removed%s\n", green, reset)
		fmt.Printf("  Not found: %s/certs/\n", configDir)
	} else {
		fmt.Println("Will remove:")
		fmt.Printf("  - %s/certs/ (CA key and certificate)\n", configDir)
		fmt.Println()
		fmt.Println("Note: The CA in your system keychain must be removed manually")
		fmt.Println("      via Keychain Access (search for 'roost-dev Local CA')")
		fmt.Println()
		if confirmStep("Remove HTTPS certificates?") {
			if err := runCertUninstall(); err != nil {
				fmt.Printf("%s⚠ Certificate removal failed: %v%s\n", yellow, err, reset)
			} else {
				fmt.Printf("%s✓ HTTPS certificates removed%s\n", green, reset)
			}
		} else {
			fmt.Println("Skipped.")
		}
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 3: Port forwarding
	fmt.Println()
	fmt.Println("Step 3/3: Port Forwarding")
	fmt.Println()
	if !isPortForwardingInstalled(tld) {
		fmt.Printf("%s✓ Already removed%s\n", green, reset)
		fmt.Println("  Not found: /etc/pf.anchors/roost-dev")
		fmt.Println("  Not found: /Library/LaunchDaemons/dev.roost.pfctl.plist")
		fmt.Printf("  Not found: /etc/resolver/%s\n", tld)
	} else {
		fmt.Println("Will remove:")
		fmt.Println("  - /etc/pf.anchors/roost-dev")
		fmt.Println("  - /Library/LaunchDaemons/dev.roost.pfctl.plist")
		fmt.Printf("  - /etc/resolver/%s\n", tld)
		fmt.Println()
		fmt.Println("Requires: sudo (will prompt for password)")
		fmt.Println()
		if confirmStep("Remove port forwarding?") {
			if err := runPortsUninstall(tld); err != nil {
				fmt.Printf("%s⚠ Port forwarding removal failed: %v%s\n", yellow, err, reset)
			}
			// runPortsUninstall prints its own success message
		} else {
			fmt.Println("Skipped.")
		}
	}
	fmt.Println()

	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Teardown complete!")
	fmt.Println()
	fmt.Println("To reinstall, run: roost-dev setup")
}

// runOverallStatus shows status of all components
func runOverallStatus() {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")

	// Load TLD from config
	globalCfg, _ := loadGlobalConfig(configDir)
	tld := "test"
	if globalCfg != nil && globalCfg.TLD != "" {
		tld = globalCfg.TLD
	}

	fmt.Println()
	fmt.Println("roost-dev status")
	fmt.Println("────────────────────────────────────────────────")

	// Check ports
	portsStatus, portsDetail := checkPortsStatus()
	fmt.Printf("  Ports     %s  %s\n", portsStatus, portsDetail)

	// Check cert
	certStatus, certDetail := checkCertStatus()
	fmt.Printf("  Cert      %s  %s\n", certStatus, certDetail)

	// Check service
	serviceStatus, serviceDetail := checkServiceStatus()
	fmt.Printf("  Service   %s  %s\n", serviceStatus, serviceDetail)

	fmt.Println("────────────────────────────────────────────────")
	fmt.Printf("  Dashboard: http://roost-dev.%s\n", tld)
	fmt.Println()
}

// checkPortsStatus returns the status of port forwarding
func checkPortsStatus() (string, string) {
	// Check if anchor file exists
	if _, err := os.Stat(pfAnchorPath); os.IsNotExist(err) {
		return "✗", "not installed"
	}

	// Check if pf rules are loaded by checking if our anchor has rules
	cmd := exec.Command("/sbin/pfctl", "-a", "roost-dev", "-sr")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "✗", "installed but not active"
	}

	return "✓", "80→9280, 443→9443"
}

// checkCertStatus returns the status of HTTPS certificates
func checkCertStatus() (string, string) {
	certsDir := getCertsDir()
	caPath := filepath.Join(certsDir, "ca.pem")

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return "✗", "not installed"
	}

	// Check if CA is trusted (simplified check)
	cmd := exec.Command("security", "find-certificate", "-c", "roost-dev Local CA", "/Library/Keychains/System.keychain")
	if err := cmd.Run(); err != nil {
		return "⚠", "CA exists but may not be trusted"
	}

	return "✓", "CA trusted"
}

// cmdDocs shows documentation
func cmdDocs(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`roost-dev docs - Show documentation

USAGE:
    roost-dev docs

Shows configuration and troubleshooting documentation.
Output is paged if running in a terminal.`)
			os.Exit(0)
		}
	}

	// Try to find docs file in several locations
	var content []byte
	var err error

	// 1. Try relative to current directory (for development)
	content, err = os.ReadFile("docs/roost-dev.txt")
	if err != nil {
		// 2. Try relative to executable
		if exe, exeErr := os.Executable(); exeErr == nil {
			content, err = os.ReadFile(filepath.Join(filepath.Dir(exe), "docs", "roost-dev.txt"))
		}
	}
	if err != nil {
		// 3. Try in source location (for go run)
		homeDir, _ := os.UserHomeDir()
		content, err = os.ReadFile(filepath.Join(homeDir, "Documents", "dev", "roost-dev", "docs", "roost-dev.txt"))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not find docs/roost-dev.txt\n")
		os.Exit(1)
	}

	// If stdout is a terminal, use a pager
	if term.IsTerminal(int(os.Stdout.Fd())) {
		pager := getPager()
		if pager != "" {
			cmd := exec.Command(pager)
			cmd.Stdin = strings.NewReader(string(content))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				return
			}
			// Fall through to direct output if pager fails
		}
	}

	// Direct output (non-terminal or no pager)
	os.Stdout.Write(content)
}

// getPager returns the pager command to use
func getPager() string {
	if pager := os.Getenv("PAGER"); pager != "" {
		return pager
	}
	if _, err := exec.LookPath("less"); err == nil {
		return "less"
	}
	if _, err := exec.LookPath("more"); err == nil {
		return "more"
	}
	return ""
}

// checkServiceStatus returns the status of the background service
func checkServiceStatus() (string, string) {
	plistPath := getUserLaunchAgentPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return "✗", "not installed"
	}

	// Check if service is running using launchctl list (pipe format)
	// Output format: "PID\tStatus\tLabel" or "-\tStatus\tLabel" if not running
	cmd := exec.Command("/bin/launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return "⚠", "installed (status unknown)"
	}

	// Look for our service in the list
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "com.roost-dev") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				pid := parts[0]
				if pid != "-" {
					return "✓", fmt.Sprintf("running (PID %s)", pid)
				}
				return "⚠", "installed but not running"
			}
		}
	}

	return "⚠", "installed but not loaded"
}
