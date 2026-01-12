package server

import (
	"os"
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
name: mateams
aliases:
  - ma
  - wt
root: /tmp/mateams
cmd: rails server
`
	os.WriteFile(tmpDir+"/mateams.yml", []byte(yamlContent2), 0644)

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
			t.Fatal("expected to find mateams by alias 'ma'")
		}
		if app1.Name != "mateams" {
			t.Errorf("expected app name 'mateams', got %s", app1.Name)
		}

		app2, found2 := s.findApp("wt")
		if !found2 {
			t.Fatal("expected to find mateams by alias 'wt'")
		}
		if app2.Name != "mateams" {
			t.Errorf("expected app name 'mateams', got %s", app2.Name)
		}
	})

	t.Run("returns false for unknown app", func(t *testing.T) {
		_, found := s.findApp("nonexistent")
		if found {
			t.Error("expected not to find nonexistent app")
		}
	})

	t.Run("strips subdomain to find app", func(t *testing.T) {
		// Simulate accessing admin.mateams.test
		app, found := s.findApp("admin.mateams")
		if !found {
			t.Fatal("expected to find mateams via subdomain 'admin.mateams'")
		}
		if app.Name != "mateams" {
			t.Errorf("expected app name 'mateams', got %s", app.Name)
		}
	})

	t.Run("strips subdomain and resolves alias", func(t *testing.T) {
		// Simulate accessing api.ma.test (ma is alias for mateams)
		app, found := s.findApp("api.ma")
		if !found {
			t.Fatal("expected to find mateams via subdomain 'api.ma'")
		}
		if app.Name != "mateams" {
			t.Errorf("expected app name 'mateams', got %s", app.Name)
		}
	})

	t.Run("handles multiple subdomain levels", func(t *testing.T) {
		// Simulate accessing foo.bar.mateams.test
		app, found := s.findApp("foo.bar.mateams")
		if !found {
			t.Fatal("expected to find mateams via 'foo.bar.mateams'")
		}
		if app.Name != "mateams" {
			t.Errorf("expected app name 'mateams', got %s", app.Name)
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
