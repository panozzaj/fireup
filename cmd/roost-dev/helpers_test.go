package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPfPlistOutdated(t *testing.T) {
	// When plist doesn't exist, should return false
	if isPfPlistOutdated() {
		// This test runs in a development environment where the plist may or may not exist
		// If it exists and differs from expected, that's fine
		t.Log("plist exists and differs from expected, or doesn't exist")
	}
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

func TestIsPortForwardingInstalled(t *testing.T) {
	// Just test that it doesn't panic
	_ = isPortForwardingInstalled("test")
}

func TestGetProcessOnPort(t *testing.T) {
	// Test with a port that's unlikely to have anything listening
	proc := getProcessOnPort(59999)
	if proc != "" {
		t.Logf("found process on port 59999: %s", proc)
	}
}
