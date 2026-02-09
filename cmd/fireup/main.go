package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/panozzaj/fireup/internal/config"
	"github.com/panozzaj/fireup/internal/diff"
	"github.com/panozzaj/fireup/internal/dns"
	"github.com/panozzaj/fireup/internal/logo"
	"github.com/panozzaj/fireup/internal/server"
)

func printLogo() {
	if !isAgent() {
		fmt.Println(logo.CLI())
	}
}

// confirmStep prompts the user for y/N confirmation and returns true if they confirm.
// If FIREUP_YES=1 is set, automatically returns true without prompting.
func confirmStep(prompt string) bool {
	if os.Getenv("FIREUP_YES") == "1" {
		return true
	}
	if isAgent() {
		fmt.Fprintf(os.Stderr, "Error: interactive prompt not supported in agent mode.\nUse FIREUP_YES=1 to skip confirmations.\n")
		return false
	}
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// confirmWithPlan shows a summary of planned changes and prompts for confirmation.
// User can press '?' to see the full diff. Returns true if confirmed, false if cancelled.
// If FIREUP_YES=1 is set, automatically returns true without prompting or printing.
// If there are no actual changes, returns true (nothing to do, but not cancelled).
func confirmWithPlan(plan *diff.Plan, prompt string) bool {
	if os.Getenv("FIREUP_YES") == "1" {
		return true
	}
	if isAgent() {
		fmt.Fprintf(os.Stderr, "Error: interactive prompt not supported in agent mode.\nUse FIREUP_YES=1 to skip confirmations.\n")
		return false
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
		// Check if asking for help on a specific command: fireup help serve
		if len(os.Args) >= 3 {
			os.Args = []string{os.Args[0], os.Args[2], "--help"}
		} else {
			printMainUsage()
			os.Exit(0)
		}
	}
	if os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version" {
		fmt.Printf("fireup %s\n", version)
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
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\nRun 'fireup help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func printMainUsage() {
	printLogo()
	var sb strings.Builder
	sb.WriteString(`
fireup - Local development proxy for all your projects

USAGE:
    fireup <command> [options]

APP STATUS:
    status            Show configured apps and their status (--json for JSON)
    list, ls          Alias for status

APP CONTROL:
    start <app>       Start an app
    stop <app>        Stop an app
    restart <app>     Restart an app
    logs [app]        View server or app logs (-f to follow)

SETUP:
    setup             Interactive setup wizard (ports + cert + service)
    setup status      Show status of setup components
    teardown          Remove all fireup configuration

ADVANCED:
    serve             Start the fireup server (usually runs as service)
    ports             Manage port forwarding (install/uninstall)
    cert              Manage HTTPS certificates (install/uninstall)
    service           Manage background service (install/uninstall)

HELP:
    docs              Full documentation (config, troubleshooting)
    <command> --help  Command-specific options

QUICK START:
    fireup setup               # Interactive setup wizard
    # Then visit http://fireup.test`)

	printCurrentProjectSection(&sb)
	fmt.Print(sb.String())
}

// cmdServe handles the 'serve' command
func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	var (
		configDir     string
		httpPort      int
		httpsPort     int
		advertisePort int
		dnsPort       int
		tld           string
	)

	fs.StringVar(&configDir, "dir", getDefaultConfigDir(), "Configuration directory")
	fs.IntVar(&httpPort, "http-port", 9280, "HTTP port to listen on")
	fs.IntVar(&httpsPort, "https-port", 9443, "HTTPS port to listen on")
	fs.IntVar(&advertisePort, "advertise-port", 80, "Port to use in URLs (0 = same as http-port)")
	fs.IntVar(&dnsPort, "dns-port", 9053, "DNS server port")
	fs.StringVar(&tld, "tld", "", "Top-level domain (default: from config or 'localhost')")

	fs.Usage = func() {
		fmt.Println(`fireup serve - Start the fireup server

USAGE:
    fireup serve [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
CONFIGURATION:
    Place config files in ~/.config/fireup/

    Command (recommended):
        echo "npm run dev" > ~/.config/fireup/myapp
        # fireup assigns a dynamic PORT, avoiding conflicts

    Static site (symlink to directory):
        ln -s ~/projects/my-site ~/.config/fireup/mysite
        # Directory must contain index.html

    Fixed port proxy (not recommended):
        echo "3000" > ~/.config/fireup/myapp
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
		fmt.Println("\n[fireup] Received shutdown signal, stopping all processes...")
		srv.Shutdown()
		fmt.Println("[fireup] Shutdown complete")
		os.Exit(0)
	}()

	printLogo()
	fmt.Printf("fireup %s\n", version)
	fmt.Printf("Configuration directory: %s\n", configDir)
	fmt.Printf("Listening on http://127.0.0.1:%d\n", httpPort)
	if urlPort == 80 {
		fmt.Printf("Dashboard at http://fireup.%s\n", tld)
	} else {
		fmt.Printf("Dashboard at http://fireup.%s:%d\n", tld, urlPort)
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
			fmt.Println(colorYellow + "WARNING: URLs like http://myapp.localhost won't work yet.")
			fmt.Println("")
			fmt.Println("  fireup is running on port 9280, but your browser will")
			fmt.Println("  try port 80. Run this once to set up the redirect:")
			fmt.Println("")
			fmt.Println("    sudo fireup install")
			fmt.Println(colorReset)
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
			fmt.Printf(`fireup %s - %s an app or subprocess

USAGE:
    fireup %s <name>

NAME FORMATS:
    myapp                 %s all services in the app
    myapp:worker          %s the 'worker' service (colon syntax)
    worker.myapp          %s the 'worker' service (dot syntax)
    worker                %s the service if name is unique across apps

Requires the fireup server to be running.
`, action, strings.Title(action), action, strings.Title(action), strings.Title(action), strings.Title(action), strings.Title(action))
			os.Exit(0)
		}
	}

	if len(args) < 1 {
		if appName, found := resolveAppFromCwd(); found {
			fmt.Fprintf(os.Stderr, "(detected %s from current directory)\n", appName)
			args = []string{appName}
		} else {
			fmt.Fprintf(os.Stderr, "Usage: fireup %s <app-name>\n", action)
			os.Exit(1)
		}
	}

	if err := runCommand(action, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdSetup is the interactive setup wizard
func cmdSetup(args []string) {
	// Check for subcommands first
	if len(args) > 0 && args[0] == "status" {
		cmdSetupStatus(args[1:])
		return
	}

	fs := flag.NewFlagSet("setup", flag.ExitOnError)

	var (
		tld       string
		configDir string
	)

	fs.StringVar(&tld, "tld", "test", "Top-level domain to configure")
	fs.StringVar(&configDir, "dir", getDefaultConfigDir(), "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`fireup setup - Interactive setup wizard

USAGE:
    fireup setup [options]
    fireup setup status    Show status of setup components

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Sets up fireup with all recommended components:

    1. Port forwarding - Forward ports 80/443 to fireup
       Lets you use http://myapp.test instead of http://localhost:9280

    2. HTTPS certificates - Generate a trusted local CA
       Enables https://myapp.test with no browser warnings

    3. Background service - Start fireup automatically on login
       fireup runs in the background so your apps are always ready

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

// cmdSetupStatus shows status of setup components (ports, cert, service)
func cmdSetupStatus(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`fireup setup status - Show status of setup components

USAGE:
    fireup setup status

Shows the status of:
- Port forwarding (ports 80/443)
- HTTPS certificates
- Background service`)
			os.Exit(0)
		}
	}

	runSetupComponentStatus()
}

// runSetupComponentStatus shows status of all setup components
func runSetupComponentStatus() {
	globalCfg, _ := getConfigWithDefaults()
	tld := globalCfg.TLD

	fmt.Println()
	fmt.Println("fireup setup status")
	fmt.Println("────────────────────────────────────────────────")

	portsStatus, portsDetail := checkPortsStatus()
	fmt.Printf("  Ports     %s  %s\n", portsStatus, portsDetail)

	certStatus, certDetail := checkCertStatus()
	fmt.Printf("  Cert      %s  %s\n", certStatus, certDetail)

	serviceStatus, serviceDetail := checkServiceStatus()
	fmt.Printf("  Service   %s  %s\n", serviceStatus, serviceDetail)

	fmt.Println("────────────────────────────────────────────────")
	fmt.Printf("  Dashboard: http://fireup.%s\n", tld)
	fmt.Println()
}

// cmdTeardown removes all fireup configuration
func cmdTeardown(args []string) {
	fs := flag.NewFlagSet("teardown", flag.ExitOnError)

	var tld string
	fs.StringVar(&tld, "tld", "test", "Top-level domain to remove")

	fs.Usage = func() {
		fmt.Println(`fireup teardown - Remove all fireup configuration

USAGE:
    fireup teardown [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Removes all fireup components:
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

// cmdStatus shows app status (replaces the old 'list' command behavior)
func cmdStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output in JSON format")

	fs.Usage = func() {
		fmt.Println(`fireup status - Show status of configured apps

USAGE:
    fireup status [options] [filter]

ARGUMENTS:
    filter            Optional filter to match app or service names (substring match)

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
Shows all configured apps, their running status, and URLs.
Use --json for machine-readable output (same as /api/status).

EXAMPLES:
    fireup status              # Show all apps
    fireup status family       # Filter to apps/services matching "family"
    fireup status --json api   # JSON output filtered to "api"

For setup component status (ports, cert, service), use:
    fireup setup status`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	// Get optional filter argument
	filter := ""
	if fs.NArg() > 0 {
		filter = fs.Arg(0)
	}

	if err := runStatus(*jsonOutput, filter); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// filterApps returns apps that match the filter (case-insensitive substring match on app name, aliases, or service names)
func filterApps(apps []AppStatus, filter string) []AppStatus {
	filter = strings.ToLower(filter)
	var result []AppStatus

	for _, app := range apps {
		// Check if app name or aliases match
		appMatches := strings.Contains(strings.ToLower(app.Name), filter)
		if !appMatches {
			for _, alias := range app.Aliases {
				if strings.Contains(strings.ToLower(alias), filter) {
					appMatches = true
					break
				}
			}
		}

		if appMatches {
			result = append(result, app)
			continue
		}

		// Check if any services match (for multi-service apps)
		if len(app.Services) > 0 {
			var matchingServices []SvcStatus
			for _, svc := range app.Services {
				if strings.Contains(strings.ToLower(svc.Name), filter) {
					matchingServices = append(matchingServices, svc)
				}
			}
			if len(matchingServices) > 0 {
				// Include app but only with matching services
				filteredApp := app
				filteredApp.Services = matchingServices
				result = append(result, filteredApp)
			}
		}
	}

	return result
}

// findSimilarNames finds app/service names similar to the filter using Levenshtein distance
func findSimilarNames(apps []AppStatus, filter string) []string {
	filter = strings.ToLower(filter)
	type scored struct {
		name  string
		score int
	}
	var candidates []scored

	for _, app := range apps {
		// Score app name
		dist := levenshtein(strings.ToLower(app.Name), filter)
		if dist <= 3 || strings.Contains(strings.ToLower(app.Name), filter[:min(len(filter), 3)]) {
			candidates = append(candidates, scored{app.Name, dist})
		}

		// Score aliases
		for _, alias := range app.Aliases {
			dist := levenshtein(strings.ToLower(alias), filter)
			if dist <= 3 || strings.Contains(strings.ToLower(alias), filter[:min(len(filter), 3)]) {
				candidates = append(candidates, scored{alias, dist})
			}
		}

		// Score services
		for _, svc := range app.Services {
			dist := levenshtein(strings.ToLower(svc.Name), filter)
			if dist <= 3 || strings.Contains(strings.ToLower(svc.Name), filter[:min(len(filter), 3)]) {
				candidates = append(candidates, scored{fmt.Sprintf("%s (service in %s)", svc.Name, app.Name), dist})
			}
		}
	}

	// Sort by score and dedupe
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})

	seen := make(map[string]bool)
	var result []string
	for _, c := range candidates {
		if !seen[c.name] && len(result) < 5 {
			seen[c.name] = true
			result = append(result, c.name)
		}
	}

	return result
}

// levenshtein calculates the Levenshtein distance between two strings
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	if len(a) > len(b) {
		a, b = b, a
	}

	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)

	for i := range prev {
		prev[i] = i
	}

	for j := 1; j <= len(b); j++ {
		curr[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[i] = min(min(curr[i-1]+1, prev[i]+1), prev[i-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[len(a)]
}

// runStatus displays app status, optionally as JSON, with optional filter
func runStatus(jsonOutput bool, filter string) error {
	globalCfg, configDir := getConfigWithDefaults()

	// Try to get status from running server
	url := fmt.Sprintf("http://fireup.%s/api/status", globalCfg.TLD)
	resp, err := http.Get(url)
	if err != nil {
		if jsonOutput {
			// Return empty array for JSON when server not running
			fmt.Println("[]")
			return nil
		}
		// Server not running - fall back to listing config files
		return listConfigFiles(configDir, globalCfg.TLD)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
		return listConfigFiles(configDir, globalCfg.TLD)
	}

	// Parse status
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	var apps []AppStatus
	if err := json.Unmarshal(body, &apps); err != nil {
		return fmt.Errorf("failed to parse status: %v", err)
	}

	// Apply filter if provided
	if filter != "" {
		apps = filterApps(apps, filter)
		if len(apps) == 0 {
			// No matches - suggest similar names
			var allApps []AppStatus
			json.Unmarshal(body, &allApps)
			suggestions := findSimilarNames(allApps, filter)
			if jsonOutput {
				fmt.Println("[]")
			} else {
				fmt.Printf("No apps or services matching %q found.\n", filter)
				if len(suggestions) > 0 {
					fmt.Printf("\nDid you mean one of these?\n")
					for _, s := range suggestions {
						fmt.Printf("  %s\n", s)
					}
				}
			}
			return nil
		}
	}

	// For JSON output, marshal filtered results
	if jsonOutput {
		output, _ := json.Marshal(apps)
		fmt.Println(string(output))
		return nil
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
			runningCount := 0
			for _, svc := range app.Services {
				if svc.Running {
					runningCount++
				}
			}
			if runningCount == 0 {
				status = "idle"
			} else if runningCount == len(app.Services) {
				status = "running"
			} else {
				status = fmt.Sprintf("%d/%d", runningCount, len(app.Services))
			}
		} else {
			if app.Running {
				status = "running"
			} else {
				status = "idle"
			}
		}

		paddedStatus := fmt.Sprintf("%-10s", status)
		switch {
		case status == "running":
			paddedStatus = colorGreen + paddedStatus + colorReset
		case status == "idle":
			paddedStatus = colorGray + paddedStatus + colorReset
		case strings.Contains(status, "/"):
			paddedStatus = colorYellow + paddedStatus + colorReset
		}

		name := app.Name
		if len(app.Aliases) > 0 {
			name = fmt.Sprintf("%s (%s)", app.Name, strings.Join(app.Aliases, ", "))
		}
		fmt.Printf("%-25s %s %s\n", name, paddedStatus, app.URL)

		// Print services for multi-service apps
		if app.Type == "multi-service" && len(app.Services) > 0 {
			for i, svc := range app.Services {
				var prefix string
				if i == len(app.Services)-1 {
					prefix = "└─"
				} else {
					prefix = "├─"
				}

				var svcStatus string
				if svc.Running {
					svcStatus = "running"
				} else {
					svcStatus = "idle"
				}

				svcPaddedStatus := fmt.Sprintf("%-10s", svcStatus)
				if svcStatus == "running" {
					svcPaddedStatus = colorGreen + svcPaddedStatus + colorReset
				} else {
					svcPaddedStatus = colorGray + svcPaddedStatus + colorReset
				}

				svcName := fmt.Sprintf("%s %s", prefix, svc.Name)
				fmt.Printf("  %-23s %s %s\n", svcName, svcPaddedStatus, svc.URL)
			}
		}
	}

	return nil
}

func runCommand(cmd, appName string) error {
	// Load config to get TLD
	globalCfg, _ := getConfigWithDefaults()

	// Show action in progress
	switch cmd {
	case "start":
		fmt.Printf("Starting %s...\n", appName)
	case "stop":
		fmt.Printf("Stopping %s...\n", appName)
	case "restart":
		fmt.Printf("Restarting %s...\n", appName)
	}

	// Make request to fireup API
	url := fmt.Sprintf("http://fireup.%s/api/%s?name=%s", globalCfg.TLD, cmd, appName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to fireup: %v (is it running?)", err)
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
	printLogo()
	fmt.Println()
	fmt.Println("Welcome to fireup setup!")
	fmt.Println()
	fmt.Println("This wizard will configure fireup with three components:")
	fmt.Println()
	fmt.Println("  1. PORT FORWARDING")
	fmt.Println("     Redirects ports 80 and 443 to fireup, so you can access")
	fmt.Println("     your apps at http://myapp.test instead of http://localhost:9280")
	fmt.Println("     Requires: sudo")
	fmt.Println("       - Writes to /etc/pf.anchors/fireup (firewall rules)")
	fmt.Printf("       - Writes to /etc/resolver/%s (DNS resolution)\n", tld)
	fmt.Println()
	fmt.Println("  2. HTTPS CERTIFICATES")
	fmt.Println("     Generates a local Certificate Authority so https://myapp.test")
	fmt.Println("     works in your browser with no warnings.")
	fmt.Println("     Requires: sudo")
	fmt.Println("       - Adds CA to /Library/Keychains/System.keychain")
	fmt.Println()
	fmt.Println("  3. BACKGROUND SERVICE")
	fmt.Println("     Installs a LaunchAgent so fireup starts automatically")
	fmt.Println("     when you log in. Your apps are always ready!")
	fmt.Println("     Requires: nothing (runs as your user)")
	fmt.Println("       - Writes to ~/Library/LaunchAgents/com.fireup.plist")
	fmt.Println()
	fmt.Println("Each step will ask for confirmation before making changes.")
	fmt.Println("Steps that need sudo will prompt for your password.")

	// Check for root AFTER showing overview so user understands the context
	if os.Geteuid() == 0 {
		fmt.Println()
		fmt.Printf("%sError: Do not run 'fireup setup' with sudo.%s\n", colorRed, colorReset)
		fmt.Println()
		fmt.Println("Run it as your normal user instead:")
		fmt.Println("  fireup setup")
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
			fmt.Printf("%s⚠ Installed but config differs%s\n", colorYellow, colorReset)
			fmt.Println("  Found: /etc/pf.anchors/fireup")
			fmt.Println("  Found: /Library/LaunchDaemons/dev.fireup.pfctl.plist (differs)")
			fmt.Printf("  Found: /etc/resolver/%s\n", tld)
			fmt.Println()
			fmt.Println("Requires: sudo (will prompt for password)")
			fmt.Println()
			if confirmStep("Update port forwarding configuration?") {
				if err := runPortsInstall(configDir, tld); err != nil {
					fmt.Printf("\n%s⚠ Update failed: %v%s\n", colorYellow, err, colorReset)
				} else {
					fmt.Printf("%s✓ Port forwarding updated%s\n", colorGreen, colorReset)
				}
			} else {
				fmt.Println("Skipped. You can update later with: fireup ports install")
			}
		} else {
			fmt.Printf("%s✓ Already installed%s\n", colorGreen, colorReset)
			fmt.Println("  Found: /etc/pf.anchors/fireup")
			fmt.Println("  Found: /Library/LaunchDaemons/dev.fireup.pfctl.plist")
			fmt.Printf("  Found: /etc/resolver/%s\n", tld)
		}
	} else {
		fmt.Println("This step lets you access apps at http://myapp.test instead of")
		fmt.Println("http://localhost:9280. It configures macOS packet filter (pf) to")
		fmt.Println("redirect ports 80/443 to fireup.")
		fmt.Println()
		fmt.Println("Requires: sudo (will prompt for password)")

		// Show summary and confirm (? shows full diff)
		if confirmPortsInstall(tld, "Install port forwarding?") {
			// Set FIREUP_YES to skip the second confirmation in runPortsInstall
			os.Setenv("FIREUP_YES", "1")
			if err := runPortsInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Port forwarding failed: %v%s\n", colorYellow, err, colorReset)
				fmt.Println("You can retry later with: fireup ports install")
			} else {
				fmt.Printf("%s✓ Port forwarding installed%s\n", colorGreen, colorReset)
			}
			os.Unsetenv("FIREUP_YES")
		} else {
			fmt.Println("Skipped. You can run this later with: fireup ports install")
		}
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 2: Certificates
	fmt.Println()
	fmt.Println("Step 2/3: HTTPS Certificates")
	fmt.Println()
	if isCertInstalled(configDir) {
		fmt.Printf("%s✓ Already installed%s\n", colorGreen, colorReset)
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
			// Set FIREUP_YES to skip the second confirmation in runCertInstall
			os.Setenv("FIREUP_YES", "1")
			if err := runCertInstall(configDir, tld); err != nil {
				fmt.Printf("\n%s⚠ Certificate setup failed: %v%s\n", colorYellow, err, colorReset)
				fmt.Println("You can retry later with: fireup cert install")
			} else {
				fmt.Printf("%s✓ HTTPS certificates installed%s\n", colorGreen, colorReset)
				// Restart service if running so it picks up the new CA
				restartServiceIfRunning()
			}
			os.Unsetenv("FIREUP_YES")
		} else {
			fmt.Println("Skipped. You can run this later with: fireup cert install")
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
		fmt.Printf("%s✓ Already installed%s\n", colorGreen, colorReset)
		fmt.Println("  Found: ~/Library/LaunchAgents/com.fireup.plist")
		if running {
			fmt.Println("  Status: running")
		} else {
			fmt.Printf("  %sStatus: not running%s\n", colorYellow, colorReset)
		}
	} else {
		fmt.Println("This step makes fireup start automatically when you log in,")
		fmt.Println("so your apps are always accessible.")
		fmt.Println()
		fmt.Println("Will also start fireup immediately.")
		fmt.Println()
		fmt.Println("Requires: nothing (no sudo needed)")

		// Show summary and confirm (? shows full diff)
		plan := serviceInstallPlan()
		if confirmWithPlan(plan, "Install background service?") {
			// Set FIREUP_YES to skip the second confirmation in runServiceInstall
			os.Setenv("FIREUP_YES", "1")
			if err := runServiceInstall(); err != nil {
				fmt.Printf("\n%s⚠ Service install failed: %v%s\n", colorYellow, err, colorReset)
				fmt.Println("You can retry later with: fireup service install")
			} else {
				fmt.Printf("%s✓ Background service installed%s\n", colorGreen, colorReset)
			}
			os.Unsetenv("FIREUP_YES")
		} else {
			fmt.Println("Skipped. You can run this later with: fireup service install")
		}
	}
	fmt.Println()

	// Final summary
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Printf("  Dashboard:  http://fireup.%s\n", tld)
	fmt.Printf("  Dashboard:  https://fireup.%s (HTTPS)\n", tld)
	fmt.Println()
	fmt.Println("Create app configs in ~/.config/fireup/")
	fmt.Println("Run 'fireup status' to check component status.")
	fmt.Println()
	fmt.Println("Note: Restart your browser for HTTPS to work (quit fully and reopen).")
}

// runTeardownWizard removes all fireup configuration
func runTeardownWizard(tld string) {
	configDir := getDefaultConfigDir()

	fmt.Println("fireup teardown")
	fmt.Println()
	fmt.Println("This will remove all fireup components from your system.")
	fmt.Println("Each step will ask for confirmation before making changes.")
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Step 1: Background service
	fmt.Println()
	fmt.Println("Step 1/3: Background Service")
	fmt.Println()
	installed, running := isServiceInstalled()
	if !installed {
		fmt.Printf("%s✓ Already removed%s\n", colorGreen, colorReset)
		fmt.Println("  Not found: ~/Library/LaunchAgents/com.fireup.plist")
	} else {
		if running {
			fmt.Println("Will also stop running service.")
		}
		// Show summary and confirm (? shows full diff)
		plan := serviceUninstallPlan()
		if confirmWithPlan(plan, "Remove background service?") {
			// Unload the agent (ignore errors - may not be running)
			exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.fireup", os.Getuid())).Run()
			if err := plan.Execute(); err != nil {
				fmt.Printf("%s⚠ Service removal failed: %v%s\n", colorYellow, err, colorReset)
			} else {
				fmt.Printf("%s✓ Background service removed%s\n", colorGreen, colorReset)
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
		fmt.Printf("%s✓ Already removed%s\n", colorGreen, colorReset)
		fmt.Printf("  Not found: %s/certs/\n", configDir)
	} else {
		fmt.Println("Note: The CA in your system keychain must be removed manually")
		fmt.Println("      via Keychain Access (search for 'fireup Local CA')")
		// Show summary and confirm (? shows full diff)
		plan := certUninstallPlan()
		if confirmWithPlan(plan, "Remove HTTPS certificates?") {
			if err := plan.Execute(); err != nil {
				fmt.Printf("%s⚠ Certificate removal failed: %v%s\n", colorYellow, err, colorReset)
			} else {
				// Also try to remove the certs directory itself if empty
				os.Remove(getCertsDir())
				fmt.Printf("%s✓ HTTPS certificates removed%s\n", colorGreen, colorReset)
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
		fmt.Printf("%s✓ Already removed%s\n", colorGreen, colorReset)
		fmt.Println("  Not found: /etc/pf.anchors/fireup")
		fmt.Println("  Not found: /Library/LaunchDaemons/dev.fireup.pfctl.plist")
		fmt.Printf("  Not found: /etc/resolver/%s\n", tld)
	} else {
		fmt.Println("Requires: sudo (will prompt for password)")
		// Show summary and confirm (? shows full diff)
		plan := portsUninstallPlan(tld)
		if confirmWithPlan(plan, "Remove port forwarding?") {
			// Set FIREUP_YES to skip the second confirmation in runPortsUninstall
			os.Setenv("FIREUP_YES", "1")
			if err := runPortsUninstall(tld); err != nil {
				fmt.Printf("%s⚠ Port forwarding removal failed: %v%s\n", colorYellow, err, colorReset)
			}
			os.Unsetenv("FIREUP_YES")
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
	fmt.Println("To reinstall, run: fireup setup")
}

// checkPortsStatus returns the status of port forwarding
func checkPortsStatus() (string, string) {
	// Check if anchor file exists
	if _, err := os.Stat(pfAnchorPath); os.IsNotExist(err) {
		return "✗", "not installed"
	}

	// Check if pf rules are loaded by checking if our anchor has rules
	// Note: This requires root access, so it may fail even when forwarding works
	cmd := exec.Command("/sbin/pfctl", "-a", "fireup", "-sr")
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
	cmd := exec.Command("security", "find-certificate", "-c", "fireup Local CA", "/Library/Keychains/System.keychain")
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
		if strings.Contains(line, "com.fireup") {
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
