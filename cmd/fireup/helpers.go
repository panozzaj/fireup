package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/panozzaj/fireup/internal/setup"
)

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
	cmd := exec.Command("launchctl", "list", "com.fireup")
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
		if !strings.Contains(string(data), "fireup") {
			warnings = append(warnings, fmt.Sprintf("existing resolver at %s (not from fireup)", resolverPath))
		}
	}

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
		fmt.Println("\nThese may conflict with fireup. Consider removing them first.")
		fmt.Print("Continue anyway? [y/N]: ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("installation cancelled")
		}
	}

	return nil
}
