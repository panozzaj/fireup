package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
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
		fmt.Println(`fireup logs - View logs from fireup or apps

USAGE:
    fireup logs [options] [app-name]

OPTIONS:
  -f            Follow log output (poll for new logs)
  -n int        Number of lines to show (0 = all available)
  --server      Show server logs instead of app logs

EXAMPLES:
    fireup logs                  Show server request logs
    fireup logs myapp            Show logs for myapp
    fireup logs -f myapp         Follow myapp logs
    fireup logs --server         Show server logs (same as no args)
    fireup logs -n 50 myapp      Show last 50 lines of myapp logs

Requires the fireup server to be running.`)
	}

	// Check for help before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	globalCfg, _ := getConfigWithDefaults()
	appName := fs.Arg(0)

	// If no app name and --server not explicitly set, try CWD resolution
	if appName == "" && !server {
		if resolved, found := resolveAppFromCwd(); found {
			fmt.Fprintf(os.Stderr, "(detected %s from current directory)\n", resolved)
			appName = resolved
		} else {
			server = true
		}
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
		url = fmt.Sprintf("http://fireup.%s/api/server-logs", tld)
	} else {
		url = fmt.Sprintf("http://fireup.%s/api/logs?name=%s", tld, appName)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to fireup: %v (is it running?)", err)
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

	lp := newLogPrinter()
	for _, line := range logLines {
		lp.Println(line)
	}

	return nil
}

// logPrinter handles colorized log output.
type logPrinter struct {
	colorize bool
	colors   map[string]string
	w        io.Writer
}

// newLogPrinter creates a log printer that colorizes prefixes when stdout is a terminal.
func newLogPrinter() *logPrinter {
	return &logPrinter{
		colorize: term.IsTerminal(int(os.Stdout.Fd())),
		colors:   make(map[string]string),
		w:        os.Stdout,
	}
}

// log prefix colors, cycled per unique prefix name
var prefixColors = []string{
	colorCyan,
	colorMagenta,
	colorGreen,
	colorBlue,
	colorBrightRed,
	colorYellow,
}

// Println prints a log line, colorizing any leading [prefix].
func (lp *logPrinter) Println(line string) {
	if !lp.colorize {
		fmt.Fprintln(lp.w, line)
		return
	}

	// Look for leading [prefix]
	if strings.HasPrefix(line, "[") {
		if idx := strings.Index(line, "] "); idx > 0 {
			prefix := line[1:idx]
			rest := line[idx+2:]

			color, ok := lp.colors[prefix]
			if !ok {
				color = prefixColors[len(lp.colors)%len(prefixColors)]
				lp.colors[prefix] = color
			}

			// Reset before prefix so it renders cleanly even if previous line
			// left ANSI state open. Don't reset after content so app styling
			// (e.g. bold spanning lines) flows naturally until the next prefix.
			fmt.Fprintf(lp.w, "%s%s[%s]%s %s\n", colorReset, color, prefix, colorReset, rest)
			return
		}
	}

	fmt.Fprintln(lp.w, line)
}

// runLogsFollow continuously polls and prints new logs
func runLogsFollow(tld, appName string, server bool, maxLines int) {
	// Track what we've already printed to avoid duplicates
	var lastLen int
	firstRun := true
	lp := newLogPrinter()

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
				url = fmt.Sprintf("http://fireup.%s/api/server-logs", tld)
			} else {
				url = fmt.Sprintf("http://fireup.%s/api/logs?name=%s", tld, appName)
			}

			resp, err := http.Get(url)
			if err != nil {
				if firstRun {
					fmt.Fprintf(os.Stderr, "Error: failed to connect to fireup: %v (is it running?)\n", err)
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
					lp.Println(logLines[i])
				}
				lastLen = len(logLines)
				firstRun = false
			} else if len(logLines) > lastLen {
				// Print only new lines
				for i := lastLen; i < len(logLines); i++ {
					lp.Println(logLines[i])
				}
				lastLen = len(logLines)
			} else if len(logLines) < lastLen {
				// Buffer wrapped, print all new content
				for _, line := range logLines {
					lp.Println(line)
				}
				lastLen = len(logLines)
			}
		}
	}
}
