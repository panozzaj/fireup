package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/server/pages"
	"github.com/panozzaj/roost-dev/internal/ui"
)

// parseServiceName parses a name in "service-app" format.
// Returns (serviceName, appName, isServiceFormat).
// For "web-myapp" returns ("web", "myapp", true).
// For "myapp" returns ("", "myapp", false).
func parseServiceName(name string) (serviceName, appName string, ok bool) {
	if idx := strings.Index(name, "-"); idx != -1 {
		return name[:idx], name[idx+1:], true
	}
	return "", name, false
}

// getClaudeCommand reads the claude_command from config.json fresh each time,
// allowing changes without restarting the server.
func (s *Server) getClaudeCommand() string {
	configPath := filepath.Join(s.cfg.Dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return s.cfg.ClaudeCommand // fallback to cached value
	}
	var cfg struct {
		ClaudeCommand string `json:"claude_command"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return s.cfg.ClaudeCommand
	}
	if cfg.ClaudeCommand == "" {
		return s.cfg.ClaudeCommand
	}
	return cfg.ClaudeCommand
}

// handleDashboard serves the web UI and API endpoints
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers for API endpoints (needed for interstitial page cross-origin fetches)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	switch r.URL.Path {
	case "/":
		ui.ServeIndex(w, r, s.cfg.TLD, s.cfg.URLPort, s.getStatus(), s.getTheme())

	case "/icons":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(pages.IconsTestPage(s.getTheme())))

	case "/api/status":
		s.handleAPIStatus(w, r)

	case "/api/events":
		s.handleSSE(w, r)

	case "/api/reload":
		s.apps.Reload()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	case "/api/stop":
		s.handleStop(w, r)

	case "/api/restart":
		s.handleRestart(w, r)

	case "/api/start":
		s.handleStart(w, r)

	case "/api/logs":
		s.handleLogs(w, r)

	case "/api/server-logs":
		// Return roost-dev's request handling logs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.requestLog.Lines())

	case "/api/debug-request":
		// Debug endpoint to see incoming request details (useful for Tailscale Serve testing)
		info := map[string]interface{}{
			"host":       r.Host,
			"url":        r.URL.String(),
			"method":     r.Method,
			"remoteAddr": r.RemoteAddr,
			"headers":    map[string][]string(r.Header),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)

	case "/api/app-status":
		s.handleAppStatus(w, r)

	case "/api/analyze-logs":
		s.handleAnalyzeLogs(w, r)

	case "/api/theme":
		if r.Method == "POST" {
			var req struct {
				Theme string `json:"theme"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if err := s.setTheme(req.Theme); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.broadcastTheme(req.Theme)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"theme": req.Theme})
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"theme": s.getTheme()})
		}

	case "/api/claude-enabled":
		// Check if claude command is configured
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"enabled": s.getClaudeCommand() != ""})

	case "/api/open-terminal":
		s.handleOpenTerminal(w, r)

	case "/api/open-config":
		s.handleOpenConfig(w, r)

	case "/api/config-path":
		s.handleConfigPath(w, r)

	default:
		http.NotFound(w, r)
	}
}

// handleStop stops an app or service
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name != "" {
		// First try to resolve as a service name (supports app:svc, svc.app, svc, svc-app)
		if match := s.resolveServiceName(name); match != nil {
			s.procs.Stop(match.ProcName)
			s.broadcastStatus()
			w.WriteHeader(http.StatusOK)
			return
		}
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		// Try direct process name first
		if _, found := s.procs.Get(name); found {
			s.procs.Stop(name)
		} else if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
			// Stop all services for multi-service app
			for _, svc := range app.Services {
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				s.procs.Stop(procName)
			}
		}
		s.broadcastStatus()
	}
	w.WriteHeader(http.StatusOK)
}

// handleRestart restarts an app or service
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	s.logRequest("API restart called for: %s", name)
	if name != "" {
		// First try to resolve as a service name (supports app:svc, svc.app, svc, svc-app)
		if match := s.resolveServiceName(name); match != nil {
			s.logRequest("  Restarting service: %s", match.ProcName)
			s.procs.Stop(match.ProcName)
			s.ensureDependencies(match.App, match.Service)
			s.procs.StartAsync(match.ProcName, match.Service.Command, match.Service.Dir, match.Service.Env)
			s.broadcastStatus()
			w.WriteHeader(http.StatusOK)
			return
		}
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		// Try direct process name first
		if proc, found := s.procs.Get(name); found {
			s.logRequest("  Restarting process: %s", proc.Name)
			// Stop then start fresh to pick up any config changes
			s.procs.Stop(proc.Name)
			s.startByName(name)
		} else if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
			// Restart all services for multi-service app
			// Stop ALL existing processes first (including those still starting/hung)
			for i := range app.Services {
				svc := &app.Services[i]
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				if proc, found := s.procs.Get(procName); found {
					status := "idle"
					if proc.IsRunning() {
						status = "running"
					} else if proc.IsStarting() {
						status = "starting"
					} else if proc.HasFailed() {
						status = "failed"
					}
					s.logRequest("  Stopping %s (was %s)", procName, status)
					s.procs.Stop(procName)
				}
			}
			// Now start all services fresh with current config
			for i := range app.Services {
				svc := &app.Services[i]
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				s.ensureDependencies(app, svc)
				s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
			}
		} else {
			// Try to start it fresh
			s.startByName(name)
		}
		s.broadcastStatus()
	}
	w.WriteHeader(http.StatusOK)
}

// handleStart starts an app or service
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name != "" {
		// First try to resolve as a service name (supports app:svc, svc.app, svc, svc-app)
		if match := s.resolveServiceName(name); match != nil {
			s.ensureDependencies(match.App, match.Service)
			s.procs.StartAsync(match.ProcName, match.Service.Command, match.Service.Dir, match.Service.Env)
			s.broadcastStatus()
			w.WriteHeader(http.StatusOK)
			return
		}
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		s.startByName(name)
		s.broadcastStatus()
	}
	w.WriteHeader(http.StatusOK)
}

// handleLogs returns logs for an app or service
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	// Resolve alias to app name
	if app, found := s.apps.GetByNameOrAlias(name); found {
		name = app.Name
	}
	var allLogs []string

	// Try direct process name first
	if proc, found := s.procs.Get(name); found {
		allLogs = proc.Logs().Lines()
	} else {
		// For multi-service apps, aggregate logs from all services
		if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
			for _, svc := range app.Services {
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				if proc, found := s.procs.Get(procName); found {
					for _, line := range proc.Logs().Lines() {
						allLogs = append(allLogs, fmt.Sprintf("[%s] %s", svc.Name, line))
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allLogs)
}

// handleAppStatus returns the status of a single app or service
func (s *Server) handleAppStatus(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	// Resolve alias to app name
	if app, found := s.apps.GetByNameOrAlias(name); found {
		name = app.Name
	}
	type singleAppStatus struct {
		Status string `json:"status"` // idle, starting, running, failed
		Error  string `json:"error,omitempty"`
	}

	status := singleAppStatus{Status: "idle"}
	if proc, found := s.procs.Get(name); found {
		if proc.IsStarting() {
			status.Status = "starting"
		} else if proc.IsRunning() {
			status.Status = "running"
		} else if proc.HasFailed() {
			status.Status = "failed"
			status.Error = proc.ExitError()
		}
	}

	// If this is a service, also check that dependencies are running
	// Format: "service-appname" -> check dependencies of service
	if status.Status == "running" {
		if serviceName, appName, ok := parseServiceName(name); ok {
			if app, svc, found := s.apps.GetService(appName, serviceName); found {
				for _, depName := range svc.DependsOn {
					// Skip dependencies that no longer exist in config
					if _, depSvc, depFound := s.apps.GetService(appName, depName); !depFound || depSvc == nil {
						continue
					}
					depProcName := fmt.Sprintf("%s-%s", depName, app.Name)
					depProc, found := s.procs.Get(depProcName)
					if !found {
						// Dependency not started yet - report starting
						status.Status = "starting"
						break
					}
					if depProc.IsStarting() {
						status.Status = "starting" // Dependency still starting
						break
					} else if depProc.HasFailed() {
						status.Status = "failed"
						status.Error = fmt.Sprintf("dependency %s failed: %s", depName, depProc.ExitError())
						break
					} else if !depProc.IsRunning() {
						status.Status = "starting" // Dependency not ready
						break
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleAnalyzeLogs uses Ollama to identify error lines in logs
func (s *Server) handleAnalyzeLogs(w http.ResponseWriter, r *http.Request) {
	if s.ollamaClient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
		return
	}

	name := r.URL.Query().Get("name")
	if app, found := s.apps.GetByNameOrAlias(name); found {
		name = app.Name
	}

	var logs []string
	if proc, found := s.procs.Get(name); found {
		logs = proc.Logs().Lines()
	}

	if len(logs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true, "errorLines": []int{}})
		return
	}

	// Run analysis async-ish but return result
	errorLines, err := s.ollamaClient.AnalyzeLogs(context.Background(), logs)
	if err != nil {
		s.logRequest("Ollama analysis error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true, "errorLines": errorLines})
}

// handleOpenTerminal opens iTerm2 with Claude Code for fixing app errors
func (s *Server) handleOpenTerminal(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter required", http.StatusBadRequest)
		return
	}

	// Resolve alias to app name
	if app, found := s.apps.GetByNameOrAlias(name); found {
		name = app.Name
	}

	// Find directory and logs for this app/service
	var dir string
	var logs []string

	// Try direct process name first (for services like "web-myapp")
	if proc, found := s.procs.Get(name); found {
		dir = proc.Dir
		logs = proc.Logs().Lines()
	}

	// If no process found, try as an app
	if dir == "" {
		if app, found := s.apps.Get(name); found {
			dir = app.Dir
			// For multi-service apps, try to get logs from first failed service
			if app.Type == config.AppTypeYAML {
				for _, svc := range app.Services {
					procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
					if proc, found := s.procs.Get(procName); found {
						if proc.HasFailed() {
							dir = svc.Dir
							if dir == "" {
								dir = proc.Dir
							}
							logs = proc.Logs().Lines()
							break
						}
					}
				}
			}
		}
	}

	// If still no dir, try to parse as service-appname
	if dir == "" {
		if serviceName, appName, ok := parseServiceName(name); ok {
			if _, svc, found := s.apps.GetService(appName, serviceName); found {
				dir = svc.Dir
			}
		}
	}

	if dir == "" {
		http.Error(w, "could not find directory for app", http.StatusNotFound)
		return
	}

	// Check if claude command is configured (read fresh from config.json)
	claudeCmd := s.getClaudeCommand()
	if claudeCmd == "" {
		http.Error(w, "claude command not configured in ~/.config/roost-dev/config.json", http.StatusBadRequest)
		return
	}

	// Build the prompt for Claude Code with roost-dev context
	logsText := strings.Join(logs, "\n")
	prompt := fmt.Sprintf(`The roost-dev app %q failed to start.

## About roost-dev
roost-dev is a local development server that manages apps via config files in ~/.config/roost-dev/.
Config file for this app: ~/.config/roost-dev/%s.yml

## Logs
`+"```"+`
%s
`+"```"+`

## Useful commands
  roost-dev restart %s  # Restart this app
  roost-dev logs %s     # View logs
  roost-dev --help      # CLI help
  roost-dev docs        # Full documentation

Please help me fix this error. After fixing, restart the app and verify it starts successfully.`,
		name, name, logsText, name, name)

	// Write prompt to a temp file to avoid shell escaping issues
	tmpFile, err := os.CreateTemp("", "roost-dev-prompt-*.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	promptFile := tmpFile.Name()
	if _, err := tmpFile.WriteString(prompt); err != nil {
		tmpFile.Close()
		os.Remove(promptFile)
		http.Error(w, fmt.Sprintf("failed to write prompt: %v", err), http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	// Use osascript to open iTerm2 with cd and claude command (interactive session)
	// Escape single quotes in the path for shell safety
	escDir := strings.ReplaceAll(dir, "'", "'\\''")
	escPromptFile := strings.ReplaceAll(promptFile, "'", "'\\''")
	script := fmt.Sprintf(`tell application "iTerm"
	activate
	set newWindow to (create window with default profile)
	tell current session of newWindow
		write text "cd '%s' && %s \"$(cat '%s')\" ; rm -f '%s'"
	end tell
end tell`, escDir, claudeCmd, escPromptFile, escPromptFile)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(promptFile)
		s.logRequest("Failed to open terminal: %v, output: %s", err, string(output))
		http.Error(w, fmt.Sprintf("failed to open terminal: %v (%s)", err, string(output)), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// getConfigPath returns the config file path for an app
func (s *Server) getConfigPath(name string) string {
	// For service names like "web-myapp", extract the app name
	appName := name
	if _, possibleApp, ok := parseServiceName(name); ok {
		if _, found := s.apps.Get(possibleApp); found {
			appName = possibleApp
		}
	}

	// Resolve alias to app name
	if app, found := s.apps.GetByNameOrAlias(appName); found {
		appName = app.Name
	}

	// Check for .yml first, then .yaml
	ymlPath := fmt.Sprintf("%s/%s.yml", s.cfg.Dir, appName)
	if _, err := os.Stat(ymlPath); err == nil {
		return ymlPath
	}
	yamlPath := fmt.Sprintf("%s/%s.yaml", s.cfg.Dir, appName)
	if _, err := os.Stat(yamlPath); err == nil {
		return yamlPath
	}
	// Check for plain file (no extension)
	plainPath := fmt.Sprintf("%s/%s", s.cfg.Dir, appName)
	if _, err := os.Stat(plainPath); err == nil {
		return plainPath
	}
	// Default to .yml
	return ymlPath
}

// handleConfigPath returns the config file path for an app
func (s *Server) handleConfigPath(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter required", http.StatusBadRequest)
		return
	}

	configPath := s.getConfigPath(name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": configPath})
}

// handleOpenConfig opens the config file in the default editor
func (s *Server) handleOpenConfig(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter required", http.StatusBadRequest)
		return
	}

	configPath := s.getConfigPath(name)

	// Use 'open' command on macOS to open in default editor
	cmd := exec.Command("open", configPath)
	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("failed to open config: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleWelcome serves the built-in welcome/test page at roost-test.<tld>
func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(pages.Welcome(s.cfg.TLD, s.cfg.Dir, s.getTheme())))
}
