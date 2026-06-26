package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsDNSInstalled(t *testing.T) {
	// Just test that it doesn't panic
	_ = isDNSInstalled("test")
}

func TestIsCertInstalled(t *testing.T) {
	// Create a temp config dir
	tmpDir := t.TempDir()

	// Without cert files, should return false
	if isCertInstalled(tmpDir) {
		t.Error("expected isCertInstalled to return false for empty dir")
	}

	// Create the expected cert files (ca-key.pem and ca.pem)
	certsDir := filepath.Join(tmpDir, "certs")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "ca-key.pem"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "ca.pem"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// With cert files, should return true
	if !isCertInstalled(tmpDir) {
		t.Error("expected isCertInstalled to return true with cert files")
	}
}

func TestGetProcessOnPort(t *testing.T) {
	// Test with a port that's unlikely to have anything listening
	proc := getProcessOnPort(59999)
	if proc != "" {
		t.Logf("found process on port 59999: %s", proc)
	}
}

func TestIsServiceInstalled(t *testing.T) {
	// Just test that it doesn't panic and returns valid values
	installed, running := isServiceInstalled()
	// If not installed, running must be false
	if !installed && running {
		t.Error("running cannot be true if not installed")
	}
}

func TestGetUserLaunchAgentPath(t *testing.T) {
	path := getUserLaunchAgentPath()

	// Should contain the expected plist name
	if !strings.Contains(path, "com.fireup.plist") {
		t.Errorf("expected path to contain com.fireup.plist, got %s", path)
	}

	// Should be in LaunchAgents directory
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("expected path to contain LaunchAgents, got %s", path)
	}
}

func TestGetCertsDir(t *testing.T) {
	dir := getCertsDir()

	// Should end with /certs
	if !strings.HasSuffix(dir, "/certs") {
		t.Errorf("expected dir to end with /certs, got %s", dir)
	}

	// Should contain fireup
	if !strings.Contains(dir, "fireup") {
		t.Errorf("expected dir to contain fireup, got %s", dir)
	}
}

func TestGetResolverContent(t *testing.T) {
	content := getResolverContent()

	// Should contain nameserver and port
	if !strings.Contains(content, "nameserver 127.0.0.1") {
		t.Error("expected content to contain nameserver 127.0.0.1")
	}
	if !strings.Contains(content, "port 9053") {
		t.Error("expected content to contain port 9053")
	}
}

func TestDefaultPorts(t *testing.T) {
	if DefaultHTTPPort != 80 {
		t.Errorf("expected DefaultHTTPPort to be 80, got %d", DefaultHTTPPort)
	}
	if DefaultHTTPSPort != 443 {
		t.Errorf("expected DefaultHTTPSPort to be 443, got %d", DefaultHTTPSPort)
	}
	if DefaultDNSPort != 9053 {
		t.Errorf("expected DefaultDNSPort to be 9053, got %d", DefaultDNSPort)
	}
}
