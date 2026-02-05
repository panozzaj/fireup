package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAppFromCwd(t *testing.T) {
	// Save and restore original cwd
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Override getDefaultConfigDir for testing
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	// Create ~/.config/fireup in the temp home
	testConfigDir := filepath.Join(tmpHome, ".config", "fireup")
	os.MkdirAll(testConfigDir, 0755)

	t.Run("exact match on app Dir", func(t *testing.T) {
		appDir := t.TempDir()
		yaml := "name: myapp\nroot: " + appDir + "\ncmd: npm start\n"
		os.WriteFile(filepath.Join(testConfigDir, "myapp.yml"), []byte(yaml), 0644)
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "myapp.yml")) })

		os.Chdir(appDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find app")
		}
		if name != "myapp" {
			t.Errorf("expected 'myapp', got %q", name)
		}
	})

	t.Run("subdirectory match", func(t *testing.T) {
		appDir := t.TempDir()
		subDir := filepath.Join(appDir, "src", "components")
		os.MkdirAll(subDir, 0755)

		yaml := "name: subapp\nroot: " + appDir + "\ncmd: npm start\n"
		os.WriteFile(filepath.Join(testConfigDir, "subapp.yml"), []byte(yaml), 0644)
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "subapp.yml")) })

		os.Chdir(subDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find app from subdirectory")
		}
		if name != "subapp" {
			t.Errorf("expected 'subapp', got %q", name)
		}
	})

	t.Run("no match", func(t *testing.T) {
		noMatchDir := t.TempDir()
		os.Chdir(noMatchDir)
		_, found := resolveAppFromCwd()
		if found {
			t.Error("expected no match for unrelated directory")
		}
	})

	t.Run("most specific wins", func(t *testing.T) {
		parentDir := t.TempDir()
		childDir := filepath.Join(parentDir, "child-project")
		os.MkdirAll(childDir, 0755)

		yaml1 := "name: parent\nroot: " + parentDir + "\ncmd: npm start\n"
		yaml2 := "name: child\nroot: " + childDir + "\ncmd: npm start\n"
		os.WriteFile(filepath.Join(testConfigDir, "parent.yml"), []byte(yaml1), 0644)
		os.WriteFile(filepath.Join(testConfigDir, "child.yml"), []byte(yaml2), 0644)
		t.Cleanup(func() {
			os.Remove(filepath.Join(testConfigDir, "parent.yml"))
			os.Remove(filepath.Join(testConfigDir, "child.yml"))
		})

		os.Chdir(childDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find app")
		}
		if name != "child" {
			t.Errorf("expected 'child' (most specific), got %q", name)
		}
	})

	t.Run("FilePath match for static app", func(t *testing.T) {
		staticDir := t.TempDir()
		os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html></html>"), 0644)

		// Create a symlink config pointing to the static dir
		os.Symlink(staticDir, filepath.Join(testConfigDir, "staticapp"))
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "staticapp")) })

		os.Chdir(staticDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find static app via FilePath")
		}
		if name != "staticapp" {
			t.Errorf("expected 'staticapp', got %q", name)
		}
	})

	t.Run("service dir match", func(t *testing.T) {
		rootDir := t.TempDir()
		svcDir := filepath.Join(rootDir, "backend")
		os.MkdirAll(svcDir, 0755)

		yaml := "name: multiapp\nroot: " + rootDir + "\nservices:\n  frontend:\n    cmd: npm start\n  backend:\n    dir: backend\n    cmd: rails s\n"
		os.WriteFile(filepath.Join(testConfigDir, "multiapp.yml"), []byte(yaml), 0644)
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "multiapp.yml")) })

		os.Chdir(svcDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find app from service dir")
		}
		if name != "multiapp" {
			t.Errorf("expected 'multiapp', got %q", name)
		}
	})

	t.Run("port proxy apps without Dir are skipped", func(t *testing.T) {
		// Create a port-only config (no Dir)
		os.WriteFile(filepath.Join(testConfigDir, "portonly"), []byte("3000"), 0644)
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "portonly")) })

		noMatchDir := t.TempDir()
		os.Chdir(noMatchDir)
		name, found := resolveAppFromCwd()
		// Should not crash, and should not match
		if found && name == "portonly" {
			t.Error("port-only app should not match any directory")
		}
	})

	t.Run("symlink resolution", func(t *testing.T) {
		realDir := t.TempDir()
		linkDir := filepath.Join(t.TempDir(), "link")
		os.Symlink(realDir, linkDir)

		yaml := "name: linkapp\nroot: " + realDir + "\ncmd: npm start\n"
		os.WriteFile(filepath.Join(testConfigDir, "linkapp.yml"), []byte(yaml), 0644)
		t.Cleanup(func() { os.Remove(filepath.Join(testConfigDir, "linkapp.yml")) })

		// cd into symlinked path
		os.Chdir(linkDir)
		name, found := resolveAppFromCwd()
		if !found {
			t.Fatal("expected to find app via symlink")
		}
		if name != "linkapp" {
			t.Errorf("expected 'linkapp', got %q", name)
		}
	})
}

func TestIsSubdir(t *testing.T) {
	t.Run("child is subdirectory", func(t *testing.T) {
		if !isSubdir("/a/b/c", "/a/b") {
			t.Error("expected /a/b/c to be subdir of /a/b")
		}
	})

	t.Run("same directory is not subdir", func(t *testing.T) {
		if isSubdir("/a/b", "/a/b") {
			t.Error("/a/b should not be subdir of /a/b (it's equal)")
		}
	})

	t.Run("parent is not subdir of child", func(t *testing.T) {
		if isSubdir("/a", "/a/b") {
			t.Error("/a should not be subdir of /a/b")
		}
	})

	t.Run("prefix but not subdir", func(t *testing.T) {
		if isSubdir("/a/bar", "/a/b") {
			t.Error("/a/bar should not be subdir of /a/b (prefix but not path boundary)")
		}
	})
}

func TestPrintCurrentProjectSection(t *testing.T) {
	// Save and restore
	origDir, _ := os.Getwd()
	origHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Chdir(origDir)
		os.Setenv("HOME", origHome)
	})

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	testConfigDir := filepath.Join(tmpHome, ".config", "fireup")
	os.MkdirAll(testConfigDir, 0755)

	// Write a global config for TLD
	os.WriteFile(filepath.Join(testConfigDir, "config.json"), []byte(`{"tld":"test"}`), 0644)

	appDir := t.TempDir()
	yaml := "name: helpapp\nroot: " + appDir + "\ncmd: npm start\n"
	os.WriteFile(filepath.Join(testConfigDir, "helpapp.yml"), []byte(yaml), 0644)

	os.Chdir(appDir)

	var sb strings.Builder
	printCurrentProjectSection(&sb)
	output := sb.String()

	if !strings.Contains(output, "CURRENT PROJECT") {
		t.Error("expected output to contain CURRENT PROJECT section")
	}
	if !strings.Contains(output, "helpapp") {
		t.Errorf("expected output to contain app name 'helpapp', got:\n%s", output)
	}
	if !strings.Contains(output, "fireup restart") {
		t.Error("expected output to contain shortcut commands")
	}
}

func TestPrintCurrentProjectSectionNoMatch(t *testing.T) {
	origDir, _ := os.Getwd()
	origHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Chdir(origDir)
		os.Setenv("HOME", origHome)
	})

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	testConfigDir := filepath.Join(tmpHome, ".config", "fireup")
	os.MkdirAll(testConfigDir, 0755)

	noMatchDir := t.TempDir()
	os.Chdir(noMatchDir)

	var sb strings.Builder
	printCurrentProjectSection(&sb)
	output := sb.String()

	if output != "" {
		t.Errorf("expected empty output when no match, got:\n%s", output)
	}
}
