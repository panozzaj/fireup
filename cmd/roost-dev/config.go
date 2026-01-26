package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const globalConfigName = "config.json"

// GlobalConfig stores persistent settings
type GlobalConfig struct {
	TLD           string        `json:"tld"`
	Ollama        *OllamaConfig `json:"ollama,omitempty"`
	ClaudeCommand string        `json:"claude_command,omitempty"` // Command to run Claude Code (default: "claude")
}

// OllamaConfig stores settings for local LLM error analysis
type OllamaConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`   // e.g., "http://localhost:11434"
	Model   string `json:"model"` // e.g., "llama3.2"
}

func loadGlobalConfig(configDir string) (*GlobalConfig, error) {
	path := filepath.Join(configDir, globalConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{TLD: "test"}, nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveGlobalConfig(configDir string, cfg *GlobalConfig) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(configDir, globalConfigName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
