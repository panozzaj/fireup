package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/panozzaj/roost-dev/internal/certs"
	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/ollama"
	"github.com/panozzaj/roost-dev/internal/process"
)

// slugify converts a name to a URL-safe slug (lowercase, spaces to dashes)
func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// collectProcessNames returns a set of all process names for currently loaded apps.
// For simple command apps, this is the app name.
// For multi-service apps, this is "{service-name}-{app-name}" for each service.
func (s *Server) collectProcessNames() map[string]bool {
	names := make(map[string]bool)
	for _, app := range s.apps.All() {
		switch app.Type {
		case config.AppTypeCommand:
			names[app.Name] = true
		case config.AppTypeYAML:
			for _, svc := range app.Services {
				procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
				names[procName] = true
			}
		}
	}
	return names
}

// Server is the main roost-dev server
type Server struct {
	cfg           *config.Config
	apps          *config.AppStore
	procs         *process.Manager
	httpSrv       *http.Server
	httpsSrv      *http.Server       // HTTPS server (optional, only if certs exist)
	requestLog    *process.LogBuffer // Reuse LogBuffer for request logging
	broadcaster   *Broadcaster       // SSE broadcaster for real-time updates
	configWatcher *config.Watcher    // Watches config directory for changes
	ollamaClient  *ollama.Client     // Optional LLM client for log analysis
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

	// Initialize Ollama client if configured
	if cfg.Ollama != nil && cfg.Ollama.Enabled {
		s.ollamaClient = ollama.New(cfg.Ollama.URL, cfg.Ollama.Model)
		fmt.Printf("Ollama error analysis enabled (model: %s)\n", cfg.Ollama.Model)
	}

	// Set up config watcher
	watcher, err := config.NewWatcher(cfg.Dir, func(changedFiles []string) {
		// Build a set of app names whose configs changed
		changedApps := make(map[string]bool)
		for _, filename := range changedFiles {
			// Strip .yml/.yaml extension to get app name
			appName := strings.TrimSuffix(filename, ".yml")
			appName = strings.TrimSuffix(appName, ".yaml")
			changedApps[appName] = true
		}

		// Collect process names for current apps before reload
		oldProcessNames := s.collectProcessNames()

		// Track which apps were running before reload
		runningApps := make(map[string]bool)
		for _, app := range s.apps.All() {
			if changedApps[app.Name] {
				switch app.Type {
				case config.AppTypeCommand:
					if proc, found := s.procs.Get(app.Name); found && (proc.IsRunning() || proc.IsStarting()) {
						runningApps[app.Name] = true
					}
				case config.AppTypeYAML:
					// Check if any service is running or starting (starting might be hung)
					for _, svc := range app.Services {
						procName := fmt.Sprintf("%s-%s", slugify(svc.Name), app.Name)
						if proc, found := s.procs.Get(procName); found && (proc.IsRunning() || proc.IsStarting()) {
							runningApps[app.Name] = true
							break
						}
					}
				}
			}
		}

		if err := s.apps.Reload(); err != nil {
			s.logRequest("Config reload error: %v", err)
			return
		}

		// Collect process names for apps after reload
		newProcessNames := s.collectProcessNames()

		// Stop processes for removed apps
		for name := range oldProcessNames {
			if _, exists := newProcessNames[name]; !exists {
				if proc, found := s.procs.Get(name); found && proc.IsRunning() {
					s.logRequest("Stopping orphaned process: %s", name)
					s.procs.Stop(name)
				}
			}
		}

		// Restart apps that were running and whose config changed
		for appName := range runningApps {
			app, exists := s.apps.Get(appName)
			if !exists {
				continue // App was removed
			}
			s.logRequest("Restarting %s (config changed)", appName)
			switch app.Type {
			case config.AppTypeCommand:
				s.procs.Stop(appName)
				s.procs.StartAsync(appName, app.Command, app.Dir, app.Env)
			case config.AppTypeYAML:
				// Restart all services for this app
				for _, svc := range app.Services {
					procName := fmt.Sprintf("%s-%s", slugify(svc.Name), appName)
					s.procs.Stop(procName)
					s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
				}
			}
		}

		s.logRequest("Config reloaded")
		s.broadcastStatus()
	})
	if err != nil {
		// Log but don't fail - config watching is optional
		fmt.Printf("Warning: could not watch config directory: %v\n", err)
	} else {
		s.configWatcher = watcher
	}

	return s, nil
}

// getCertsDir returns the path to the certs directory
func (s *Server) getCertsDir() string {
	return filepath.Join(s.cfg.Dir, "certs")
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

	// Start HTTPS servers if CA exists (dynamic cert generation)
	certsDir := s.getCertsDir()
	if certs.CAExists(certsDir) {
		certManager, err := certs.NewManager(certsDir, s.cfg.TLD)
		if err != nil {
			fmt.Printf("Warning: failed to load CA: %v\n", err)
		} else {
			go s.startHTTPS(mux, certManager, "127.0.0.1")
			go s.startHTTPS(mux, certManager, "[::1]")
		}
	}

	// Start HTTP on IPv6 as well
	go s.startHTTPv6(mux)

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.HTTPPort)
	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s.httpSrv.ListenAndServe()
}

// startHTTPv6 starts an HTTP server on IPv6 localhost
func (s *Server) startHTTPv6(handler http.Handler) {
	addr := fmt.Sprintf("[::1]:%d", s.cfg.HTTPPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		// IPv6 might not be available, that's OK
	}
}

// startHTTPS starts the HTTPS server with dynamic certificate generation
func (s *Server) startHTTPS(handler http.Handler, certManager *certs.Manager, host string) {
	addr := fmt.Sprintf("%s:%d", host, s.cfg.HTTPSPort)

	srv := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: certManager.TLSConfig(),
	}

	// Only store the IPv4 server for shutdown and logging
	if host == "127.0.0.1" {
		s.httpsSrv = srv
		fmt.Printf("HTTPS listening on https://%s:%d (dynamic certs)\n", host, s.cfg.HTTPSPort)
	}

	if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		// IPv6 might not be available, that's OK
		if host != "[::1]" {
			fmt.Printf("HTTPS server error: %v\n", err)
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	fmt.Println("[roost-dev] Shutdown: stopping config watcher...")
	if s.configWatcher != nil {
		s.configWatcher.Stop()
	}
	fmt.Println("[roost-dev] Shutdown: stopping all processes...")
	s.procs.StopAll()
	fmt.Println("[roost-dev] Shutdown: closing HTTP servers...")
	if s.httpSrv != nil {
		s.httpSrv.Close()
	}
	if s.httpsSrv != nil {
		s.httpsSrv.Close()
	}
	fmt.Println("[roost-dev] Shutdown: done")
}

// logRequest logs a request handling event
func (s *Server) logRequest(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	s.requestLog.Write([]byte(fmt.Sprintf("[%s] %s\n", timestamp, msg)))
	fmt.Printf("[%s] %s\n", timestamp, msg) // Also print to stdout
}

// getTheme reads the theme from config-theme.json, defaults to "system"
func (s *Server) getTheme() string {
	data, err := os.ReadFile(filepath.Join(s.cfg.Dir, "config-theme.json"))
	if err != nil {
		return "system"
	}
	var cfg struct {
		Theme string `json:"theme"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "system"
	}
	if cfg.Theme == "light" || cfg.Theme == "dark" || cfg.Theme == "system" {
		return cfg.Theme
	}
	return "system"
}

// setTheme writes the theme to config-theme.json
func (s *Server) setTheme(theme string) error {
	if theme != "light" && theme != "dark" && theme != "system" {
		return fmt.Errorf("invalid theme: %s", theme)
	}
	data, _ := json.Marshal(map[string]string{"theme": theme})
	return os.WriteFile(filepath.Join(s.cfg.Dir, "config-theme.json"), data, 0644)
}
