package main

import (
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
	"github.com/panozzaj/roost-dev/internal/diff"
	"github.com/panozzaj/roost-dev/internal/dns"
	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/server"
)

func printLogo() {
	if os.Getenv("CLAUDECODE") != "1" {
		fmt.Println(logo.CLI())
	}
}

// confirmStep prompts the user for y/N confirmation and returns true if they confirm.
// If ROOST_DEV_YES=1 is set, automatically returns true without prompting.
func confirmStep(prompt string) bool {
	if os.Getenv("ROOST_DEV_YES") == "1" {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// confirmWithPlan shows a summary of planned changes and prompts for confirmation.
// User can press '?' to see the full diff. Returns true if confirmed, false if cancelled.
// If ROOST_DEV_YES=1 is set, automatically returns true without prompting or printing.
// If there are no actual changes, returns true (nothing to do, but not cancelled).
func confirmWithPlan(plan *diff.Plan, prompt string) bool {
	if os.Getenv("ROOST_DEV_YES") == "1" {
		return true
	}

	summary := plan.Summary()
	if summary == nil {
		return true // No changes needed, proceed silently
	}

	summary.Print()

	for {
		fmt.Printf("%s [y/N, ? for diff]: ", prompt)
		var response string
		fmt.Scanln(&response)

		switch response {
		case "y", "Y":
			return true
		case "?":
			fmt.Println()
			plan.Preview()
		default:
			return false
		}
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
	case "logs":
		cmdLogs(args)
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
    logs [app]        View server or app logs (-f to follow)

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

// cmdAppControl handles start/stop/restart commands
func cmdAppControl(action string, args []string) {
	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Printf(`roost-dev %s - %s an app or subprocess

USAGE:
    roost-dev %s <name>

NAME FORMATS:
    myapp                 %s all services in the app
    myapp:worker          %s the 'worker' service (colon syntax)
    worker.myapp          %s the 'worker' service (dot syntax)
    worker                %s the service if name is unique across apps

Requires the roost-dev server to be running.
`, action, strings.Title(action), action, strings.Title(action), strings.Title(action), strings.Title(action), strings.Title(action))
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
		fmt.Println("Requires: sudo (will prompt for password)")

		// Show summary and confirm (? shows full diff)
		if confirmPortsInstall(tld, "Install port forwarding?") {
			// Set ROOST_DEV_YES to skip the second confirmation in runPortsInstall
			os.Setenv("ROOST_DEV_YES", "1")
			if err := runPortsInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Port forwarding failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev ports install")
			} else {
				fmt.Printf("%s✓ Port forwarding installed%s\n", green, reset)
			}
			os.Unsetenv("ROOST_DEV_YES")
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
		fmt.Println("Will also add CA to /Library/Keychains/System.keychain (trusted root)")
		fmt.Println()
		fmt.Println("Requires: sudo (will prompt for password)")

		// Show summary and confirm (? shows full diff)
		plan := certInstallPlan()
		if confirmWithPlan(plan, "Install HTTPS certificates?") {
			// Set ROOST_DEV_YES to skip the second confirmation in runCertInstall
			os.Setenv("ROOST_DEV_YES", "1")
			if err := runCertInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Certificate setup failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev cert install")
			} else {
				fmt.Printf("%s✓ HTTPS certificates installed%s\n", green, reset)
			}
			os.Unsetenv("ROOST_DEV_YES")
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
		fmt.Println("Will also start roost-dev immediately.")
		fmt.Println()
		fmt.Println("Requires: nothing (no sudo needed)")

		// Show summary and confirm (? shows full diff)
		plan := serviceInstallPlan()
		if confirmWithPlan(plan, "Install background service?") {
			// Set ROOST_DEV_YES to skip the second confirmation in runServiceInstall
			os.Setenv("ROOST_DEV_YES", "1")
			if err := runServiceInstall(); err != nil {
				fmt.Printf("\n%s⚠ Service install failed: %v%s\n", yellow, err, reset)
				fmt.Println("You can retry later with: roost-dev service install")
			} else {
				fmt.Printf("%s✓ Background service installed%s\n", green, reset)
			}
			os.Unsetenv("ROOST_DEV_YES")
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
		if running {
			fmt.Println("Will also stop running service.")
		}
		// Show summary and confirm (? shows full diff)
		plan := serviceUninstallPlan()
		if confirmWithPlan(plan, "Remove background service?") {
			// Unload the agent
			exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.roost-dev", os.Getuid())).Run()
			if err := plan.Execute(); err != nil {
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
		fmt.Println("Note: The CA in your system keychain must be removed manually")
		fmt.Println("      via Keychain Access (search for 'roost-dev Local CA')")
		// Show summary and confirm (? shows full diff)
		plan := certUninstallPlan()
		if confirmWithPlan(plan, "Remove HTTPS certificates?") {
			if err := plan.Execute(); err != nil {
				fmt.Printf("%s⚠ Certificate removal failed: %v%s\n", yellow, err, reset)
			} else {
				// Also try to remove the certs directory itself if empty
				os.Remove(getCertsDir())
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
		fmt.Println("Requires: sudo (will prompt for password)")
		// Show summary and confirm (? shows full diff)
		plan := portsUninstallPlan(tld)
		if confirmWithPlan(plan, "Remove port forwarding?") {
			// Set ROOST_DEV_YES to skip the second confirmation in runPortsUninstall
			os.Setenv("ROOST_DEV_YES", "1")
			if err := runPortsUninstall(tld); err != nil {
				fmt.Printf("%s⚠ Port forwarding removal failed: %v%s\n", yellow, err, reset)
			}
			os.Unsetenv("ROOST_DEV_YES")
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
	// Note: This requires root access, so it may fail even when forwarding works
	cmd := exec.Command("/sbin/pfctl", "-a", "roost-dev", "-sr")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return "✓", "80→9280, 443→9443"
	}

	// Fallback: try connecting to port 80 to see if forwarding is actually working
	// This works even without root access to read pf rules
	conn, err := net.DialTimeout("tcp", "127.0.0.1:80", 500*time.Millisecond)
	if err == nil {
		conn.Close()
		return "✓", "80→9280, 443→9443"
	}

	return "✗", "installed but not active"
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
