package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSimpleApp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}
	store := NewAppStore(cfg)

	t.Run("parses port number", func(t *testing.T) {
		path := filepath.Join(tmpDir, "myapp")
		os.WriteFile(path, []byte("3000"), 0644)

		app, err := store.loadSimpleApp("myapp", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypePort {
			t.Errorf("expected AppTypePort, got %v", app.Type)
		}
		if app.Port != 3000 {
			t.Errorf("expected port 3000, got %d", app.Port)
		}
	})

	t.Run("parses command", func(t *testing.T) {
		path := filepath.Join(tmpDir, "cmdapp")
		os.WriteFile(path, []byte("rails server -p $PORT"), 0644)

		app, err := store.loadSimpleApp("cmdapp", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypeCommand {
			t.Errorf("expected AppTypeCommand, got %v", app.Type)
		}
		if app.Command != "rails server -p $PORT" {
			t.Errorf("unexpected command: %s", app.Command)
		}
	})

	t.Run("rejects empty file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty")
		os.WriteFile(path, []byte(""), 0644)

		_, err := store.loadSimpleApp("empty", path)
		if err == nil {
			t.Error("expected error for empty file")
		}
	})
}

func TestLoadYAMLApp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}
	store := NewAppStore(cfg)

	t.Run("parses single-service shorthand", func(t *testing.T) {
		yaml := `
name: myapp
description: My Application
root: /tmp/myapp
cmd: rails server -p $PORT
`
		path := filepath.Join(tmpDir, "myapp.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("myapp.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Name != "myapp" {
			t.Errorf("expected name 'myapp', got %s", app.Name)
		}
		if app.Description != "My Application" {
			t.Errorf("expected description 'My Application', got %s", app.Description)
		}
		if app.Type != AppTypeCommand {
			t.Errorf("expected AppTypeCommand, got %v", app.Type)
		}
		if app.Command != "rails server -p $PORT" {
			t.Errorf("unexpected command: %s", app.Command)
		}
	})

	t.Run("parses single service in services map", func(t *testing.T) {
		yaml := `
name: singleservice
root: /tmp/singleservice
services:
  web:
    cmd: rails server
`
		path := filepath.Join(tmpDir, "singleservice.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("singleservice.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Single service should be treated as simple command
		if app.Type != AppTypeCommand {
			t.Errorf("expected AppTypeCommand for single service, got %v", app.Type)
		}
	})

	t.Run("parses multi-service app", func(t *testing.T) {
		yaml := `
name: multiapp
root: /tmp/multiapp
services:
  web:
    cmd: rails server
    default: true
  worker:
    cmd: sidekiq
`
		path := filepath.Join(tmpDir, "multiapp.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("multiapp.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypeYAML {
			t.Errorf("expected AppTypeYAML, got %v", app.Type)
		}
		if len(app.Services) != 2 {
			t.Errorf("expected 2 services, got %d", len(app.Services))
		}

		// Check that default is set correctly
		var hasDefault bool
		for _, svc := range app.Services {
			if svc.Name == "web" && svc.Default {
				hasDefault = true
			}
		}
		if !hasDefault {
			t.Error("expected web service to have default: true")
		}
	})

	t.Run("uses filename when name not specified", func(t *testing.T) {
		yaml := `
root: /tmp/unnamed
cmd: rails server
`
		path := filepath.Join(tmpDir, "unnamed.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("unnamed.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Name != "unnamed" {
			t.Errorf("expected name 'unnamed', got %s", app.Name)
		}
	})

	t.Run("parses env variables", func(t *testing.T) {
		yaml := `
name: envapp
root: /tmp/envapp
cmd: rails server
env:
  RAILS_ENV: development
  DATABASE_URL: postgres://localhost/mydb
`
		path := filepath.Join(tmpDir, "envapp.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("envapp.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Env["RAILS_ENV"] != "development" {
			t.Errorf("expected RAILS_ENV=development, got %s", app.Env["RAILS_ENV"])
		}
	})
}

func TestAppStore(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}

	// Create some test apps
	os.WriteFile(filepath.Join(tmpDir, "app1"), []byte("3000"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app2"), []byte("rails server"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("3001"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"tld":"test"}`), 0644)

	t.Run("loads apps from directory", func(t *testing.T) {
		store := NewAppStore(cfg)
		err := store.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if _, found := store.Get("app1"); !found {
			t.Error("expected app1 to be loaded")
		}
		if _, found := store.Get("app2"); !found {
			t.Error("expected app2 to be loaded")
		}
	})

	t.Run("skips hidden files", func(t *testing.T) {
		store := NewAppStore(cfg)
		store.Load()

		if _, found := store.Get(".hidden"); found {
			t.Error("hidden file should not be loaded")
		}
	})

	t.Run("skips config.json", func(t *testing.T) {
		store := NewAppStore(cfg)
		store.Load()

		if _, found := store.Get("config.json"); found {
			t.Error("config.json should not be loaded as an app")
		}
	})

	t.Run("All returns sorted apps", func(t *testing.T) {
		store := NewAppStore(cfg)
		store.Load()

		apps := store.All()
		if len(apps) < 2 {
			t.Fatalf("expected at least 2 apps, got %d", len(apps))
		}
		// Apps should be sorted alphabetically
		for i := 1; i < len(apps); i++ {
			if apps[i-1].Name > apps[i].Name {
				t.Errorf("apps not sorted: %s > %s", apps[i-1].Name, apps[i].Name)
			}
		}
	})

	t.Run("Reload clears and reloads apps", func(t *testing.T) {
		store := NewAppStore(cfg)
		store.Load()

		// Add a new app
		os.WriteFile(filepath.Join(tmpDir, "app3"), []byte("4000"), 0644)

		err := store.Reload()
		if err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		if _, found := store.Get("app3"); !found {
			t.Error("expected app3 to be loaded after reload")
		}
	})
}

func TestGetService(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}

	yaml := `
name: multiapp
root: /tmp/multiapp
services:
  web:
    cmd: rails server
  worker:
    cmd: sidekiq
`
	os.WriteFile(filepath.Join(tmpDir, "multiapp.yml"), []byte(yaml), 0644)

	store := NewAppStore(cfg)
	store.Load()

	t.Run("finds service by name", func(t *testing.T) {
		app, svc, found := store.GetService("multiapp", "web")
		if !found {
			t.Fatal("expected to find web service")
		}
		if app.Name != "multiapp" {
			t.Errorf("expected app name 'multiapp', got %s", app.Name)
		}
		if svc.Name != "web" {
			t.Errorf("expected service name 'web', got %s", svc.Name)
		}
	})

	t.Run("returns false for unknown service", func(t *testing.T) {
		_, _, found := store.GetService("multiapp", "unknown")
		if found {
			t.Error("expected not to find unknown service")
		}
	})

	t.Run("returns false for unknown app", func(t *testing.T) {
		_, _, found := store.GetService("unknown", "web")
		if found {
			t.Error("expected not to find service in unknown app")
		}
	})
}

func TestTopologicalSort(t *testing.T) {
	t.Run("sorts services with dependencies after their dependencies", func(t *testing.T) {
		services := []Service{
			{Name: "web", DependsOn: []string{"api"}},
			{Name: "api", DependsOn: nil},
		}

		sorted := topologicalSort(services)

		if len(sorted) != 2 {
			t.Fatalf("expected 2 services, got %d", len(sorted))
		}
		// api should come before web
		if sorted[0].Name != "api" {
			t.Errorf("expected api first, got %s", sorted[0].Name)
		}
		if sorted[1].Name != "web" {
			t.Errorf("expected web second, got %s", sorted[1].Name)
		}
	})

	t.Run("handles chain of dependencies", func(t *testing.T) {
		services := []Service{
			{Name: "c", DependsOn: []string{"b"}},
			{Name: "a", DependsOn: nil},
			{Name: "b", DependsOn: []string{"a"}},
		}

		sorted := topologicalSort(services)

		// Should be: a, b, c
		if sorted[0].Name != "a" {
			t.Errorf("expected a first, got %s", sorted[0].Name)
		}
		if sorted[1].Name != "b" {
			t.Errorf("expected b second, got %s", sorted[1].Name)
		}
		if sorted[2].Name != "c" {
			t.Errorf("expected c third, got %s", sorted[2].Name)
		}
	})

	t.Run("handles no dependencies", func(t *testing.T) {
		services := []Service{
			{Name: "web"},
			{Name: "api"},
		}

		sorted := topologicalSort(services)

		// Should be alphabetically sorted when no deps
		if sorted[0].Name != "api" {
			t.Errorf("expected api first (alphabetical), got %s", sorted[0].Name)
		}
	})

	t.Run("handles cycle gracefully", func(t *testing.T) {
		services := []Service{
			{Name: "a", DependsOn: []string{"b"}},
			{Name: "b", DependsOn: []string{"a"}},
		}

		sorted := topologicalSort(services)

		// Should return original on cycle
		if len(sorted) != 2 {
			t.Errorf("expected 2 services returned on cycle, got %d", len(sorted))
		}
	})

	t.Run("ignores unknown dependencies", func(t *testing.T) {
		services := []Service{
			{Name: "web", DependsOn: []string{"unknown"}},
			{Name: "api"},
		}

		sorted := topologicalSort(services)

		// Should still sort, ignoring unknown dep
		if len(sorted) != 2 {
			t.Fatalf("expected 2 services, got %d", len(sorted))
		}
	})
}

func TestAliasResolution(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}

	t.Run("parses single alias from YAML", func(t *testing.T) {
		yaml := `
name: blog
alias: panozzaj
root: /tmp/blog
cmd: bin/serve
`
		os.WriteFile(filepath.Join(tmpDir, "blog.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		app, found := store.Get("blog")
		if !found {
			t.Fatal("expected blog app to be loaded")
		}
		if len(app.Aliases) != 1 || app.Aliases[0] != "panozzaj" {
			t.Errorf("expected alias 'panozzaj', got %v", app.Aliases)
		}
	})

	t.Run("parses multiple aliases from YAML", func(t *testing.T) {
		yaml := `
name: mysite
aliases:
  - site1
  - site2
  - site3
root: /tmp/mysite
cmd: npm start
`
		os.WriteFile(filepath.Join(tmpDir, "mysite.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		app, found := store.Get("mysite")
		if !found {
			t.Fatal("expected mysite app to be loaded")
		}
		if len(app.Aliases) != 3 {
			t.Errorf("expected 3 aliases, got %d", len(app.Aliases))
		}
	})

	t.Run("merges alias and aliases fields", func(t *testing.T) {
		yaml := `
name: merged
alias: single
aliases:
  - multi1
  - multi2
root: /tmp/merged
cmd: npm start
`
		os.WriteFile(filepath.Join(tmpDir, "merged.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		app, found := store.Get("merged")
		if !found {
			t.Fatal("expected merged app to be loaded")
		}
		// Should have all 3 aliases
		if len(app.Aliases) != 3 {
			t.Errorf("expected 3 aliases (merged), got %d: %v", len(app.Aliases), app.Aliases)
		}
	})

	t.Run("GetByNameOrAlias finds by exact name", func(t *testing.T) {
		yaml := `
name: exactapp
alias: exactalias
root: /tmp/exactapp
cmd: npm start
`
		os.WriteFile(filepath.Join(tmpDir, "exactapp.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		app, found := store.GetByNameOrAlias("exactapp")
		if !found {
			t.Fatal("expected to find app by exact name")
		}
		if app.Name != "exactapp" {
			t.Errorf("expected name 'exactapp', got %s", app.Name)
		}
	})

	t.Run("GetByNameOrAlias finds by alias", func(t *testing.T) {
		yaml := `
name: realname
alias: nickname
root: /tmp/realname
cmd: npm start
`
		os.WriteFile(filepath.Join(tmpDir, "realname.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		app, found := store.GetByNameOrAlias("nickname")
		if !found {
			t.Fatal("expected to find app by alias")
		}
		if app.Name != "realname" {
			t.Errorf("expected name 'realname', got %s", app.Name)
		}
	})

	t.Run("GetByNameOrAlias returns false for unknown", func(t *testing.T) {
		store := NewAppStore(cfg)
		store.Load()

		_, found := store.GetByNameOrAlias("nonexistent")
		if found {
			t.Error("expected not to find nonexistent app")
		}
	})

	t.Run("GetService finds multi-service app by alias", func(t *testing.T) {
		yaml := `
name: multiserviceapp
alias: msa
root: /tmp/multiserviceapp
services:
  web:
    cmd: rails server
  worker:
    cmd: sidekiq
`
		os.WriteFile(filepath.Join(tmpDir, "multiserviceapp.yml"), []byte(yaml), 0644)

		store := NewAppStore(cfg)
		store.Load()

		// Should find by alias
		app, svc, found := store.GetService("msa", "web")
		if !found {
			t.Fatal("expected to find service via app alias")
		}
		if app.Name != "multiserviceapp" {
			t.Errorf("expected app name 'multiserviceapp', got %s", app.Name)
		}
		if svc.Name != "web" {
			t.Errorf("expected service name 'web', got %s", svc.Name)
		}
	})
}

func TestDependsOnParsing(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}
	store := NewAppStore(cfg)

	t.Run("parses depends_on from YAML", func(t *testing.T) {
		yaml := `
name: myapp
root: /tmp/myapp
services:
  web:
    cmd: npm start
    depends_on: [api, worker]
  api:
    cmd: python server.py
  worker:
    cmd: sidekiq
`
		path := filepath.Join(tmpDir, "myapp.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("myapp.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Find web service
		var webSvc *Service
		for i := range app.Services {
			if app.Services[i].Name == "web" {
				webSvc = &app.Services[i]
				break
			}
		}

		if webSvc == nil {
			t.Fatal("web service not found")
		}

		if len(webSvc.DependsOn) != 2 {
			t.Errorf("expected 2 dependencies, got %d", len(webSvc.DependsOn))
		}
	})

	t.Run("services are topologically sorted", func(t *testing.T) {
		yaml := `
name: depapp
root: /tmp/depapp
services:
  web:
    cmd: npm start
    depends_on: [api]
  api:
    cmd: python server.py
`
		path := filepath.Join(tmpDir, "depapp.yml")
		os.WriteFile(path, []byte(yaml), 0644)

		app, err := store.loadYAMLApp("depapp.yml", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// api should come before web in the services slice
		if app.Services[0].Name != "api" {
			t.Errorf("expected api first after topological sort, got %s", app.Services[0].Name)
		}
		if app.Services[1].Name != "web" {
			t.Errorf("expected web second after topological sort, got %s", app.Services[1].Name)
		}
	})
}

func TestLoadStaticApp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Dir: tmpDir}
	store := NewAppStore(cfg)

	t.Run("loads directory with index.html", func(t *testing.T) {
		// Create a static site directory
		staticDir := filepath.Join(tmpDir, "mysite")
		os.MkdirAll(staticDir, 0755)
		os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>Hello</html>"), 0644)

		app, err := store.loadStaticApp("mysite", staticDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypeStatic {
			t.Errorf("expected AppTypeStatic, got %v", app.Type)
		}
		if app.FilePath != staticDir {
			t.Errorf("expected FilePath %s, got %s", staticDir, app.FilePath)
		}
	})

	t.Run("allows directory without index.html", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty-site")
		os.MkdirAll(emptyDir, 0755)

		app, err := store.loadStaticApp("empty-site", emptyDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if app.Type != AppTypeStatic {
			t.Errorf("expected AppTypeStatic, got %v", app.Type)
		}
		if app.FilePath != emptyDir {
			t.Errorf("expected FilePath %s, got %s", emptyDir, app.FilePath)
		}
	})

	t.Run("loads symlink to directory", func(t *testing.T) {
		// Create target directory
		targetDir := filepath.Join(tmpDir, "target-site")
		os.MkdirAll(targetDir, 0755)
		os.WriteFile(filepath.Join(targetDir, "index.html"), []byte("<html>Target</html>"), 0644)

		// Create symlink config
		linkPath := filepath.Join(tmpDir, "linked-site")
		os.Symlink(targetDir, linkPath)

		// loadApp should follow symlink
		app, err := store.loadApp("linked-site", linkPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypeStatic {
			t.Errorf("expected AppTypeStatic, got %v", app.Type)
		}
		if app.FilePath != targetDir {
			t.Errorf("expected FilePath %s, got %s", targetDir, app.FilePath)
		}
	})

	t.Run("loads file path as static site", func(t *testing.T) {
		// Create a simple file config pointing to a directory
		staticDir := filepath.Join(tmpDir, "path-site")
		os.MkdirAll(staticDir, 0755)
		os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>Path</html>"), 0644)

		configFile := filepath.Join(tmpDir, "pathconfig")
		os.WriteFile(configFile, []byte(staticDir), 0644)

		app, err := store.loadSimpleApp("pathconfig", configFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Type != AppTypeStatic {
			t.Errorf("expected AppTypeStatic, got %v", app.Type)
		}
	})
}
