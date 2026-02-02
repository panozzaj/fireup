package setup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockFileSystem is an in-memory filesystem for testing.
type MockFileSystem struct {
	Files   map[string][]byte // path -> content
	HomeDir string
}

// NewMockFileSystem creates an empty mock filesystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:   make(map[string][]byte),
		HomeDir: "/Users/testuser",
	}
}

// mockFileInfo implements os.FileInfo for mock files.
type mockFileInfo struct {
	name string
	size int64
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() any           { return nil }

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if content, ok := m.Files[name]; ok {
		return mockFileInfo{name: filepath.Base(name), size: int64(len(content))}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	m.Files[name] = data
	return nil
}

func (m *MockFileSystem) Remove(name string) error {
	if _, ok := m.Files[name]; ok {
		delete(m.Files, name)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFileSystem) RemoveAll(path string) error {
	// Remove all files with this prefix
	for k := range m.Files {
		if k == path || len(k) > len(path) && k[:len(path)+1] == path+"/" {
			delete(m.Files, k)
		}
	}
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// Directories are implicit in mock - just return success
	return nil
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if content, ok := m.Files[name]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.HomeDir, nil
}

// AddFile adds a file to the mock filesystem.
func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.Files[path] = content
}

func TestIsPortForwardingInstalled(t *testing.T) {
	t.Run("returns false when no files exist", func(t *testing.T) {
		fs := NewMockFileSystem()
		checker := &Checker{FS: fs}

		if checker.IsPortForwardingInstalled("test") {
			t.Error("expected false when no files exist")
		}
	})

	t.Run("returns false when only anchor exists", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/etc/pf.anchors/fireup", []byte("rules"))
		checker := &Checker{FS: fs}

		if checker.IsPortForwardingInstalled("test") {
			t.Error("expected false when only anchor exists")
		}
	})

	t.Run("returns false when LaunchDaemon missing", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/etc/pf.anchors/fireup", []byte("rules"))
		fs.AddFile("/etc/resolver/test", []byte("nameserver"))
		checker := &Checker{FS: fs}

		if checker.IsPortForwardingInstalled("test") {
			t.Error("expected false when LaunchDaemon missing")
		}
	})

	t.Run("returns false when resolver missing", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/etc/pf.anchors/fireup", []byte("rules"))
		fs.AddFile("/Library/LaunchDaemons/dev.fireup.pfctl.plist", []byte("plist"))
		checker := &Checker{FS: fs}

		if checker.IsPortForwardingInstalled("test") {
			t.Error("expected false when resolver missing")
		}
	})

	t.Run("returns true when all files exist", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/etc/pf.anchors/fireup", []byte("rules"))
		fs.AddFile("/Library/LaunchDaemons/dev.fireup.pfctl.plist", []byte("plist"))
		fs.AddFile("/etc/resolver/test", []byte("nameserver"))
		checker := &Checker{FS: fs}

		if !checker.IsPortForwardingInstalled("test") {
			t.Error("expected true when all files exist")
		}
	})

	t.Run("uses correct TLD for resolver path", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/etc/pf.anchors/fireup", []byte("rules"))
		fs.AddFile("/Library/LaunchDaemons/dev.fireup.pfctl.plist", []byte("plist"))
		fs.AddFile("/etc/resolver/dev", []byte("nameserver"))
		checker := &Checker{FS: fs}

		if !checker.IsPortForwardingInstalled("dev") {
			t.Error("expected true with dev TLD")
		}
		if checker.IsPortForwardingInstalled("test") {
			t.Error("expected false with test TLD when dev resolver exists")
		}
	})
}

func TestIsCertInstalled(t *testing.T) {
	t.Run("returns false when no certs exist", func(t *testing.T) {
		fs := NewMockFileSystem()
		checker := &Checker{FS: fs}

		if checker.IsCertInstalled("/config") {
			t.Error("expected false when no certs exist")
		}
	})

	t.Run("returns false when only key exists", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/config/certs/ca-key.pem", []byte("key"))
		checker := &Checker{FS: fs}

		if checker.IsCertInstalled("/config") {
			t.Error("expected false when only key exists")
		}
	})

	t.Run("returns false when only cert exists", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/config/certs/ca.pem", []byte("cert"))
		checker := &Checker{FS: fs}

		if checker.IsCertInstalled("/config") {
			t.Error("expected false when only cert exists")
		}
	})

	t.Run("returns true when both files exist", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/config/certs/ca-key.pem", []byte("key"))
		fs.AddFile("/config/certs/ca.pem", []byte("cert"))
		checker := &Checker{FS: fs}

		if !checker.IsCertInstalled("/config") {
			t.Error("expected true when both files exist")
		}
	})

	t.Run("uses correct config directory", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/home/user/.config/fireup/certs/ca-key.pem", []byte("key"))
		fs.AddFile("/home/user/.config/fireup/certs/ca.pem", []byte("cert"))
		checker := &Checker{FS: fs}

		if !checker.IsCertInstalled("/home/user/.config/fireup") {
			t.Error("expected true with full config path")
		}
		if checker.IsCertInstalled("/other/config") {
			t.Error("expected false with different config path")
		}
	})
}

func TestIsServiceInstalled(t *testing.T) {
	t.Run("returns false when plist does not exist", func(t *testing.T) {
		fs := NewMockFileSystem()
		checker := &Checker{FS: fs}

		if checker.IsServiceInstalled("/Users/testuser") {
			t.Error("expected false when plist does not exist")
		}
	})

	t.Run("returns true when plist exists", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/Users/testuser/Library/LaunchAgents/com.fireup.plist", []byte("plist"))
		checker := &Checker{FS: fs}

		if !checker.IsServiceInstalled("/Users/testuser") {
			t.Error("expected true when plist exists")
		}
	})

	t.Run("uses correct home directory", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/Users/alice/Library/LaunchAgents/com.fireup.plist", []byte("plist"))
		checker := &Checker{FS: fs}

		if !checker.IsServiceInstalled("/Users/alice") {
			t.Error("expected true for alice's home")
		}
		if checker.IsServiceInstalled("/Users/bob") {
			t.Error("expected false for bob's home")
		}
	})
}

func TestMockFileSystem(t *testing.T) {
	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		fs := NewMockFileSystem()
		content := []byte("hello world")

		err := fs.WriteFile("/test/file.txt", content, 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		read, err := fs.ReadFile("/test/file.txt")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}

		if string(read) != string(content) {
			t.Errorf("expected %q, got %q", content, read)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/test/file.txt", []byte("content"))

		if _, err := fs.Stat("/test/file.txt"); err != nil {
			t.Error("file should exist before remove")
		}

		err := fs.Remove("/test/file.txt")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		if _, err := fs.Stat("/test/file.txt"); err == nil {
			t.Error("file should not exist after remove")
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		fs := NewMockFileSystem()
		fs.AddFile("/test/dir/file1.txt", []byte("1"))
		fs.AddFile("/test/dir/file2.txt", []byte("2"))
		fs.AddFile("/test/dir/subdir/file3.txt", []byte("3"))
		fs.AddFile("/test/other/file.txt", []byte("other"))

		err := fs.RemoveAll("/test/dir")
		if err != nil {
			t.Fatalf("RemoveAll failed: %v", err)
		}

		if _, err := fs.Stat("/test/dir/file1.txt"); err == nil {
			t.Error("file1 should be removed")
		}
		if _, err := fs.Stat("/test/dir/subdir/file3.txt"); err == nil {
			t.Error("file3 should be removed")
		}
		if _, err := fs.Stat("/test/other/file.txt"); err != nil {
			t.Error("other file should still exist")
		}
	})
}
