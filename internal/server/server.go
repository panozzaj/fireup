package server

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/process"
	"github.com/panozzaj/roost-dev/internal/proxy"
	"github.com/panozzaj/roost-dev/internal/ui"
)

// slugify converts a name to a URL-safe slug (lowercase, spaces to dashes)
func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

const asciiLogo = `
                 __            __
   _________  ____  _____/ /_      ____/ /__ _   __
  / ___/ __ \/ __ \/ ___/ __/_____/ __  / _ \ | / /
 / /  / /_/ / /_/ (__  ) /_/_____/ /_/ /  __/ |/ /
/_/   \____/\____/____/\__/      \__,_/\___/|___/
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
        :root {
            --bg-primary: #1a1a2e;
            --bg-logs: #0f172a;
            --text-primary: #eee;
            --text-secondary: #9ca3af;
            --text-muted: #6b7280;
            --border-color: #374151;
            --btn-bg: #374151;
            --btn-hover: #4b5563;
            --logs-text: #d1d5db;
        }
        @media (prefers-color-scheme: light) {
            :root {
                --bg-primary: #f5f5f5;
                --bg-logs: #ffffff;
                --text-primary: #1a1a1a;
                --text-secondary: #4b5563;
                --text-muted: #6b7280;
                --border-color: #e5e7eb;
                --btn-bg: #e5e7eb;
                --btn-hover: #d1d5db;
                --logs-text: #374151;
            }
        }
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            margin: 0;
            padding: 60px 40px 40px;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
        }
        .container {
            text-align: center;
            max-width: 700px;
            width: 100%%;
        }
        .logo {
            font-family: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;
            font-size: 12px;
            white-space: pre;
            margin-bottom: 40px;
            letter-spacing: 0;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }
        .logo a {
            color: var(--text-muted);
            text-decoration: none;
            transition: color 0.3s;
        }
        .logo a:hover {
            background: linear-gradient(90deg, #ff6b6b, #feca57, #48dbfb, #ff9ff3, #54a0ff, #5f27cd);
            background-size: 200%% auto;
            -webkit-background-clip: text;
            background-clip: text;
            color: transparent;
            animation: rainbow 2s linear infinite;
        }
        @keyframes rainbow {
            0%% { background-position: 0%% center; }
            100%% { background-position: 200%% center; }
        }
        h1 {
            font-size: 24px;
            margin: 0 0 16px 0;
            color: var(--text-primary);
        }
        .status {
            font-size: 16px;
            color: var(--text-secondary);
            margin-bottom: 24px;
        }
        .status.error {
            color: #f87171;
        }
        .spinner {
            width: 40px;
            height: 40px;
            border: 3px solid var(--border-color);
            border-top-color: #22c55e;
            border-radius: 50%%;
            animation: spin 1s linear infinite;
            margin: 0 auto 24px;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        .logs {
            background: var(--bg-logs);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 16px;
            text-align: left;
            max-height: 350px;
            overflow-y: auto;
            margin-bottom: 24px;
        }
        .logs-title {
            color: var(--text-secondary);
            font-size: 12px;
            margin-bottom: 8px;
        }
        .logs-content {
            font-family: "SF Mono", Monaco, monospace;
            font-size: 12px;
            line-height: 1.5;
            white-space: pre-wrap;
            word-break: break-all;
            color: var(--logs-text);
            min-height: 100px;
        }
        .logs-empty {
            color: var(--text-muted);
            font-style: italic;
        }
        .btn {
            background: var(--btn-bg);
            color: var(--text-primary);
            border: none;
            padding: 10px 24px;
            border-radius: 6px;
            font-size: 14px;
            cursor: pointer;
        }
        .btn:hover {
            background: var(--btn-hover);
        }
        .btn-primary {
            background: #22c55e;
            color: #fff;
        }
        .btn-primary:hover:not(:disabled) {
            background: #16a34a;
        }
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        .retry-btn {
            display: none;
        }
        .logs-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: -16px;
            background: var(--bg-logs);
            z-index: 1;
            margin: -16px -16px 8px -16px;
            padding: 16px 16px 8px 16px;
            border-bottom: 1px solid var(--border-color);
        }
        .copy-btn {
            padding: 4px 12px;
            font-size: 12px;
            margin-top: -4px;
        }
    </style>
</head>
<body>
    <div class="container" data-error="%s" data-app="%s" data-tld="%s">
        <div class="logo"><a href="http://roost-dev.%s/" title="roost-dev dashboard">%s</a></div>
        <h1>%s</h1>
        <div class="status" id="status">%s...</div>
        <div class="spinner" id="spinner"></div>
        <div class="logs" id="logs">
            <div class="logs-header">
                <div class="logs-title">Logs</div>
                <button class="btn copy-btn" id="copy-btn" onclick="copyLogs()">Copy</button>
            </div>
            <div class="logs-content" id="logs-content"><span class="logs-empty">Waiting for output...</span></div>
        </div>
        <button class="btn btn-primary retry-btn" id="retry-btn" onclick="restartAndRetry()">Restart</button>
    </div>
    <script>
        const container = document.querySelector('.container');
        const appName = container.dataset.app;
        const tld = container.dataset.tld;
        let failed = %t;
        let lastLogCount = 0;
        const startTime = Date.now();
        const MIN_WAIT_MS = 2000; // Wait at least 2 seconds before redirecting

        async function poll() {
            console.log('poll() called');
            try {
                // Fetch status and logs in parallel
                const [statusRes, logsRes] = await Promise.all([
                    fetch('http://roost-dev.' + tld + '/api/app-status?name=' + encodeURIComponent(appName)),
                    fetch('http://roost-dev.' + tld + '/api/logs?name=' + encodeURIComponent(appName))
                ]);
                const status = await statusRes.json();
                const lines = await logsRes.json();
                console.log('poll() got status:', status.status, 'logs:', lines?.length || 0);

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
                    // Ensure minimum wait time has elapsed before redirecting
                    // This gives services time to fully initialize even after port is ready
                    const elapsed = Date.now() - startTime;
                    if (elapsed < MIN_WAIT_MS) {
                        document.getElementById('status').textContent = 'Almost ready...';
                        setTimeout(poll, MIN_WAIT_MS - elapsed);
                        return;
                    }
                    document.getElementById('status').textContent = 'Ready! Redirecting...';
                    document.getElementById('spinner').style.borderTopColor = '#22c55e';
                    setTimeout(() => location.reload(), 300);
                    return;
                } else if (status.status === 'failed') {
                    console.log('poll() detected failed, calling showError');
                    showError(status.error);
                    return;
                }

                // Still starting, poll again (fast for responsive UI)
                setTimeout(poll, 200);
            } catch (e) {
                console.error('Poll failed:', e);
                setTimeout(poll, 1000);
            }
        }

        function showError(msg) {
            console.log('showError() called with:', msg);
            document.getElementById('spinner').style.display = 'none';
            const statusEl = document.getElementById('status');
            statusEl.textContent = 'Failed to start' + (msg ? ': ' + msg : '');
            statusEl.classList.add('error');
            const btn = document.getElementById('retry-btn');
            btn.style.display = 'inline-block';
            btn.disabled = false;
            btn.textContent = 'Restart';
            console.log('showError() done, button should be visible and enabled');
        }

        function copyLogs() {
            const content = document.getElementById('logs-content');
            const btn = document.getElementById('copy-btn');
            const text = content.textContent;

            // We use execCommand('copy') instead of navigator.clipboard.writeText() because
            // the Clipboard API requires a "secure context" (HTTPS or literal localhost).
            // Even though *.test domains resolve to 127.0.0.1, browsers check the hostname,
            // not the resolved IP - so app.test is considered insecure.
            // execCommand('copy') is deprecated but works in non-secure contexts by copying
            // selected text from a DOM element (hence the hidden textarea trick).
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);

            btn.textContent = 'Copied!';
            setTimeout(() => btn.textContent = 'Copy', 500);
        }

        async function restartAndRetry() {
            const btn = document.getElementById('retry-btn');
            const statusEl = document.getElementById('status');
            btn.textContent = 'Restarting...';
            btn.disabled = true;
            statusEl.textContent = 'Restarting...';
            statusEl.classList.remove('error');
            document.getElementById('spinner').style.display = 'block';
            document.getElementById('logs-content').innerHTML = '<span class="logs-empty">Restarting...</span>';

            try {
                const url = 'http://roost-dev.' + tld + '/api/restart?name=' + encodeURIComponent(appName);
                const res = await fetch(url);
                if (!res.ok) {
                    throw new Error('Restart API returned ' + res.status);
                }
                // Reset state for fresh polling (but keep logs visible until new ones arrive)
                failed = false;
                lastLogCount = 0;
                statusEl.textContent = 'Starting...';
                btn.style.display = 'none';
                btn.textContent = 'Restart';
                btn.disabled = false;
                // Start polling immediately - restart API already waited 500ms internally
                poll();
            } catch (e) {
                console.error('Restart failed:', e);
                btn.textContent = 'Restart';
                btn.disabled = false;
                statusEl.textContent = 'Restart failed: ' + e.message;
                statusEl.classList.add('error');
                document.getElementById('spinner').style.display = 'none';
            }
        }

        if (failed) {
            // Get error from data attribute (safer than inline string)
            const errorMsg = container.dataset.error || '';
            showError(errorMsg);
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
</html>`,
		statusText,                  // title
		html.EscapeString(appName),  // title
		html.EscapeString(errorMsg), // data-error attribute
		html.EscapeString(appName),  // data-app attribute
		html.EscapeString(tld),      // data-tld attribute
		html.EscapeString(tld),      // logo link href
		asciiLogo,                   // logo text (hardcoded, safe)
		html.EscapeString(appName),  // h1
		statusText,                  // status text
		failed)                      // JS boolean
}

// Server is the main roost-dev server
type Server struct {
	cfg           *config.Config
	apps          *config.AppStore
	procs         *process.Manager
	httpSrv       *http.Server
	requestLog    *process.LogBuffer // Reuse LogBuffer for request logging
	broadcaster   *Broadcaster       // SSE broadcaster for real-time updates
	configWatcher *config.Watcher    // Watches config directory for changes
}

// New creates a new server
func New(cfg *config.Config) (*Server, error) {
	apps := config.NewAppStore(cfg)
	if err := apps.Load(); err != nil {
		return nil, fmt.Errorf("loading apps: %w", err)
	}

	s := &Server{
		cfg:         cfg,
		apps:        apps,
		procs:       process.NewManager(),
		requestLog:  process.NewLogBuffer(500), // Keep last 500 request log entries
		broadcaster: NewBroadcaster(),
	}

	// Set up config watcher
	watcher, err := config.NewWatcher(cfg.Dir, func() {
		if err := s.apps.Reload(); err != nil {
			s.logRequest("Config reload error: %v", err)
		} else {
			s.logRequest("Config reloaded")
			s.broadcastStatus()
		}
	})
	if err != nil {
		// Log but don't fail - config watching is optional
		fmt.Printf("Warning: could not watch config directory: %v\n", err)
	} else {
		s.configWatcher = watcher
	}

	return s, nil
}

// logRequest logs a request handling event
func (s *Server) logRequest(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	s.requestLog.Write([]byte(fmt.Sprintf("[%s] %s\n", timestamp, msg)))
	fmt.Printf("[%s] %s\n", timestamp, msg) // Also print to stdout
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start config watcher
	if s.configWatcher != nil {
		s.configWatcher.Start()
	}

	// Periodic status broadcast to catch state changes (process ready/failed)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if s.broadcaster.ClientCount() > 0 {
				s.broadcastStatus()
			}
		}
	}()

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
	if s.configWatcher != nil {
		s.configWatcher.Stop()
	}
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

	// Check for dashboard or roost-dev subdomains (for test services)
	if host == "roost-dev."+s.cfg.TLD || host == "roost-dev" {
		s.handleDashboard(w, r)
		return
	}
	if strings.HasSuffix(host, ".roost-dev."+s.cfg.TLD) {
		// Subdomain of roost-dev.test → route to roost-dev-tests services
		subdomain := strings.TrimSuffix(host, ".roost-dev."+s.cfg.TLD)
		if app, svc, found := s.apps.GetService("roost-dev-tests", subdomain); found {
			s.handleService(w, r, app, svc)
			return
		}
		http.Error(w, errorPage(fmt.Sprintf("Service not found: %s\n\nAvailable at roost-dev-tests", subdomain)), http.StatusNotFound)
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
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Write([]byte(interstitialPage(app.Name, s.cfg.TLD, true, proc.ExitError())))
			return
		}
		if found && proc.IsStarting() {
			// Starting - show interstitial
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Write([]byte(interstitialPage(app.Name, s.cfg.TLD, false, "")))
			return
		}
		// Idle - start async and show interstitial
		s.procs.StartAsync(app.Name, app.Command, app.Dir, app.Env)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
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
			url := fmt.Sprintf("http://%s-%s.%s", slugify(svc.Name), app.Name, s.cfg.TLD)
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
		procName := fmt.Sprintf("%s-%s", slugify(dep.Name), app.Name)
		proc, found := s.procs.Get(procName)
		if !found || (!proc.IsRunning() && !proc.IsStarting()) {
			// Start the dependency
			s.procs.StartAsync(procName, dep.Command, dep.Dir, dep.Env)
		}
	}
}

// handleService handles a request for a service within a multi-service app
func (s *Server) handleService(w http.ResponseWriter, r *http.Request, app *config.App, svc *config.Service) {
	procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
	s.logRequest("handleService: %s (path=%s)", procName, r.URL.Path)

	// Start dependencies first
	s.ensureDependencies(app, svc)

	// Check process status and serve appropriately
	proc, found := s.procs.Get(procName)
	s.logRequest("  %s: found=%v, running=%v, starting=%v, failed=%v",
		procName, found,
		found && proc.IsRunning(),
		found && proc.IsStarting(),
		found && proc.HasFailed())

	if found && proc.IsRunning() {
		// Already running - proxy directly
		s.logRequest("  -> PROXY to port %d", proc.Port)
		proxy.NewReverseProxy(proc.Port).ServeHTTP(w, r)
		return
	}
	if found && proc.HasFailed() {
		// Failed - show interstitial with error
		s.logRequest("  -> INTERSTITIAL (failed)")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(interstitialPage(procName, s.cfg.TLD, true, proc.ExitError())))
		return
	}
	if found && proc.IsStarting() {
		// Starting - show interstitial
		s.logRequest("  -> INTERSTITIAL (starting)")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(interstitialPage(procName, s.cfg.TLD, false, "")))
		return
	}
	// Idle - start async and show interstitial
	s.logRequest("  -> INTERSTITIAL (idle, starting %s)", procName)
	s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
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
			// TODO: Consider pre-allocating ports and passing PORT_<SERVICE> env vars
			// to each service, so they can reference each other's ports directly.
			// Example: frontend could use $PORT_API to connect to the api service.
			for i := range app.Services {
				svc := &app.Services[i]
				s.ensureDependencies(app, svc)
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
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
			procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
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
		ui.ServeIndex(w, r, s.cfg.TLD, s.cfg.URLPort, s.getStatus())

	case "/api/status":
		s.handleAPIStatus(w, r)

	case "/api/events":
		s.handleSSE(w, r)

	case "/api/reload":
		s.apps.Reload()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	case "/api/stop":
		name := r.URL.Query().Get("name")
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		if name != "" {
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

	case "/api/restart":
		name := r.URL.Query().Get("name")
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		s.logRequest("API restart called for: %s", name)
		if name != "" {
			// Try direct process name first
			if proc, found := s.procs.Get(name); found {
				s.logRequest("  Restarting process: %s", proc.Name)
				s.procs.Restart(proc.Name)
			} else if app, found := s.apps.Get(name); found && app.Type == config.AppTypeYAML {
				// Restart all services for multi-service app
				for i := range app.Services {
					svc := &app.Services[i]
					procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
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
			s.broadcastStatus()
		}
		w.WriteHeader(http.StatusOK)

	case "/api/start":
		name := r.URL.Query().Get("name")
		// Resolve alias to app name
		if app, found := s.apps.GetByNameOrAlias(name); found {
			name = app.Name
		}
		if name != "" {
			s.startByName(name)
			s.broadcastStatus()
		}
		w.WriteHeader(http.StatusOK)

	case "/api/logs":
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

	case "/api/server-logs":
		// Return roost-dev's request handling logs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.requestLog.Lines())

	case "/api/app-status":
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
			if idx := strings.Index(name, "-"); idx != -1 {
				serviceName := name[:idx]
				appName := name[idx+1:]
				if app, svc, found := s.apps.GetService(appName, serviceName); found {
					for _, depName := range svc.DependsOn {
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

	default:
		http.NotFound(w, r)
	}
}

type serviceStatus struct {
	Name     string `json:"name"`
	Running  bool   `json:"running"`
	Starting bool   `json:"starting,omitempty"`
	Failed   bool   `json:"failed,omitempty"`
	Error    string `json:"error,omitempty"`
	Port     int    `json:"port,omitempty"`
	Uptime   string `json:"uptime,omitempty"`
	Default  bool   `json:"default,omitempty"`
	URL      string `json:"url,omitempty"`
}

type appStatus struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Aliases     []string        `json:"aliases,omitempty"`
	Type        string          `json:"type"`
	URL         string          `json:"url"`
	Running     bool            `json:"running,omitempty"`
	Starting    bool            `json:"starting,omitempty"`
	Failed      bool            `json:"failed,omitempty"`
	Error       string          `json:"error,omitempty"`
	Port        int             `json:"port,omitempty"`
	Uptime      string          `json:"uptime,omitempty"`
	Services    []serviceStatus `json:"services,omitempty"`
}

// getStatus returns the current status of all apps as JSON
func (s *Server) getStatus() []byte {
	var status []appStatus

	// Build base URL with port if not 80
	baseURL := func(name string) string {
		if s.cfg.URLPort == 80 {
			return fmt.Sprintf("http://%s.%s", name, s.cfg.TLD)
		}
		return fmt.Sprintf("http://%s.%s:%d", name, s.cfg.TLD, s.cfg.URLPort)
	}

	for _, app := range s.apps.All() {
		if app.Hidden {
			continue
		}
		as := appStatus{
			Name:        app.Name,
			Description: app.Description,
			Aliases:     app.Aliases,
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
				} else if proc.IsStarting() {
					as.Starting = true
					as.Port = proc.Port
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
				ss := serviceStatus{Name: svc.Name, Default: svc.Default}
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				// Set service URL
				if app.Name == "roost-dev-tests" {
					ss.URL = fmt.Sprintf("http://%s.roost-dev.%s", svc.Name, s.cfg.TLD)
				} else if svc.Default {
					ss.URL = baseURL(app.Name)
				} else {
					ss.URL = baseURL(fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name))
				}
				if proc, found := s.procs.Get(procName); found {
					if proc.IsRunning() {
						ss.Running = true
						ss.Port = proc.Port
						ss.Uptime = proc.Uptime().Round(1e9).String()
					} else if proc.IsStarting() {
						ss.Starting = true
						ss.Port = proc.Port
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

	data, _ := json.Marshal(status)
	return data
}

// handleAPIStatus returns status of all apps and processes
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(s.getStatus())
}

// handleSSE handles Server-Sent Events connections for real-time updates
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to broadcasts
	ch := s.broadcaster.Subscribe()
	defer s.broadcaster.Unsubscribe(ch)

	// Send initial status immediately
	s.sendStatusToClient(w, flusher)

	// Listen for updates or client disconnect
	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// sendStatusToClient sends current status to a single SSE client
func (s *Server) sendStatusToClient(w http.ResponseWriter, flusher http.Flusher) {
	status := s.getStatusJSON()
	fmt.Fprintf(w, "data: %s\n\n", status)
	flusher.Flush()
}

// broadcastStatus sends current status to all connected SSE clients
func (s *Server) broadcastStatus() {
	status := s.getStatusJSON()
	s.broadcaster.Broadcast(status)
}

// getStatusJSON returns the current status as JSON bytes
func (s *Server) getStatusJSON() []byte {
	type serviceStatus struct {
		Name     string `json:"name"`
		Running  bool   `json:"running"`
		Starting bool   `json:"starting,omitempty"`
		Failed   bool   `json:"failed,omitempty"`
		Error    string `json:"error,omitempty"`
		Port     int    `json:"port,omitempty"`
		Uptime   string `json:"uptime,omitempty"`
		Default  bool   `json:"default,omitempty"`
		URL      string `json:"url,omitempty"`
	}

	type appStatus struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Aliases     []string        `json:"aliases,omitempty"`
		Type        string          `json:"type"`
		URL         string          `json:"url"`
		Running     bool            `json:"running,omitempty"`
		Starting    bool            `json:"starting,omitempty"`
		Failed      bool            `json:"failed,omitempty"`
		Error       string          `json:"error,omitempty"`
		Port        int             `json:"port,omitempty"`
		Uptime      string          `json:"uptime,omitempty"`
		Services    []serviceStatus `json:"services,omitempty"`
	}

	var status []appStatus

	baseURL := func(name string) string {
		if s.cfg.URLPort == 80 {
			return fmt.Sprintf("http://%s.%s", name, s.cfg.TLD)
		}
		return fmt.Sprintf("http://%s.%s:%d", name, s.cfg.TLD, s.cfg.URLPort)
	}

	for _, app := range s.apps.All() {
		if app.Hidden {
			continue
		}
		as := appStatus{
			Name:        app.Name,
			Description: app.Description,
			Aliases:     app.Aliases,
			URL:         baseURL(app.Name),
		}

		switch app.Type {
		case config.AppTypePort:
			as.Type = "port"
			as.Port = app.Port
			as.Running = true

		case config.AppTypeCommand:
			as.Type = "command"
			if proc, found := s.procs.Get(app.Name); found {
				if proc.IsRunning() {
					as.Running = true
					as.Port = proc.Port
					as.Uptime = proc.Uptime().Round(1e9).String()
				} else if proc.IsStarting() {
					as.Starting = true
					as.Port = proc.Port
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
			for _, svc := range app.Services {
				ss := serviceStatus{Name: svc.Name, Default: svc.Default}
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				// Set service URL
				if app.Name == "roost-dev-tests" {
					ss.URL = fmt.Sprintf("http://%s.roost-dev.%s", svc.Name, s.cfg.TLD)
				} else if svc.Default {
					ss.URL = baseURL(app.Name)
				} else {
					ss.URL = baseURL(fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name))
				}
				if proc, found := s.procs.Get(procName); found {
					if proc.IsRunning() {
						ss.Running = true
						ss.Port = proc.Port
						ss.Uptime = proc.Uptime().Round(1e9).String()
					} else if proc.IsStarting() {
						ss.Starting = true
						ss.Port = proc.Port
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

	data, _ := json.Marshal(status)
	return data
}
