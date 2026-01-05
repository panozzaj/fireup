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

// Server is the main roost-dev server
type Server struct {
	cfg      *config.Config
	apps     *config.AppStore
	procs    *process.Manager
	httpSrv  *http.Server
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

	addr := fmt.Sprintf(":%d", s.cfg.HTTPPort)
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
		http.Error(w, fmt.Sprintf("Invalid host: %s (expected *.%s)", host, s.cfg.TLD), http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("App not found: %s\n\nCreate a config file at: %s/%s", name, s.cfg.Dir, name), http.StatusNotFound)
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
		// Start process if needed, then proxy
		proc, err := s.ensureProcess(app.Name, app.Command, app.Dir, app.Env)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start process: %v", err), http.StatusInternalServerError)
			return
		}
		proxy.NewReverseProxy(proc.Port).ServeHTTP(w, r)

	case config.AppTypeStatic:
		// Serve static files
		proxy.NewStaticHandler(app.FilePath).ServeHTTP(w, r)

	case config.AppTypeYAML:
		// Multi-service app - default to first service or show list
		if len(app.Services) == 1 {
			s.handleService(w, r, app, &app.Services[0])
			return
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

// handleService handles a request for a service within a multi-service app
func (s *Server) handleService(w http.ResponseWriter, r *http.Request, app *config.App, svc *config.Service) {
	procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)

	proc, err := s.ensureProcess(procName, svc.Command, svc.Dir, svc.Env)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start service: %v", err), http.StatusInternalServerError)
		return
	}

	proxy.NewReverseProxy(proc.Port).ServeHTTP(w, r)
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

// handleDashboard serves the web UI
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
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
			s.procs.Stop(name)
		}
		w.WriteHeader(http.StatusOK)

	case "/api/restart":
		name := r.URL.Query().Get("name")
		if name != "" {
			proc, found := s.procs.Get(name)
			if found {
				s.procs.Restart(proc.Name)
			}
		}
		w.WriteHeader(http.StatusOK)

	case "/api/logs":
		name := r.URL.Query().Get("name")
		proc, found := s.procs.Get(name)
		if !found {
			http.Error(w, "Process not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(proc.Logs().Lines())

	default:
		http.NotFound(w, r)
	}
}

// handleAPIStatus returns status of all apps and processes
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	type serviceStatus struct {
		Name    string `json:"name"`
		Running bool   `json:"running"`
		Port    int    `json:"port,omitempty"`
		Uptime  string `json:"uptime,omitempty"`
	}

	type appStatus struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Type        string          `json:"type"`
		URL         string          `json:"url"`
		Running     bool            `json:"running,omitempty"`
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
			if proc, found := s.procs.Get(app.Name); found && proc.IsRunning() {
				as.Running = true
				as.Port = proc.Port
				as.Uptime = proc.Uptime().Round(1e9).String()
			}

		case config.AppTypeStatic:
			as.Type = "static"
			as.Running = true

		case config.AppTypeYAML:
			as.Type = "multi-service"
			for _, svc := range app.Services {
				ss := serviceStatus{Name: svc.Name}
				procName := fmt.Sprintf("%s-%s", svc.Name, app.Name)
				if proc, found := s.procs.Get(procName); found && proc.IsRunning() {
					ss.Running = true
					ss.Port = proc.Port
					ss.Uptime = proc.Uptime().Round(1e9).String()
				}
				as.Services = append(as.Services, ss)
			}
		}

		status = append(status, as)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
