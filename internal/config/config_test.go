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
