package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Default(t *testing.T) {
	// With non-existent config, should return defaults
	tmpDir := t.TempDir()
	cfg, err := loadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TLD != "test" {
		t.Errorf("expected default TLD 'test', got %q", cfg.TLD)
	}
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a config file
	content := `{"tld": "local", "claude_command": "claude-custom"}`
	if err := os.WriteFile(filepath.Join(tmpDir, globalConfigName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TLD != "local" {
		t.Errorf("expected TLD 'local', got %q", cfg.TLD)
	}
	if cfg.ClaudeCommand != "claude-custom" {
		t.Errorf("expected ClaudeCommand 'claude-custom', got %q", cfg.ClaudeCommand)
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &GlobalConfig{
		TLD:           "dev",
		ClaudeCommand: "claude-dev",
		Ollama: &OllamaConfig{
			Enabled: true,
			URL:     "http://localhost:11434",
			Model:   "llama3.2",
		},
	}

	if err := saveGlobalConfig(tmpDir, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read it back
	loaded, err := loadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.TLD != cfg.TLD {
		t.Errorf("TLD mismatch: got %q, want %q", loaded.TLD, cfg.TLD)
	}
	if loaded.ClaudeCommand != cfg.ClaudeCommand {
		t.Errorf("ClaudeCommand mismatch: got %q, want %q", loaded.ClaudeCommand, cfg.ClaudeCommand)
	}
	if loaded.Ollama == nil {
		t.Fatal("expected Ollama config to be set")
	}
	if !loaded.Ollama.Enabled {
		t.Error("expected Ollama.Enabled to be true")
	}
	if loaded.Ollama.Model != "llama3.2" {
		t.Errorf("Ollama.Model mismatch: got %q, want %q", loaded.Ollama.Model, "llama3.2")
	}
}

func TestLoadGlobalConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(tmpDir, globalConfigName), []byte("{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadGlobalConfig(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
