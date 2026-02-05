package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panozzaj/fireup/internal/config"
)

// resolveAppFromCwd tries to match the current working directory to a configured app.
// Returns the app name and true if found, or empty string and false otherwise.
func resolveAppFromCwd() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}

	// Resolve symlinks in cwd
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", false
	}
	cwd = filepath.Clean(cwd)

	apps := loadAppsForCLI()
	if len(apps) == 0 {
		return "", false
	}

	var bestName string
	var bestLen int

	for _, app := range apps {
		// Collect all directories associated with this app
		var dirs []string
		if app.Dir != "" {
			dirs = append(dirs, app.Dir)
		}
		if app.FilePath != "" {
			dirs = append(dirs, app.FilePath)
		}
		// Check service-level directories for multi-service apps
		for _, svc := range app.Services {
			if svc.Dir != "" {
				dirs = append(dirs, svc.Dir)
			}
		}

		for _, dir := range dirs {
			// Resolve symlinks and clean
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil {
				continue
			}
			resolved = filepath.Clean(resolved)

			if cwd == resolved || isSubdir(cwd, resolved) {
				if len(resolved) > bestLen {
					bestName = app.Name
					bestLen = len(resolved)
				}
			}
		}
	}

	if bestName != "" {
		return bestName, true
	}
	return "", false
}

// resolvedAppInfo returns info about a resolved app for display in help/docs.
type resolvedAppInfo struct {
	Name      string
	ConfigDir string
	TLD       string
}

// resolveAppInfo tries to resolve the CWD to an app and returns display info.
func resolveAppInfo() (*resolvedAppInfo, bool) {
	name, found := resolveAppFromCwd()
	if !found {
		return nil, false
	}

	globalCfg, configDir := getConfigWithDefaults()
	return &resolvedAppInfo{
		Name:      name,
		ConfigDir: configDir,
		TLD:       globalCfg.TLD,
	}, true
}

// isSubdir returns true if child is a subdirectory of parent.
func isSubdir(child, parent string) bool {
	// Ensure parent ends with separator for prefix check
	parentPrefix := parent
	if !strings.HasSuffix(parentPrefix, string(filepath.Separator)) {
		parentPrefix += string(filepath.Separator)
	}
	return strings.HasPrefix(child, parentPrefix)
}

// loadAppsForCLI creates an AppStore, loads configs, and returns all apps.
func loadAppsForCLI() []*config.App {
	configDir := getDefaultConfigDir()
	cfg := &config.Config{Dir: configDir}
	store := config.NewAppStore(cfg)
	if err := store.Load(); err != nil {
		return nil
	}
	return store.All()
}

// printCurrentProjectSection prints the "CURRENT PROJECT" section for help/docs output.
func printCurrentProjectSection(w *strings.Builder) {
	info, found := resolveAppInfo()
	if !found {
		return
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	displayCwd := cwd
	if home != "" && strings.HasPrefix(displayCwd, home) {
		displayCwd = "~" + displayCwd[len(home):]
	}

	displayConfigDir := info.ConfigDir
	if home != "" && strings.HasPrefix(displayConfigDir, home) {
		displayConfigDir = "~" + displayConfigDir[len(home):]
	}

	w.WriteString(fmt.Sprintf(`
CURRENT PROJECT (detected from %s):
    Name:       %s
    Config:     %s
    URL:        http://%s.%s

    In this directory, app name is optional:
        fireup restart     Restart %s
        fireup start       Start %s
        fireup stop        Stop %s
        fireup logs -f     Follow %s logs
`, displayCwd, info.Name,
		displayConfigDir,
		info.Name, info.TLD,
		info.Name, info.Name, info.Name, info.Name))
}
