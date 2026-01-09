package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/process"
	"github.com/panozzaj/roost-dev/internal/proxy"
	"github.com/panozzaj/roost-dev/internal/ui"
)

const asciiLogo = `
    ___  ___  ___  ____ _____      ___  ____ _  _
    |__| |  | |  | [__   |   ____ |  \ |___ |  |
    |  \ |__| |__| ___]  |        |__/ |___  \/
`

func errorPage(msg string) string {
	return asciiLogo + "\n" + msg + "\n"
}

func errorPageWithLogs(msg string, logs []string) string {
	result := asciiLogo + "\n" + msg + "\n"
	if len(logs) > 0 {
		result += "\n--- Recent logs ---\n"
		// Show last 20 lines
		start := 0
		if len(logs) > 20 {
			start = len(logs) - 20
		}
		for _, line := range logs[start:] {
			result += line + "\n"
		}
	}
	return result
}

func interstitialPage(appName, tld string, failed bool, errorMsg string) string {
	statusText := "Starting"
	if failed {
		statusText = "Failed to start"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s %s</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #1a1a2e;
            color: #eee;
            margin: 0;
            padding: 40px;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
        }
        .container {
            text-align: center;
            max-width: 700px;
            width: 100%%;
        }
        .logo {
            font-family: "SF Mono", Monaco, monospace;
            font-size: 12px;
            white-space: pre;
            color: #6b7280;
            margin-bottom: 40px;
        }
        h1 {
            font-size: 24px;
            margin: 0 0 16px 0;
            color: #fff;
        }
        .status {
            font-size: 16px;
            color: #9ca3af;
            margin-bottom: 24px;
        }
        .status.error {
            color: #f87171;
        }
        .spinner {
            width: 40px;
            height: 40px;
            border: 3px solid #374151;
            border-top-color: #22c55e;
            border-radius: 50%%;
            animation: spin 1s linear infinite;
            margin: 0 auto 24px;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        .logs {
            background: #0f172a;
            border: 1px solid #374151;
            border-radius: 8px;
            padding: 16px;
            text-align: left;
            max-height: 350px;
            overflow-y: auto;
            margin-bottom: 24px;
        }
        .logs-title {
            color: #9ca3af;
            font-size: 12px;
            margin-bottom: 8px;
        }
        .logs-content {
            font-family: "SF Mono", Monaco, monospace;
            font-size: 12px;
            line-height: 1.5;
            white-space: pre-wrap;
            word-break: break-all;
            color: #d1d5db;
            min-height: 100px;
        }
        .logs-empty {
            color: #6b7280;
            font-style: italic;
        }
        .retry-btn {
            background: #22c55e;
            color: #fff;
            border: none;
            padding: 10px 24px;
            border-radius: 6px;
            font-size: 14px;
            cursor: pointer;
            display: none;
        }
        .retry-btn:hover {
            background: #16a34a;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">%s</div>
        <h1>%s</h1>
        <div class="status" id="status">%s...</div>
        <div class="spinner" id="spinner"></div>
        <div class="logs" id="logs">
            <div class="logs-title">Logs</div>
            <div class="logs-content" id="logs-content"><span class="logs-empty">Waiting for output...</span></div>
        </div>
        <button class="retry-btn" id="retry-btn" onclick="location.reload()">Retry</button>
    </div>
    <script>
        const appName = '%s';
        const tld = '%s';
        let failed = %t;
        let lastLogCount = 0;

        async function poll() {
            try {
                // Fetch status and logs in parallel
                const [statusRes, logsRes] = await Promise.all([
                    fetch('http://roost-dev.' + tld + '/api/app-status?name=' + encodeURIComponent(appName)),
                    fetch('http://roost-dev.' + tld + '/api/logs?name=' + encodeURIComponent(appName))
                ]);
                const status = await statusRes.json();
                const lines = await logsRes.json();

                // Update logs
                if (lines && lines.length > 0) {
                    const content = document.getElementById('logs-content');
                    content.textContent = lines.join('\n');
                    // Auto-scroll if new lines
                    if (lines.length > lastLogCount) {
                        const logsDiv = document.getElementById('logs');
                        logsDiv.scrollTop = logsDiv.scrollHeight;
                        lastLogCount = lines.length;
                    }
                }

                // Check status
                if (status.status === 'running') {
                    document.getElementById('status').textContent = 'Ready! Redirecting...';
                    document.getElementById('spinner').style.borderTopColor = '#22c55e';
                    setTimeout(() => location.reload(), 300);
                    return;
                } else if (status.status === 'failed') {
                    showError(status.error);
                    return;
                }

                // Still starting, poll again
                setTimeout(poll, 500);
            } catch (e) {
                console.error('Poll failed:', e);
                setTimeout(poll, 1000);
            }
        }

        function showError(msg) {
            document.getElementById('spinner').style.display = 'none';
            const statusEl = document.getElementById('status');
            statusEl.textContent = 'Failed to start' + (msg ? ': ' + msg : '');
            statusEl.classList.add('error');
            document.getElementById('retry-btn').style.display = 'inline-block';
        }

        if (failed) {
            showError('%s');
            // Still fetch logs once for failed state
            fetch('http://roost-dev.' + tld + '/api/logs?name=' + encodeURIComponent(appName))
                .then(r => r.json())
                .then(lines => {
                    if (lines && lines.length > 0) {
                        document.getElementById('logs-content').textContent = lines.join('\n');
                    }
                });
        } else {
            poll();
        }
    </script>
</body>
</html>`, statusText, appName, asciiLogo, appName, statusText, appName, tld, failed, errorMsg)
}

// Server is the main roost-dev server
type Server struct {
	cfg     *config.Config
	apps    *config.AppStore
	procs   *process.Manager
	httpSrv *http.Server
}

// New creates a new server
func New(cfg *config.Config) (*Server, error) {
	apps := config.NewAppStore(cfg)
	if err := apps.Load(); err != nil {
		return nil, fmt.Errorf("loading apps: %w", err)
	}

	return &Server{
		cfg:   cfg,
		apps:  apps,
		procs: process.NewManager(),
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.HTTPPort)
	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.procs.StopAll()
	if s.httpSrv != nil {
		s.httpSrv.Close()
	}
}

// handleRequest routes requests based on hostname
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	// Remove port from host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Check for dashboard
	if host == "roost-dev."+s.cfg.TLD || host == "roost-dev" {
		s.handleDashboard(w, r)
		return
	}

	// Parse hostname: [service-]appname.tld
	if !strings.HasSuffix(host, "."+s.cfg.TLD) {
		http.Error(w, errorPage(fmt.Sprintf("Invalid host: %s (expected *.%s)", host, s.cfg.TLD)), http.StatusBadRequest)
		return
	}

	// Remove TLD
	name := strings.TrimSuffix(host, "."+s.cfg.TLD)

	// Check for service-app pattern (service-appname)
	if idx := strings.Index(name, "-"); idx != -1 {
		serviceName := name[:idx]
		appName := name[idx+1:]

		// Try to find as multi-service app
		app, service, found := s.apps.GetService(appName, serviceName)
		if found {
			s.handleService(w, r, app, service)
			return
		}
		// If not found as service, continue to try other patterns
	}

	// Try progressively shorter names to support subdomains
	// e.g., admin.mateams → try "admin.mateams", then "mateams"
	app, found := s.findApp(name)
	if !found {
		// Reload config and try again
		s.apps.Reload()
		app, found = s.findApp(name)
	}

	if !found {
		http.Error(w, errorPage(fmt.Sprintf("App not found: %s\n\nCreate a config file at: %s/%s", name, s.cfg.Dir, name)), http.StatusNotFound)
		return
	}

	s.handleApp(w, r, app)
}

// findApp tries to find an app by progressively shorter names
// e.g., "admin.mateams" → try "admin.mateams", then "mateams"
func (s *Server) findApp(name string) (*config.App, bool) {
	// Try exact match first
	if app, found := s.apps.Get(name); found {
		return app, true
	}

	// Try progressively shorter names (strip leading subdomain)
	for {
		idx := strings.Index(name, ".")
		if idx == -1 {
			break
		}
		name = name[idx+1:]
		if app, found := s.apps.Get(name); found {
			return app, true
		}
	}

	return nil, false
}

// handleApp handles a request for a simple app
func (s *Server) handleApp(w http.ResponseWriter, r *http.Request, app *config.App) {
	switch app.Type {
	case config.AppTypePort:
		// Simple proxy to fixed port
		proxy.NewReverseProxy(app.Port).ServeHTTP(w, r)

	case config.AppTypeCommand:
		// Check process status and serve appropriately
		proc, found := s.procs.Get(app.Name)
		if found && proc.IsRunning() {
			// Already running - proxy directly
			proxy.NewReverseProxy(proc.Port).ServeHTTP(w, r)
			return
		}
		if found && proc.HasFailed() {
			// Failed - show interstitial with error
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(interstitialPage(app.Name, s.cfg.TLD, true, proc.ExitError())))
			return
		}
		if found && proc.IsStarting() {
			// Starting - show interstitial
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(interstitialPage(app.Name, s.cfg.TLD, false, "")))
			return
		}
		// Idle - start async and show interstitial
		s.procs.StartAsync(app.Name, app.Command, app.Dir, app.Env)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(interstitialPage(app.Name, s.cfg.TLD, false, "")))

	case config.AppTypeStatic:
		// Serve static files
		proxy.NewStaticHandler(app.FilePath).ServeHTTP(w, r)

	case config.AppTypeYAML:
		// Multi-service app - use default service, first service, or show list
		if len(app.Services) == 1 {
			s.handleService(w, r, app, &app.Services[0])
			return
		}

		// Check for a default service
		for i := range app.Services {
			if app.Services[i].Default {
				s.handleService(w, r, app, &app.Services[i])
				return
			}
		}

		// Show available services
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>%s</h1>\n<p>Available services:</p>\n<ul>\n", app.Name)
		for _, svc := range app.Services {
			url := fmt.Sprintf("http://%s-%s.%s", svc.Name, app.Name, s.cfg.TLD)
			fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>\n", url, svc.Name)
		}
		fmt.Fprintf(w, "</ul>\n")
	}
}

// findService finds a service by name within an app
func (s *Server) findService(app *config.App, name string) *config.Service {
	for i := range app.Services {
		if app.Services[i].Name == name {
			return &app.Services[i]
		}
	}
	return nil
}

// ensureDependencies starts any dependencies that aren't already running
func (s *Server) ensureDependencies(app *config.App, svc *config.Service) {
	for _, depName := range svc.DependsOn {
		dep := s.findService(app, depName)
		if dep == nil {
			continue // Skip unknown dependencies
		}
		procName := fmt.Sprintf("%s-%s", dep.Name, app.Name)
		proc, found := s.procs.Get(procName)
		if !found || (!proc.IsRunning() && !proc.IsStarting()) {
			// Start the dependency
			s.procs.StartAsync(procName, dep.Command, dep.Dir, dep.Env)
		}
	}
}

// handleService handles a request for a service within a multi-service app
func (s *Server) handleService(w http.ResponseWriter, r *http.Request, app *config.App, svc *config.Service) {
	// Start dependencies first
	s.ensureDependencies(app, svc)

	procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)

	// Check process status and serve appropriately
	proc, found := s.procs.Get(procName)
	if found && proc.IsRunning() {
		// Already running - proxy directly
		proxy.NewReverseProxy(proc.Port).ServeHTTP(w, r)
		return
	}
	if found && proc.HasFailed() {
		// Failed - show interstitial with error
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(interstitialPage(procName, s.cfg.TLD, true, proc.ExitError())))
		return
	}
	if found && proc.IsStarting() {
		// Starting - show interstitial
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(interstitialPage(procName, s.cfg.TLD, false, "")))
		return
	}
	// Idle - start async and show interstitial
	s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(interstitialPage(procName, s.cfg.TLD, false, "")))
}

// ensureProcess ensures a process is running
func (s *Server) ensureProcess(name, command, dir string, env map[string]string) (*process.Process, error) {
	// Check if already running
	if proc, found := s.procs.Get(name); found && proc.IsRunning() {
		return proc, nil
	}

	// Start new process
	return s.procs.Start(name, command, dir, env)
}

// startByName starts a process by its name (e.g., "myapp" or "web-myapp" for services)
func (s *Server) startByName(name string) {
	// Try as an app first
	if app, found := s.apps.Get(name); found {
		switch app.Type {
		case config.AppTypeCommand:
			s.procs.StartAsync(app.Name, app.Command, app.Dir, app.Env)
		case config.AppTypeYAML:
			// Start all services for multi-service app, respecting depends_on
			for i := range app.Services {
				svc := &app.Services[i]
				s.ensureDependencies(app, svc)
				procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
				s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
			}
		}
		return
	}

	// Try as a service (format: "service-appname")
	for _, app := range s.apps.All() {
		if app.Type != config.AppTypeYAML {
			continue
		}
		for i := range app.Services {
			svc := &app.Services[i]
			procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
			if procName == name {
				// Start dependencies first
				s.ensureDependencies(app, svc)
				s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
				return
			}
		}
	}
}

// handleDashboard serves the web UI
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
		ui.ServeIndex(w, r, s.cfg.TLD, s.cfg.URLPort)

	case "/api/status":
		s.handleAPIStatus(w, r)

	case "/api/reload":
		s.apps.Reload()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	case "/api/stop":
		name := r.URL.Query().Get("name")
		if name != "" {
			// Try direct process name first
			if _, found := s.procs.Get(name); found {
				s.procs.Stop(name)
			} else if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
				// Stop all services for multi-service app
				for _, svc := range app.Services {
					procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
					s.procs.Stop(procName)
				}
			}
		}
		w.WriteHeader(http.StatusOK)

	case "/api/restart":
		name := r.URL.Query().Get("name")
		if name != "" {
			// Try direct process name first
			if proc, found := s.procs.Get(name); found {
				s.procs.Restart(proc.Name)
			} else if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
				// Restart all services for multi-service app
				for i := range app.Services {
					svc := &app.Services[i]
					procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
					if proc, found := s.procs.Get(procName); found {
						s.procs.Restart(proc.Name)
					} else {
						s.ensureDependencies(app, svc)
						s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
					}
				}
			} else {
				// Try to start it fresh
				s.startByName(name)
			}
		}
		w.WriteHeader(http.StatusOK)

	case "/api/start":
		name := r.URL.Query().Get("name")
		if name != "" {
			s.startByName(name)
		}
		w.WriteHeader(http.StatusOK)

	case "/api/logs":
		name := r.URL.Query().Get("name")
		var allLogs []string

		// Try direct process name first
		if proc, found := s.procs.Get(name); found {
			allLogs = proc.Logs().Lines()
		} else {
			// For multi-service apps, aggregate logs from all services
			if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
				for _, svc := range app.Services {
					procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
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

	case "/api/app-status":
		name := r.URL.Query().Get("name")
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)

	default:
		http.NotFound(w, r)
	}
}

// handleAPIStatus returns status of all apps and processes
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	type serviceStatus struct {
		Name    string `json:"name"`
		Running bool   `json:"running"`
		Failed  bool   `json:"failed,omitempty"`
		Error   string `json:"error,omitempty"`
		Port    int    `json:"port,omitempty"`
		Uptime  string `json:"uptime,omitempty"`
	}

	type appStatus struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Type        string          `json:"type"`
		URL         string          `json:"url"`
		Running     bool            `json:"running,omitempty"`
		Failed      bool            `json:"failed,omitempty"`
		Error       string          `json:"error,omitempty"`
		Port        int             `json:"port,omitempty"`
		Uptime      string          `json:"uptime,omitempty"`
		Services    []serviceStatus `json:"services,omitempty"`
	}

	var status []appStatus

	// Build base URL with port if not 80
	baseURL := func(name string) string {
		if s.cfg.URLPort == 80 {
			return fmt.Sprintf("http://%s.%s", name, s.cfg.TLD)
		}
		return fmt.Sprintf("http://%s.%s:%d", name, s.cfg.TLD, s.cfg.URLPort)
	}

	for _, app := range s.apps.All() {
		as := appStatus{
			Name:        app.Name,
			Description: app.Description,
			URL:         baseURL(app.Name),
		}

		switch app.Type {
		case config.AppTypePort:
			as.Type = "port"
			as.Port = app.Port
			as.Running = true // Assumed running

		case config.AppTypeCommand:
			as.Type = "command"
			if proc, found := s.procs.Get(app.Name); found {
				if proc.IsRunning() {
					as.Running = true
					as.Port = proc.Port
					as.Uptime = proc.Uptime().Round(1e9).String()
				} else if proc.HasFailed() {
					as.Failed = true
					as.Error = proc.ExitError()
				}
			}

		case config.AppTypeStatic:
			as.Type = "static"
			as.Running = true

		case config.AppTypeYAML:
			as.Type = "multi-service"
			// Keep base URL (app.test) - default service routes there automatically
			for _, svc := range app.Services {
				ss := serviceStatus{Name: svc.Name}
				procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
				if proc, found := s.procs.Get(procName); found {
					if proc.IsRunning() {
						ss.Running = true
						ss.Port = proc.Port
						ss.Uptime = proc.Uptime().Round(1e9).String()
					} else if proc.HasFailed() {
						ss.Failed = true
						ss.Error = proc.ExitError()
					}
				}
				as.Services = append(as.Services, ss)
			}
		}

		status = append(status, as)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
