package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/panozzaj/roost-dev/internal/diff"
)

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

// generateServicePlistContent generates the LaunchAgent plist content.
// This is extracted so the same content generation is used for both preview and execution.
func generateServicePlistContent() (string, error) {
	homeDir, _ := os.UserHomeDir()
	logsDir := filepath.Join(homeDir, "Library", "Logs", "roost-dev")

	// Find roost-dev binary
	binaryPath, err := exec.LookPath("roost-dev")
	if err != nil {
		// Fall back to go/bin
		binaryPath = filepath.Join(homeDir, "go", "bin", "roost-dev")
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
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
`, binaryPath, envSection.String(), logsDir, logsDir), nil
}

// serviceInstallPlan returns a Plan for service installation.
// The same Plan is used for both preview and execution to ensure they stay in sync.
func serviceInstallPlan() *diff.Plan {
	plan := diff.NewPlan()
	plan.Create(getUserLaunchAgentPath(), generateServicePlistContent)
	return plan
}

// serviceUninstallPlan returns a Plan for service removal.
func serviceUninstallPlan() *diff.Plan {
	plan := diff.NewPlan()
	plan.Delete(getUserLaunchAgentPath())
	return plan
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

	// Show summary and confirm (? shows full diff)
	plan := serviceInstallPlan()
	if !confirmWithPlan(plan, "Install background service?") {
		return fmt.Errorf("installation cancelled")
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

	// Execute the plan for file creation
	if err := plan.Execute(); err != nil {
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

	// Show summary and confirm (? shows full diff)
	plan := serviceUninstallPlan()
	if !confirmWithPlan(plan, "Remove background service?") {
		return fmt.Errorf("removal cancelled")
	}

	// Unload the agent
	exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.roost-dev", os.Getuid())).Run()

	// Execute the plan for file deletion
	if err := plan.Execute(); err != nil {
		return err
	}

	fmt.Printf("%s✓ Service removed%s\n", green, reset)
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
