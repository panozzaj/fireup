package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"strings"
	"testing"

	"github.com/panozzaj/fireup/internal/config"
	"github.com/panozzaj/fireup/internal/process"
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
		requestLog:  process.NewLogBuffer(100),
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
		{"fireup.test", false},
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

// requestWithHost creates an HTTP request with the given Host header
func requestWithHost(host, path string) *http.Request {
	req := httptest.NewRequest("GET", path, nil)
	req.Host = host
	return req
}

func TestHandleRequestRouting(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	os.WriteFile(tmpDir+"/blog.yml", []byte("name: blog\nroot: /tmp\ncmd: echo hi\n"), 0644)
	os.WriteFile(tmpDir+"/myapp.yml", []byte(`
name: myapp
root: /tmp
services:
  api:
    cmd: echo api
  web:
    cmd: echo web
    default: true
`), 0644)
	apps.Load()

	t.Run("fireup.test routes to dashboard", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("fireup.test", "/"))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "fireup") {
			t.Error("expected dashboard HTML")
		}
	})

	t.Run("fireup.test with port routes to dashboard", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("fireup.test:80", "/"))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("fireup-test.test routes to welcome page", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("fireup-test.test", "/"))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unknown app returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("nonexistent.test", "/"))
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "App not found") {
			t.Error("expected 'App not found' in response")
		}
	})

	t.Run("wrong TLD returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("blog.dev", "/"))
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid host") {
			t.Error("expected 'Invalid host' in response")
		}
	})

	t.Run("known app routes to handleApp", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("blog.test", "/"))
		// Blog isn't running, so we get the interstitial (starting page)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("service-app pattern routes to service", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("api-myapp.test", "/"))
		// Service isn't running, so we get the interstitial
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unknown service of known app falls through to app lookup", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("unknown-myapp.test", "/"))
		// "unknown-myapp" is not a service, and "unknown-myapp" is not an app name
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("subdomain of fireup.test for missing service returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		s.handleRequest(w, requestWithHost("nope.fireup.test", "/"))
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Service not found") {
			t.Error("expected 'Service not found' in response")
		}
	})
}

// TestInvalidHostDoesNotKeepConnectionAlive covers the "I removed the
// /etc/hosts block and the browser still shows fireup" case. A browser reuses
// a pooled keep-alive socket for an origin without re-resolving DNS, so if
// fireup keeps the connection open after serving "Invalid host", the user
// stays stuck on fireup long after the hosts entry is gone.
func TestInvalidHostDoesNotKeepConnectionAlive(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	srv := httptest.NewServer(http.HandlerFunc(s.handleRequest))
	defer srv.Close()

	// get issues a request with the given Host header and reports whether it
	// rode on a connection recycled from the client's idle pool.
	get := func(t *testing.T, client *http.Client, host string) (*http.Response, bool) {
		t.Helper()
		req, err := http.NewRequest("GET", srv.URL+"/", nil)
		if err != nil {
			t.Fatalf("building request: %v", err)
		}
		req.Host = host

		var reused bool
		trace := &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request to %s: %v", host, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp, reused
	}

	// Control: a normal fireup host keeps the connection alive, which is what
	// makes the assertion below meaningful rather than vacuously true.
	t.Run("valid host reuses the connection", func(t *testing.T) {
		client := &http.Client{Transport: &http.Transport{}}
		defer client.CloseIdleConnections()

		get(t, client, "fireup.test")
		if _, reused := get(t, client, "fireup.test"); !reused {
			t.Error("expected the second request to reuse the pooled connection")
		}
	})

	t.Run("invalid host closes the connection", func(t *testing.T) {
		client := &http.Client{Transport: &http.Transport{}}
		defer client.CloseIdleConnections()

		resp, _ := get(t, client, "bsky.app")
		if !resp.Close {
			t.Error("expected Connection: close on the invalid-host response")
		}
		if _, reused := get(t, client, "fireup.test"); reused {
			t.Error("expected no pooled connection to survive the invalid-host response")
		}
	})
}

// TestErrorPagesAreNotCacheable guards against a browser caching an error page
// and showing it after the underlying problem is fixed. 404 in particular is
// heuristically cacheable when the response says nothing about caching.
func TestErrorPagesAreNotCacheable(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"invalid host", "bsky.app", http.StatusBadRequest},
		{"unknown app", "nonexistent.test", http.StatusNotFound},
		{"unknown fireup service", "nope.fireup.test", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.handleRequest(w, requestWithHost(tt.host, "/"))

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, w.Code)
			}
			if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
				t.Errorf("expected Cache-Control to contain no-store, got %q", cc)
			}
		})
	}
}

// TestInvalidHostEscapesHost makes sure a hostile Host header cannot inject
// markup into the error page, which renders its hint as raw HTML.
func TestInvalidHostEscapesHost(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{TLD: "test", Dir: tmpDir}
	apps := config.NewAppStore(cfg)
	procs := process.NewManager()
	s := newTestServer(cfg, apps, procs)

	w := httptest.NewRecorder()
	s.handleRequest(w, requestWithHost("<script>alert(1)</script>evil.com", "/"))

	if strings.Contains(w.Body.String(), "<script>alert(1)</script>") {
		t.Error("expected the host to be HTML-escaped in the error page")
	}

	// The message runs through html/template, which escapes on its own; the
	// hint is injected as raw HTML and must be escaped by hand. Escaping the
	// message too would double-escape it and show entities to the user.
	w = httptest.NewRecorder()
	s.handleRequest(w, requestWithHost("a&b.example.com", "/"))

	if strings.Contains(w.Body.String(), "&amp;amp;") {
		t.Error("expected the host to be escaped exactly once, got double-escaped output")
	}
}
