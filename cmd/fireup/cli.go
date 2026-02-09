package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// ANSI color codes
const (
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
	colorMagenta   = "\033[35m"
	colorCyan      = "\033[36m"
	colorGray      = "\033[90m"
	colorBrightRed = "\033[91m"
	colorDim       = "\033[2m"
	colorReset     = "\033[0m"
)

// checkHelpFlag checks if any argument is a help flag and prints usage if so.
// Returns true if help was requested (and program should exit).
func checkHelpFlag(args []string, usage string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(usage)
			return true
		}
	}
	return false
}

// getDefaultConfigDir returns the default configuration directory path.
func getDefaultConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "fireup")
}

// getConfigWithDefaults loads the global config, returning defaults if not found.
// Returns the config and the config directory path.
func getConfigWithDefaults() (*GlobalConfig, string) {
	configDir := getDefaultConfigDir()
	globalCfg, err := loadGlobalConfig(configDir)
	if err != nil {
		globalCfg = &GlobalConfig{TLD: "test"}
	}
	return globalCfg, configDir
}

// requireNonRoot returns an error if running as root.
func requireNonRoot(action string) error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("cannot %s as root; run without sudo", action)
	}
	return nil
}

// isRoot returns true if running as root.
func isRoot() bool {
	return os.Geteuid() == 0
}

// isAgent returns true if running under an AI coding agent (e.g. Claude Code).
// Interactive prompts should fail fast when this is true.
func isAgent() bool {
	return os.Getenv("CLAUDECODE") == "1"
}
