package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/ollama"
	"github.com/panozzaj/roost-dev/internal/process"
)

// slugify converts a name to a URL-safe slug (lowercase, spaces to dashes)
func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
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
