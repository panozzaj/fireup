package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/process"
	"github.com/panozzaj/roost-dev/internal/proxy"
	"github.com/panozzaj/roost-dev/internal/server/pages"
)

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

	// Built-in welcome page at roost-test.<tld>
	if host == "roost-test."+s.cfg.TLD || host == "roost-test" {
		s.handleWelcome(w, r)
		return
	}
	if strings.HasSuffix(host, ".roost-dev."+s.cfg.TLD) {
		// Subdomain of roost-dev.test → route to roost-dev-tests services
		subdomain := strings.TrimSuffix(host, ".roost-dev."+s.cfg.TLD)
		if app, svc, found := s.apps.GetService("roost-dev-tests", subdomain); found {
			s.handleService(w, r, app, svc)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, pages.Error(
			"Service not found",
			fmt.Sprintf("No service named '%s' in roost-dev-tests", subdomain),
			`<p class="hint">Check available services at <a href="http://roost-dev.test">roost-dev.test</a></p>`,
			s.cfg.TLD, s.getTheme()))
		return
	}

	// Parse hostname: [service-]appname.tld
	if !strings.HasSuffix(host, "."+s.cfg.TLD) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, pages.Error(
			"Invalid host",
			fmt.Sprintf("Expected *.%s, got %s", s.cfg.TLD, host),
			"",
			s.cfg.TLD, s.getTheme()))
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, pages.Error(
			"App not found",
			fmt.Sprintf("No app configured for '%s'", name),
			fmt.Sprintf(`<p class="hint">Create config at: %s/%s.yml</p>`, html.EscapeString(s.cfg.Dir), html.EscapeString(name)),
			s.cfg.TLD, s.getTheme()))
		return
	}

	s.handleApp(w, r, app)
}

// findApp tries to find an app by progressively shorter names
// e.g., "admin.mateams" → try "admin.mateams", then "mateams"
func (s *Server) findApp(name string) (*config.App, bool) {
	// Try exact match or alias first
	if app, found := s.apps.GetByNameOrAlias(name); found {
		return app, true
	}

	// Try progressively shorter names (strip leading subdomain)
	for {
		idx := strings.Index(name, ".")
		if idx == -1 {
			break
		}
		name = name[idx+1:]
		if app, found := s.apps.GetByNameOrAlias(name); found {
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
		proxy.NewReverseProxy(app.Port, s.getTheme()).ServeHTTP(w, r)

	case config.AppTypeCommand:
		// Check process status and serve appropriately
		proc, found := s.procs.Get(app.Name)
		if found && proc.IsRunning() {
			// Already running - proxy directly
			proxy.NewReverseProxy(proc.Port, s.getTheme()).ServeHTTP(w, r)
			return
		}
		if found && proc.HasFailed() {
			// Failed - show interstitial with error
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Write([]byte(pages.Interstitial(app.Name, s.cfg.TLD, s.getTheme(), true, proc.ExitError())))
			return
		}
		if found && proc.IsStarting() {
			// Starting - show interstitial
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Write([]byte(pages.Interstitial(app.Name, s.cfg.TLD, s.getTheme(), false, "")))
			return
		}
		// Idle - start async and show interstitial
		_, err := s.procs.StartAsync(app.Name, app.Command, app.Dir, app.Env)
		if err != nil {
			// Immediate failure (e.g., directory doesn't exist)
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Write([]byte(pages.Interstitial(app.Name, s.cfg.TLD, s.getTheme(), true, err.Error())))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(pages.Interstitial(app.Name, s.cfg.TLD, s.getTheme(), false, "")))

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
		proxy.NewReverseProxy(proc.Port, s.getTheme()).ServeHTTP(w, r)
		return
	}
	if found && proc.HasFailed() {
		// Failed - show interstitial with error
		s.logRequest("  -> INTERSTITIAL (failed)")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(pages.Interstitial(procName, s.cfg.TLD, s.getTheme(), true, proc.ExitError())))
		return
	}
	if found && proc.IsStarting() {
		// Starting - show interstitial
		s.logRequest("  -> INTERSTITIAL (starting)")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(pages.Interstitial(procName, s.cfg.TLD, s.getTheme(), false, "")))
		return
	}
	// Idle - start async and show interstitial
	s.logRequest("  -> INTERSTITIAL (idle, starting %s)", procName)
	_, err := s.procs.StartAsync(procName, svc.Command, svc.Dir, svc.Env)
	if err != nil {
		// Immediate failure (e.g., directory doesn't exist)
		s.logRequest("  -> FAILED to start: %v", err)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Write([]byte(pages.Interstitial(procName, s.cfg.TLD, s.getTheme(), true, err.Error())))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Write([]byte(pages.Interstitial(procName, s.cfg.TLD, s.getTheme(), false, "")))
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
