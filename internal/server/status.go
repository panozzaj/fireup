package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/panozzaj/roost-dev/internal/config"
)

// serviceStatus represents the status of a single service
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

// appStatus represents the status of an app
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
	// Reuse getStatus since it already does what we need
	return s.getStatus()
}
