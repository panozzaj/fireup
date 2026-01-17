// Package setup provides filesystem abstractions for setup/teardown operations.
package setup

import (
	"os"
	"path/filepath"
)

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(path string) error
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(name string) ([]byte, error)
	UserHomeDir() (string, error)
}

// OSFileSystem implements FileSystem using the real os package.
type OSFileSystem struct{}

func (OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// Checker verifies installation status of roost-dev components.
type Checker struct {
	FS FileSystem
}

// NewChecker creates a Checker with the real filesystem.
func NewChecker() *Checker {
	return &Checker{FS: OSFileSystem{}}
}

// IsPortForwardingInstalled checks if port forwarding appears to be set up.
func (c *Checker) IsPortForwardingInstalled(tld string) bool {
	if _, err := c.FS.Stat("/etc/pf.anchors/roost-dev"); err != nil {
		return false
	}
	if _, err := c.FS.Stat("/Library/LaunchDaemons/dev.roost.pfctl.plist"); err != nil {
		return false
	}
	resolverPath := "/etc/resolver/" + tld
	if _, err := c.FS.Stat(resolverPath); err != nil {
		return false
	}
	return true
}

// IsCertInstalled checks if certificates appear to be set up.
func (c *Checker) IsCertInstalled(configDir string) bool {
	caKeyPath := filepath.Join(configDir, "certs", "ca-key.pem")
	caCertPath := filepath.Join(configDir, "certs", "ca.pem")
	if _, err := c.FS.Stat(caKeyPath); err != nil {
		return false
	}
	if _, err := c.FS.Stat(caCertPath); err != nil {
		return false
	}
	return true
}

// IsServiceInstalled checks if the background service plist exists.
// Note: This only checks for the plist file, not if the service is running
// (running status requires exec.Command which is not abstracted here).
func (c *Checker) IsServiceInstalled(homeDir string) bool {
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.roost-dev.plist")
	if _, err := c.FS.Stat(plistPath); err != nil {
		return false
	}
	return true
}
