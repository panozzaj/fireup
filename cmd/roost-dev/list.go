package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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

		// Pad status first, then add color codes (so ANSI codes don't affect width)
		paddedStatus := fmt.Sprintf("%-10s", status)
		switch {
		case status == "running":
			paddedStatus = "\033[32m" + paddedStatus + "\033[0m" // green
		case status == "idle":
			paddedStatus = "\033[90m" + paddedStatus + "\033[0m" // gray
		case strings.Contains(status, "/"):
			paddedStatus = "\033[33m" + paddedStatus + "\033[0m" // yellow for partial
		}

		name := app.Name
		if len(app.Aliases) > 0 {
			name = fmt.Sprintf("%s (%s)", app.Name, strings.Join(app.Aliases, ", "))
		}
		fmt.Printf("%-25s %s %s\n", name, paddedStatus, app.URL)

		// Print services for multi-service apps (tree view)
		if app.Type == "multi-service" && len(app.Services) > 0 {
			for i, svc := range app.Services {
				// Determine tree character
				var prefix string
				if i == len(app.Services)-1 {
					prefix = "└─"
				} else {
					prefix = "├─"
				}

				// Determine service status
				var svcStatus string
				if svc.Running {
					svcStatus = "running"
				} else {
					svcStatus = "idle"
				}

				// Format and colorize status
				svcPaddedStatus := fmt.Sprintf("%-10s", svcStatus)
				if svcStatus == "running" {
					svcPaddedStatus = "\033[32m" + svcPaddedStatus + "\033[0m" // green
				} else {
					svcPaddedStatus = "\033[90m" + svcPaddedStatus + "\033[0m" // gray
				}

				svcName := fmt.Sprintf("%s %s", prefix, svc.Name)
				fmt.Printf("  %-23s %s %s\n", svcName, svcPaddedStatus, svc.URL)
			}
		}
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
