package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// cmdDocs shows documentation
func cmdDocs(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`fireup docs - Show documentation

USAGE:
    fireup docs

Shows configuration and troubleshooting documentation.
Output is paged if running in a terminal.`)
			os.Exit(0)
		}
	}

	// Try to find docs file in several locations
	var content []byte
	var err error

	// 1. Try relative to current directory (for development)
	content, err = os.ReadFile("docs/fireup.txt")
	if err != nil {
		// 2. Try relative to executable
		if exe, exeErr := os.Executable(); exeErr == nil {
			content, err = os.ReadFile(filepath.Join(filepath.Dir(exe), "docs", "fireup.txt"))
		}
	}
	if err != nil {
		// 3. Try in source location (for go run)
		homeDir, _ := os.UserHomeDir()
		content, err = os.ReadFile(filepath.Join(homeDir, "Documents", "dev", "fireup", "docs", "fireup.txt"))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not find docs/fireup.txt\n")
		os.Exit(1)
	}

	// Append project section if CWD matches a project
	var sb strings.Builder
	printCurrentProjectSection(&sb)
	projectSection := sb.String()
	fullContent := string(content) + projectSection

	// If stdout is a terminal, use a pager
	if term.IsTerminal(int(os.Stdout.Fd())) {
		pager := getPager()
		if pager != "" {
			cmd := exec.Command(pager)
			cmd.Stdin = strings.NewReader(fullContent)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				return
			}
			// Fall through to direct output if pager fails
		}
	}

	// Direct output (non-terminal or no pager)
	fmt.Print(fullContent)
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
