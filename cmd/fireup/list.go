package main

import (
	"fmt"
	"os"
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

// cmdList handles the 'list' command (alias for status)
func cmdList(args []string) {
	if checkHelpFlag(args, `fireup list - List configured apps and their status

USAGE:
    fireup list [--json]

This command is an alias for 'fireup status'.
Shows all configured apps, their running status, and URLs.
Use --json for machine-readable output.`) {
		os.Exit(0)
	}

	// Delegate to status command
	cmdStatus(args)
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
		// Skip hidden files, config files, and directories
		if strings.HasPrefix(name, ".") || name == "config.json" || name == "config-theme.json" {
			continue
		}
		// Skip certs directory
		if entry.IsDir() && name == "certs" {
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
	fmt.Println("\nStart the server with: fireup serve")

	return nil
}
