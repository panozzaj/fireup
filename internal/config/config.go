package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config holds the global configuration
type Config struct {
	Dir       string
	HTTPPort  int  // Port to listen on
	HTTPSPort int
	URLPort   int  // Port to use in generated URLs (for pf forwarding)
	TLD       string
}

// App represents a configured application
type App struct {
	Name        string
	Description string   // Optional display name/description
	Type        AppType
	Port        int      // For static port proxy
	Command     string   // For command-based apps
	Dir         string   // Working directory
	FilePath    string   // For static file serving
	Services    []Service // For multi-service YAML configs
	Env         map[string]string
}

// Service represents a service within a multi-service app
type Service struct {
	Name    string
	Dir     string
	Command string
	Port    int // Assigned dynamically
	Env     map[string]string
	Default bool // If true, this service handles requests to the base app URL
}

// AppType indicates how to handle the app
type AppType int

const (
	AppTypePort    AppType = iota // Proxy to fixed port
	AppTypeCommand               // Run command with dynamic port
	AppTypeStatic                // Serve static files
	AppTypeYAML                  // Multi-service YAML config
)

// AppStore manages loaded app configurations
type AppStore struct {
	mu   sync.RWMutex
	apps map[string]*App
	cfg  *Config
}

// NewAppStore creates a new app store
func NewAppStore(cfg *Config) *AppStore {
	return &AppStore{
		apps: make(map[string]*App),
		cfg:  cfg,
	}
}

// Load reads all configurations from the config directory
func (s *AppStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(s.cfg.Dir, name)

		// Skip hidden files and the global config file
		if strings.HasPrefix(name, ".") || name == "config.json" {
			continue
		}

		app, err := s.loadApp(name, path)
		if err != nil {
			fmt.Printf("Warning: failed to load %s: %v\n", name, err)
			continue
		}

		s.apps[app.Name] = app
	}

	return nil
}

// loadApp loads a single app configuration
func (s *AppStore) loadApp(name, path string) (*App, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	// Handle symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return nil, err
		}
		// Expand ~ in symlink target
		if strings.HasPrefix(target, "~") {
			home, _ := os.UserHomeDir()
			target = filepath.Join(home, target[1:])
		}
		return s.loadStaticApp(name, target)
	}

	// Handle directories (serve as static)
	if info.IsDir() {
		return s.loadStaticApp(name, path)
	}

	// Handle YAML files
	if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
		return s.loadYAMLApp(name, path)
	}

	// Handle simple files (port, command, or path)
	return s.loadSimpleApp(name, path)
}

// loadStaticApp creates an app config for static file serving
func (s *AppStore) loadStaticApp(name, path string) (*App, error) {
	// Check if it's a directory or file
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		// Check for index.html
		indexPath := filepath.Join(path, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			return nil, fmt.Errorf("directory has no index.html: %s", path)
		}
	}

	return &App{
		Name:     name,
		Type:     AppTypeStatic,
		FilePath: path,
	}, nil
}

// loadYAMLApp loads a YAML configuration (single or multi-service)
func (s *AppStore) loadYAMLApp(name, path string) (*App, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var yamlCfg struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Root        string `yaml:"root"`
		Command     string `yaml:"cmd"` // For single-service shorthand
		Env         map[string]string `yaml:"env"` // For single-service shorthand
		Services    map[string]struct {
			Dir     string            `yaml:"dir"`
			Command string            `yaml:"cmd"`
			Env     map[string]string `yaml:"env"`
			Default bool              `yaml:"default"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Use filename without extension if name not specified
	appName := yamlCfg.Name
	if appName == "" {
		appName = strings.TrimSuffix(name, filepath.Ext(name))
	}

	// Expand ~ in root
	root := yamlCfg.Root
	if strings.HasPrefix(root, "~") {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, root[1:])
	}

	// Single-service shorthand: cmd at top level
	if yamlCfg.Command != "" {
		return &App{
			Name:        appName,
			Description: yamlCfg.Description,
			Type:        AppTypeCommand,
			Command:     yamlCfg.Command,
			Dir:         root,
			Env:         yamlCfg.Env,
		}, nil
	}

	// Single service in services map → treat as simple command
	if len(yamlCfg.Services) == 1 {
		for _, svcCfg := range yamlCfg.Services {
			svcDir := root
			if svcCfg.Dir != "" {
				svcDir = filepath.Join(root, svcCfg.Dir)
			}
			return &App{
				Name:        appName,
				Description: yamlCfg.Description,
				Type:        AppTypeCommand,
				Command:     svcCfg.Command,
				Dir:         svcDir,
				Env:         svcCfg.Env,
			}, nil
		}
	}

	// Multi-service
	var services []Service
	for svcName, svcCfg := range yamlCfg.Services {
		svcDir := root
		if svcCfg.Dir != "" {
			svcDir = filepath.Join(root, svcCfg.Dir)
		}

		services = append(services, Service{
			Name:    svcName,
			Dir:     svcDir,
			Command: svcCfg.Command,
			Env:     svcCfg.Env,
			Default: svcCfg.Default,
		})
	}

	return &App{
		Name:        appName,
		Description: yamlCfg.Description,
		Type:        AppTypeYAML,
		Dir:         root,
		Services:    services,
	}, nil
}

// loadSimpleApp loads a simple config file (port number, command, or path)
func (s *AppStore) loadSimpleApp(name, path string) (*App, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, fmt.Errorf("empty config file")
	}

	// Check if it's a port number
	if port, err := strconv.Atoi(content); err == nil && port > 0 && port < 65536 {
		return &App{
			Name: name,
			Type: AppTypePort,
			Port: port,
		}, nil
	}

	// Check if it's a file path (starts with / or ~)
	if strings.HasPrefix(content, "/") || strings.HasPrefix(content, "~") {
		filePath := content
		if strings.HasPrefix(filePath, "~") {
			home, _ := os.UserHomeDir()
			filePath = filepath.Join(home, filePath[1:])
		}

		// Verify it exists
		if _, err := os.Stat(filePath); err == nil {
			return s.loadStaticApp(name, filePath)
		}
	}

	// Otherwise treat as a command
	return &App{
		Name:    name,
		Type:    AppTypeCommand,
		Command: content,
	}, nil
}

// Get returns an app by name
func (s *AppStore) Get(name string) (*App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[name]
	return app, ok
}

// GetService returns a specific service from a multi-service app
func (s *AppStore) GetService(appName, serviceName string) (*App, *Service, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.apps[appName]
	if !ok || app.Type != AppTypeYAML {
		return nil, nil, false
	}

	for i := range app.Services {
		if app.Services[i].Name == serviceName {
			return app, &app.Services[i], true
		}
	}

	return app, nil, false
}

// All returns all loaded apps sorted alphabetically
func (s *AppStore) All() []*App {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apps := make([]*App, 0, len(s.apps))
	for _, app := range s.apps {
		apps = append(apps, app)
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})

	return apps
}

// Reload refreshes the app configurations
func (s *AppStore) Reload() error {
	// Clear existing
	s.mu.Lock()
	s.apps = make(map[string]*App)
	s.mu.Unlock()

	return s.Load()
}
