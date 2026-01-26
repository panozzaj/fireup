package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func cmdLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)

	var (
		follow bool
		server bool
		lines  int
	)

	fs.BoolVar(&follow, "f", false, "Follow log output (poll for new logs)")
	fs.BoolVar(&server, "server", false, "Show server logs instead of app logs")
	fs.IntVar(&lines, "n", 0, "Number of lines to show (0 = all available)")

	fs.Usage = func() {
		fmt.Println(`roost-dev logs - View logs from roost-dev or apps

USAGE:
    roost-dev logs [options] [app-name]

OPTIONS:
  -f            Follow log output (poll for new logs)
  -n int        Number of lines to show (0 = all available)
  --server      Show server logs instead of app logs

EXAMPLES:
    roost-dev logs                  Show server request logs
    roost-dev logs myapp            Show logs for myapp
    roost-dev logs -f myapp         Follow myapp logs
    roost-dev logs --server         Show server logs (same as no args)
    roost-dev logs -n 50 myapp      Show last 50 lines of myapp logs

Requires the roost-dev server to be running.`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	// Load config to get TLD
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "roost-dev")
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		globalCfg = &GlobalConfig{TLD: "test"}
	}

	appName := fs.Arg(0)

	// If no app name and not explicitly --server, default to server logs
	if appName == "" {
		server = true
	}

	if follow {
		runLogsFollow(globalCfg.TLD, appName, server, lines)
	} else {
		if err := runLogsOnce(globalCfg.TLD, appName, server, lines); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// runLogsOnce fetches and prints logs once
func runLogsOnce(tld, appName string, server bool, maxLines int) error {
	var url string
	if server || appName == "" {
		url = fmt.Sprintf("http://roost-dev.%s/api/server-logs", tld)
	} else {
		url = fmt.Sprintf("http://roost-dev.%s/api/logs?name=%s", tld, appName)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to roost-dev: %v (is it running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("app not found: %s", appName)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var logLines []string
	if err := json.NewDecoder(resp.Body).Decode(&logLines); err != nil {
		return fmt.Errorf("failed to parse logs: %v", err)
	}

	// Apply line limit if specified
	if maxLines > 0 && len(logLines) > maxLines {
		logLines = logLines[len(logLines)-maxLines:]
	}

	for _, line := range logLines {
		fmt.Println(line)
	}

	return nil
}

// runLogsFollow continuously polls and prints new logs
func runLogsFollow(tld, appName string, server bool, maxLines int) {
	// Track what we've already printed to avoid duplicates
	var lastLen int
	firstRun := true

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println()
			return
		case <-ticker.C:
			var url string
			if server || appName == "" {
				url = fmt.Sprintf("http://roost-dev.%s/api/server-logs", tld)
			} else {
				url = fmt.Sprintf("http://roost-dev.%s/api/logs?name=%s", tld, appName)
			}

			resp, err := http.Get(url)
			if err != nil {
				if firstRun {
					fmt.Fprintf(os.Stderr, "Error: failed to connect to roost-dev: %v (is it running?)\n", err)
					os.Exit(1)
				}
				continue // Transient error, keep trying
			}

			var logLines []string
			json.NewDecoder(resp.Body).Decode(&logLines)
			resp.Body.Close()

			if firstRun {
				// On first run, apply line limit and print
				startIdx := 0
				if maxLines > 0 && len(logLines) > maxLines {
					startIdx = len(logLines) - maxLines
				}
				for i := startIdx; i < len(logLines); i++ {
					fmt.Println(logLines[i])
				}
				lastLen = len(logLines)
				firstRun = false
			} else if len(logLines) > lastLen {
				// Print only new lines
				for i := lastLen; i < len(logLines); i++ {
					fmt.Println(logLines[i])
				}
				lastLen = len(logLines)
			} else if len(logLines) < lastLen {
				// Buffer wrapped, print all new content
				for _, line := range logLines {
					fmt.Println(line)
				}
				lastLen = len(logLines)
			}
		}
	}
}
