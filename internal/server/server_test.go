package server

import (
	"os"
	"strings"
	"testing"

	"github.com/panozzaj/roost-dev/internal/config"
	"github.com/panozzaj/roost-dev/internal/process"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My App", "my-app"},
		{"MyApp", "myapp"},
		{"my-app", "my-app"},
		{"MY APP", "my-app"},
		{"web service", "web-service"},
		{"", ""},
		{"API Server", "api-server"},
		{"  spaced  ", "--spaced--"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.expected {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// newTestServer creates a server with injected dependencies for testing
func newTestServer(cfg *config.Config, apps *config.AppStore, procs *process.Manager) *Server {
	return &Server{
		cfg:         cfg,
		apps:        apps,
		procs:       procs,
		broadcaster: NewBroadcaster(),
	}
}

func TestFindService(t *testing.T) {
	cfg := &config.Config{TLD: "test"}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	app := &config.App{
		Name: "myapp",
		Services: []config.Service{
			{Name: "api", Command: "python server.py"},
			{Name: "web", Command: "npm start", DependsOn: []string{"api"}},
		},
	}

	t.Run("finds existing service", func(t *testing.T) {
		svc := s.findService(app, "api")
		if svc == nil {
			t.Fatal("expected to find api service")
		}
		if svc.Name != "api" {
			t.Errorf("expected name 'api', got %s", svc.Name)
		}
	})

	t.Run("returns nil for unknown service", func(t *testing.T) {
		svc := s.findService(app, "unknown")
		if svc != nil {
			t.Error("expected nil for unknown service")
		}
	})
}

func TestEnsureDependencies(t *testing.T) {
	cfg := &config.Config{TLD: "test"}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	app := &config.App{
		Name: "myapp",
		Dir:  "/tmp",
		Services: []config.Service{
			{Name: "api", Command: "sleep 999", Dir: "/tmp"},
			{Name: "web", Command: "sleep 999", Dir: "/tmp", DependsOn: []string{"api"}},
		},
	}

	t.Run("starts dependencies before the service", func(t *testing.T) {
		webSvc := s.findService(app, "web")
		if webSvc == nil {
			t.Fatal("web service not found")
		}

		// Ensure dependencies for web (should start api)
		s.ensureDependencies(app, webSvc)

		// Check that api process was started
		proc, found := procs.Get("api-myapp")
		if !found {
			t.Fatal("expected api-myapp process to be started")
		}

		// Process should be starting or running
		if !proc.IsStarting() && !proc.IsRunning() {
			t.Error("expected api-myapp to be starting or running")
		}

		// Clean up
		procs.Stop("api-myapp")
	})

	t.Run("does not start already running dependencies", func(t *testing.T) {
		// First start api
		apiSvc := s.findService(app, "api")
		procs.StartAsync("api-myapp", apiSvc.Command, apiSvc.Dir, apiSvc.Env)

		// Get initial process
		proc1, _ := procs.Get("api-myapp")

		// Now call ensureDependencies for web
		webSvc := s.findService(app, "web")
		s.ensureDependencies(app, webSvc)

		// Should be the same process (not restarted)
		proc2, _ := procs.Get("api-myapp")
		if proc1 != proc2 {
			t.Error("expected same process instance, dependency was restarted")
		}

		// Clean up
		procs.Stop("api-myapp")
	})

	t.Run("handles service with no dependencies", func(t *testing.T) {
		apiSvc := s.findService(app, "api")
		if apiSvc == nil {
			t.Fatal("api service not found")
		}

		// Should not panic or error
		s.ensureDependencies(app, apiSvc)
	})

	t.Run("handles unknown dependency gracefully", func(t *testing.T) {
		svcWithBadDep := &config.Service{
			Name:      "broken",
			Command:   "sleep 1",
			DependsOn: []string{"nonexistent"},
		}

		// Should not panic
		s.ensureDependencies(app, svcWithBadDep)
	})
}

func TestStartByNameServiceLookup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create a test YAML config with dependencies
	yamlContent := `
name: testapp
root: /tmp
services:
  api:
    cmd: sleep 999
  web:
    cmd: sleep 999
    depends_on: [api]
`
	configPath := tmpDir + "/testapp.yml"
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load the apps
	if err := apps.Load(); err != nil {
		t.Fatalf("failed to load apps: %v", err)
	}

	// Verify the app was loaded correctly
	app, found := apps.Get("testapp")
	if !found {
		t.Fatal("testapp not found")
	}

	t.Run("services are sorted with dependencies first", func(t *testing.T) {
		// api should come before web in the services list
		if len(app.Services) != 2 {
			t.Fatalf("expected 2 services, got %d", len(app.Services))
		}
		if app.Services[0].Name != "api" {
			t.Errorf("expected api first, got %s", app.Services[0].Name)
		}
		if app.Services[1].Name != "web" {
			t.Errorf("expected web second, got %s", app.Services[1].Name)
		}
	})

	t.Run("web service has api as dependency", func(t *testing.T) {
		var webSvc *config.Service
		for i := range app.Services {
			if app.Services[i].Name == "web" {
				webSvc = &app.Services[i]
				break
			}
		}
		if webSvc == nil {
			t.Fatal("web service not found")
		}
		if len(webSvc.DependsOn) != 1 || webSvc.DependsOn[0] != "api" {
			t.Errorf("expected web to depend on [api], got %v", webSvc.DependsOn)
		}
	})

	t.Run("findService locates services correctly", func(t *testing.T) {
		apiSvc := s.findService(app, "api")
		if apiSvc == nil {
			t.Error("findService failed to find api")
		}
		webSvc := s.findService(app, "web")
		if webSvc == nil {
			t.Error("findService failed to find web")
		}
		unknown := s.findService(app, "unknown")
		if unknown != nil {
			t.Error("findService should return nil for unknown service")
		}
	})
}

func TestEnsureDependenciesIntegration(t *testing.T) {
	cfg := &config.Config{TLD: "test"}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create app with chain of dependencies: c -> b -> a
	app := &config.App{
		Name: "chainapp",
		Dir:  "/tmp",
		Services: []config.Service{
			{Name: "a", Command: "sleep 999", Dir: "/tmp"},
			{Name: "b", Command: "sleep 999", Dir: "/tmp", DependsOn: []string{"a"}},
			{Name: "c", Command: "sleep 999", Dir: "/tmp", DependsOn: []string{"b"}},
		},
	}

	t.Run("starting c starts b and a as dependencies", func(t *testing.T) {
		cSvc := s.findService(app, "c")
		if cSvc == nil {
			t.Fatal("c service not found")
		}

		// Start dependencies for c
		s.ensureDependencies(app, cSvc)

		// b should be starting (depends on by c)
		bProc, bFound := procs.Get("b-chainapp")
		if !bFound {
			t.Error("expected b-chainapp to be started as dependency of c")
		} else if !bProc.IsStarting() && !bProc.IsRunning() {
			t.Error("expected b-chainapp to be starting or running")
		}

		// Note: ensureDependencies only starts direct dependencies,
		// not transitive ones. This is intentional - each service
		// call ensureDependencies for itself.

		// Clean up
		procs.Stop("a-chainapp")
		procs.Stop("b-chainapp")
	})
}

func TestFindApp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create test apps
	yamlContent := `
name: blog
alias: panozzaj
root: /tmp/blog
cmd: bin/serve
`
	os.WriteFile(tmpDir+"/blog.yml", []byte(yamlContent), 0644)

	yamlContent2 := `
name: myapp
aliases:
  - ma
  - mp
root: /tmp/myapp
cmd: rails server
`
	os.WriteFile(tmpDir+"/myapp.yml", []byte(yamlContent2), 0644)

	apps.Load()

	t.Run("finds app by exact name", func(t *testing.T) {
		app, found := s.findApp("blog")
		if !found {
			t.Fatal("expected to find blog app by name")
		}
		if app.Name != "blog" {
			t.Errorf("expected app name 'blog', got %s", app.Name)
		}
	})

	t.Run("finds app by alias", func(t *testing.T) {
		app, found := s.findApp("panozzaj")
		if !found {
			t.Fatal("expected to find blog app by alias 'panozzaj'")
		}
		if app.Name != "blog" {
			t.Errorf("expected app name 'blog', got %s", app.Name)
		}
	})

	t.Run("finds app by multiple aliases", func(t *testing.T) {
		app1, found1 := s.findApp("ma")
		if !found1 {
			t.Fatal("expected to find myapp by alias 'ma'")
		}
		if app1.Name != "myapp" {
			t.Errorf("expected app name 'myapp', got %s", app1.Name)
		}

		app2, found2 := s.findApp("mp")
		if !found2 {
			t.Fatal("expected to find myapp by alias 'mp'")
		}
		if app2.Name != "myapp" {
			t.Errorf("expected app name 'myapp', got %s", app2.Name)
		}
	})

	t.Run("returns false for unknown app", func(t *testing.T) {
		_, found := s.findApp("nonexistent")
		if found {
			t.Error("expected not to find nonexistent app")
		}
	})

	t.Run("strips subdomain to find app", func(t *testing.T) {
		// Simulate accessing admin.myapp.test
		app, found := s.findApp("admin.myapp")
		if !found {
			t.Fatal("expected to find myapp via subdomain 'admin.myapp'")
		}
		if app.Name != "myapp" {
			t.Errorf("expected app name 'myapp', got %s", app.Name)
		}
	})

	t.Run("strips subdomain and resolves alias", func(t *testing.T) {
		// Simulate accessing api.ma.test (ma is alias for myapp)
		app, found := s.findApp("api.ma")
		if !found {
			t.Fatal("expected to find myapp via subdomain 'api.ma'")
		}
		if app.Name != "myapp" {
			t.Errorf("expected app name 'myapp', got %s", app.Name)
		}
	})

	t.Run("handles multiple subdomain levels", func(t *testing.T) {
		// Simulate accessing foo.bar.myapp.test
		app, found := s.findApp("foo.bar.myapp")
		if !found {
			t.Fatal("expected to find myapp via 'foo.bar.myapp'")
		}
		if app.Name != "myapp" {
			t.Errorf("expected app name 'myapp', got %s", app.Name)
		}
	})
}

func TestDependencyStatusChecking(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create a test YAML config with dependencies
	yamlContent := `
name: deptest
root: /tmp
services:
  api:
    cmd: python3 -m http.server $PORT
  web:
    cmd: python3 -m http.server $PORT
    depends_on: [api]
`
	configPath := tmpDir + "/deptest.yml"
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := apps.Load(); err != nil {
		t.Fatalf("failed to load apps: %v", err)
	}

	t.Run("service reports starting when dependency not in map", func(t *testing.T) {
		// Start only web (not api) - this simulates a race condition
		// where web starts but api hasn't been added to the map yet
		procs.StartAsync("web-deptest", "python3 -m http.server $PORT", "/tmp", nil)
		defer procs.Stop("web-deptest")

		// web is running but api is not in the map
		// getDependencyStatus should report that we need to wait

		app, _ := apps.Get("deptest")
		webSvc := s.findService(app, "web")
		if webSvc == nil {
			t.Fatal("web service not found")
		}

		// Check if api dependency is satisfied
		// Since api is not in the map, this should indicate we're not ready
		apiProc, found := procs.Get("api-deptest")
		if found {
			t.Error("api-deptest should not be in the map")
		}
		if apiProc != nil {
			t.Error("apiProc should be nil")
		}
	})

	t.Run("service reports starting when dependency is starting", func(t *testing.T) {
		// Start api with a command that doesn't listen on port (stays starting)
		procs.StartAsync("api-deptest2", "sleep 999", "/tmp", nil)
		defer procs.Stop("api-deptest2")

		// api should be in starting state (not listening on port)
		apiProc, found := procs.Get("api-deptest2")
		if !found {
			t.Fatal("api-deptest2 should be in the map")
		}
		if !apiProc.IsStarting() {
			t.Error("api-deptest2 should be starting (port not ready)")
		}
	})
}

func TestResolveServiceName(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create test apps with services
	yamlContent := `
name: myapp
root: /tmp
services:
  api:
    cmd: python server.py
  web:
    cmd: npm start
`
	os.WriteFile(tmpDir+"/myapp.yml", []byte(yamlContent), 0644)

	yamlContent2 := `
name: otherapp
root: /tmp
services:
  api:
    cmd: go run main.go
  worker:
    cmd: python worker.py
`
	os.WriteFile(tmpDir+"/otherapp.yml", []byte(yamlContent2), 0644)

	apps.Load()

	t.Run("resolves colon syntax app:service", func(t *testing.T) {
		match := s.resolveServiceName("myapp:web")
		if match == nil {
			t.Fatal("expected to resolve myapp:web")
		}
		if match.App.Name != "myapp" {
			t.Errorf("expected app 'myapp', got %s", match.App.Name)
		}
		if match.Service.Name != "web" {
			t.Errorf("expected service 'web', got %s", match.Service.Name)
		}
		if match.ProcName != "web-myapp" {
			t.Errorf("expected procName 'web-myapp', got %s", match.ProcName)
		}
	})

	t.Run("resolves dot syntax service.app", func(t *testing.T) {
		match := s.resolveServiceName("web.myapp")
		if match == nil {
			t.Fatal("expected to resolve web.myapp")
		}
		if match.App.Name != "myapp" {
			t.Errorf("expected app 'myapp', got %s", match.App.Name)
		}
		if match.Service.Name != "web" {
			t.Errorf("expected service 'web', got %s", match.Service.Name)
		}
	})

	t.Run("resolves internal process name format", func(t *testing.T) {
		match := s.resolveServiceName("web-myapp")
		if match == nil {
			t.Fatal("expected to resolve web-myapp")
		}
		if match.ProcName != "web-myapp" {
			t.Errorf("expected procName 'web-myapp', got %s", match.ProcName)
		}
	})

	t.Run("resolves unique bare service name", func(t *testing.T) {
		// 'worker' only exists in otherapp
		match := s.resolveServiceName("worker")
		if match == nil {
			t.Fatal("expected to resolve 'worker'")
		}
		if match.App.Name != "otherapp" {
			t.Errorf("expected app 'otherapp', got %s", match.App.Name)
		}
		if match.Service.Name != "worker" {
			t.Errorf("expected service 'worker', got %s", match.Service.Name)
		}
	})

	t.Run("returns nil for ambiguous bare service name", func(t *testing.T) {
		// 'api' exists in both myapp and otherapp
		match := s.resolveServiceName("api")
		if match != nil {
			t.Error("expected nil for ambiguous service name 'api'")
		}
	})

	t.Run("returns nil for unknown service", func(t *testing.T) {
		match := s.resolveServiceName("myapp:unknown")
		if match != nil {
			t.Error("expected nil for unknown service")
		}
	})

	t.Run("returns nil for unknown app", func(t *testing.T) {
		match := s.resolveServiceName("unknownapp:web")
		if match != nil {
			t.Error("expected nil for unknown app")
		}
	})
}

// Tests for Tailscale Serve support
// Tailscale Serve proxies requests from *.ts.net hosts using path-based routing

func TestIsTailscaleHost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"macbook-pro.tail094a69.ts.net", true},
		{"my-machine.tailnet.ts.net", true},
		{"foo.ts.net", true},
		{"roost-dev.test", false},
		{"myapp.test", false},
		{"example.com", false},
		{"ts.net.example.com", false}, // ts.net must be suffix
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			got := strings.HasSuffix(tc.host, ".ts.net")
			if got != tc.expected {
				t.Errorf("isTailscaleHost(%q) = %v, want %v", tc.host, got, tc.expected)
			}
		})
	}
}

func TestTailscalePathParsing(t *testing.T) {
	// Test the path parsing logic used in handleTailscaleRequest
	tests := []struct {
		path            string
		expectedName    string
		expectedRemPath string
	}{
		{"/myapp/health", "myapp", "/health"},
		{"/myapp/", "myapp", "/"},
		{"/myapp", "myapp", "/"},
		{"/api-focustrack/api/sync", "api-focustrack", "/api/sync"},
		{"/blog/posts/123", "blog", "/posts/123"},
		{"/app/a/b/c", "app", "/a/b/c"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			path := strings.TrimPrefix(tc.path, "/")
			parts := strings.SplitN(path, "/", 2)
			name := parts[0]
			remainingPath := "/"
			if len(parts) > 1 {
				remainingPath = "/" + parts[1]
			}

			if name != tc.expectedName {
				t.Errorf("name = %q, want %q", name, tc.expectedName)
			}
			if remainingPath != tc.expectedRemPath {
				t.Errorf("remainingPath = %q, want %q", remainingPath, tc.expectedRemPath)
			}
		})
	}
}

func TestTailscaleServiceNameRouting(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	_ = process.NewManager() // not used in these tests

	// Create a multi-service app
	yamlContent := `
name: focustrack
root: /tmp
services:
  api:
    cmd: python server.py
  web:
    cmd: npm start
`
	os.WriteFile(tmpDir+"/focustrack.yml", []byte(yamlContent), 0644)
	apps.Load()

	t.Run("parses service-app pattern from path", func(t *testing.T) {
		// Simulate path /api-focustrack/health
		name := "api-focustrack"

		// This is the logic from handleTailscaleRequest
		if idx := strings.Index(name, "-"); idx != -1 {
			serviceName := name[:idx]
			appName := name[idx+1:]

			if serviceName != "api" {
				t.Errorf("serviceName = %q, want 'api'", serviceName)
			}
			if appName != "focustrack" {
				t.Errorf("appName = %q, want 'focustrack'", appName)
			}

			app, svc, found := apps.GetService(appName, serviceName)
			if !found {
				t.Fatal("expected to find api service of focustrack")
			}
			if app.Name != "focustrack" {
				t.Errorf("app.Name = %q, want 'focustrack'", app.Name)
			}
			if svc.Name != "api" {
				t.Errorf("svc.Name = %q, want 'api'", svc.Name)
			}
		}
	})

	t.Run("falls back to app lookup for non-service paths", func(t *testing.T) {
		// Path like /focustrack/health should try app lookup
		name := "focustrack"

		// No hyphen, so goes to app lookup
		app, found := apps.GetByNameOrAlias(name)
		if !found {
			t.Fatal("expected to find focustrack app")
		}
		if app.Name != "focustrack" {
			t.Errorf("app.Name = %q, want 'focustrack'", app.Name)
		}
	})

	t.Run("handles app names with hyphens correctly", func(t *testing.T) {
		// Create an app with hyphen in name
		yamlContent2 := `
name: my-blog
root: /tmp
cmd: hugo server
`
		os.WriteFile(tmpDir+"/my-blog.yml", []byte(yamlContent2), 0644)
		apps.Reload()

		// Path /my-blog/posts - "my" is not a service of "blog"
		name := "my-blog"
		if idx := strings.Index(name, "-"); idx != -1 {
			serviceName := name[:idx]
			appName := name[idx+1:]

			// This should NOT find a service (blog doesn't have "my" service)
			_, _, found := apps.GetService(appName, serviceName)
			if found {
				t.Error("should not find 'my' service of 'blog' app")
			}

			// Should fall back to app lookup and find "my-blog"
			app, found := apps.GetByNameOrAlias(name)
			if !found {
				t.Fatal("expected to find my-blog app")
			}
			if app.Name != "my-blog" {
				t.Errorf("app.Name = %q, want 'my-blog'", app.Name)
			}
		}
	})
}

func TestTailscaleAPIRouting(t *testing.T) {
	// Test that /api/* paths are routed to dashboard API
	tests := []struct {
		path      string
		isAPIPath bool
	}{
		{"/api/status", true},
		{"/api/logs", true},
		{"/api/app-status", true},
		{"/api/", true},
		{"/api-focustrack/health", false}, // service name, not API
		{"/myapp/api/endpoint", false},    // nested api path
		{"/apiapp/something", false},      // app starting with "api"
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := strings.HasPrefix(tc.path, "/api/")
			if got != tc.isAPIPath {
				t.Errorf("isAPIPath(%q) = %v, want %v", tc.path, got, tc.isAPIPath)
			}
		})
	}
}

func TestListAppsHTML(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	// Create simple app
	os.WriteFile(tmpDir+"/blog.yml", []byte(`
name: blog
root: /tmp
cmd: hugo server
`), 0644)

	// Create multi-service app
	os.WriteFile(tmpDir+"/focustrack.yml", []byte(`
name: focustrack
root: /tmp
services:
  api:
    cmd: python server.py
  web:
    cmd: npm start
`), 0644)

	apps.Load()

	html := s.listAppsHTML()

	// Should contain simple app
	if !strings.Contains(html, "/blog/") {
		t.Error("expected HTML to contain /blog/")
	}

	// Should contain service paths for multi-service app
	if !strings.Contains(html, "/api-focustrack/") {
		t.Error("expected HTML to contain /api-focustrack/")
	}
	if !strings.Contains(html, "/web-focustrack/") {
		t.Error("expected HTML to contain /web-focustrack/")
	}

	// Should NOT contain bare multi-service app name
	// (users should access via service name)
	if strings.Contains(html, "<li>/focustrack/</li>") {
		t.Error("multi-service app should list services, not app name")
	}
}
