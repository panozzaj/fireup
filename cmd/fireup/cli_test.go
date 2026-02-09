package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/panozzaj/fireup/internal/diff"
)

func TestCheckHelpFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"no args", []string{}, false},
		{"no help flag", []string{"foo", "bar"}, false},
		{"short help", []string{"-h"}, true},
		{"long help", []string{"--help"}, true},
		{"help word", []string{"help"}, true},
		{"help in middle", []string{"foo", "-h", "bar"}, true},
		{"similar but not help", []string{"-help", "helper"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkHelpFlag(tt.args, "test usage")
			if result != tt.expected {
				t.Errorf("checkHelpFlag(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfigDir(t *testing.T) {
	dir := getDefaultConfigDir()

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".config", "fireup")

	if dir != expected {
		t.Errorf("getDefaultConfigDir() = %q, want %q", dir, expected)
	}
}

func TestGetConfigWithDefaults(t *testing.T) {
	cfg, configDir := getConfigWithDefaults()

	if cfg == nil {
		t.Fatal("getConfigWithDefaults() returned nil config")
	}

	if cfg.TLD == "" {
		t.Error("expected non-empty TLD")
	}

	if configDir == "" {
		t.Error("expected non-empty configDir")
	}
}

func TestIsRoot(t *testing.T) {
	// Just verify it doesn't panic and returns a boolean
	result := isRoot()
	// We can't easily test the actual value since it depends on how tests are run
	_ = result
}

func TestRequireNonRoot(t *testing.T) {
	err := requireNonRoot("test action")

	if isRoot() {
		if err == nil {
			t.Error("expected error when running as root")
		}
	} else {
		if err != nil {
			t.Errorf("unexpected error when not root: %v", err)
		}
	}
}

func TestIsAgent(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"set to 1", "1", true},
		{"unset", "", false},
		{"set to 0", "0", false},
		{"set to true", "true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("CLAUDECODE", tt.envValue)
			} else {
				t.Setenv("CLAUDECODE", "")
			}
			if got := isAgent(); got != tt.want {
				t.Errorf("isAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmStep_AgentMode(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("FIREUP_YES", "")

	if confirmStep("Do something?") {
		t.Error("confirmStep should return false in agent mode")
	}
}

func TestConfirmStep_FireupYesTakesPrecedence(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("FIREUP_YES", "1")

	if !confirmStep("Do something?") {
		t.Error("confirmStep should return true when FIREUP_YES=1, even in agent mode")
	}
}

func TestConfirmWithPlan_AgentMode(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("FIREUP_YES", "")

	plan := diff.NewPlan()
	plan.CreateStatic("/tmp/fireup-test-nonexistent", "test")

	if confirmWithPlan(plan, "Apply changes?") {
		t.Error("confirmWithPlan should return false in agent mode")
	}
}

func TestConfirmWithPlan_FireupYesTakesPrecedence(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("FIREUP_YES", "1")

	plan := diff.NewPlan()
	plan.CreateStatic("/tmp/fireup-test-nonexistent", "test")

	if !confirmWithPlan(plan, "Apply changes?") {
		t.Error("confirmWithPlan should return true when FIREUP_YES=1, even in agent mode")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"a", "a", 0},
		{"a", "b", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "adc", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
		{"my-app", "my-ap", 1},
		{"my-app", "myapp", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := levenshtein(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestFilterApps(t *testing.T) {
	apps := []AppStatus{
		{Name: "my-app", Type: "multi-service", Services: []SvcStatus{
			{Name: "web", Running: true},
			{Name: "worker", Running: false},
		}},
		{Name: "frontend", Type: "command", Aliases: []string{"ui"}},
		{Name: "api-server", Type: "command"},
	}

	tests := []struct {
		name          string
		filter        string
		expectedApps  int
		expectedNames []string
		checkServices bool
		expectedSvcs  int
	}{
		{
			name:          "exact app name match",
			filter:        "my-app",
			expectedApps:  1,
			expectedNames: []string{"my-app"},
		},
		{
			name:          "partial app name match",
			filter:        "my-",
			expectedApps:  1,
			expectedNames: []string{"my-app"},
		},
		{
			name:          "case insensitive match",
			filter:        "MY-APP",
			expectedApps:  1,
			expectedNames: []string{"my-app"},
		},
		{
			name:          "alias match",
			filter:        "ui",
			expectedApps:  1,
			expectedNames: []string{"frontend"},
		},
		{
			name:          "service name match filters services",
			filter:        "worker",
			expectedApps:  1,
			expectedNames: []string{"my-app"},
			checkServices: true,
			expectedSvcs:  1,
		},
		{
			name:          "multiple matches",
			filter:        "api",
			expectedApps:  1,
			expectedNames: []string{"api-server"},
		},
		{
			name:         "no match",
			filter:       "nonexistent",
			expectedApps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterApps(apps, tt.filter)
			if len(result) != tt.expectedApps {
				t.Errorf("filterApps() returned %d apps, want %d", len(result), tt.expectedApps)
			}
			for i, name := range tt.expectedNames {
				if i < len(result) && result[i].Name != name {
					t.Errorf("filterApps()[%d].Name = %q, want %q", i, result[i].Name, name)
				}
			}
			if tt.checkServices && len(result) > 0 {
				if len(result[0].Services) != tt.expectedSvcs {
					t.Errorf("filterApps()[0].Services has %d services, want %d", len(result[0].Services), tt.expectedSvcs)
				}
			}
		})
	}
}

func TestFindSimilarNames(t *testing.T) {
	apps := []AppStatus{
		{Name: "my-app", Type: "multi-service", Services: []SvcStatus{
			{Name: "web"},
			{Name: "worker"},
		}},
		{Name: "frontend", Aliases: []string{"ui"}},
		{Name: "api-server"},
	}

	tests := []struct {
		name             string
		filter           string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "typo in app name",
			filter:        "my-ap",
			shouldContain: []string{"my-app"},
		},
		{
			name:          "typo in app name 2",
			filter:        "my-apa",
			shouldContain: []string{"my-app"},
		},
		{
			name:          "partial prefix match",
			filter:        "fro",
			shouldContain: []string{"frontend"},
		},
		{
			name:             "completely different",
			filter:           "xyz123",
			shouldNotContain: []string{"my-app", "frontend", "api-server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSimilarNames(apps, tt.filter)
			for _, expected := range tt.shouldContain {
				found := false
				for _, r := range result {
					if r == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("findSimilarNames(%q) should contain %q, got %v", tt.filter, expected, result)
				}
			}
			for _, notExpected := range tt.shouldNotContain {
				for _, r := range result {
					if r == notExpected {
						t.Errorf("findSimilarNames(%q) should not contain %q", tt.filter, notExpected)
					}
				}
			}
		})
	}
}
