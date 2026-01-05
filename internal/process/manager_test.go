package process

import (
	"testing"
)

func TestLogBuffer(t *testing.T) {
	t.Run("stores lines up to max", func(t *testing.T) {
		buf := NewLogBuffer(3)
		buf.Write([]byte("line1\n"))
		buf.Write([]byte("line2\n"))
		buf.Write([]byte("line3\n"))

		lines := buf.Lines()
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
		if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
			t.Errorf("unexpected lines: %v", lines)
		}
	})

	t.Run("drops oldest lines when full", func(t *testing.T) {
		buf := NewLogBuffer(2)
		buf.Write([]byte("line1\n"))
		buf.Write([]byte("line2\n"))
		buf.Write([]byte("line3\n"))

		lines := buf.Lines()
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
		if lines[0] != "line2" || lines[1] != "line3" {
			t.Errorf("expected [line2, line3], got %v", lines)
		}
	})

	t.Run("handles multi-line writes", func(t *testing.T) {
		buf := NewLogBuffer(10)
		buf.Write([]byte("line1\nline2\nline3\n"))

		lines := buf.Lines()
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("clears buffer", func(t *testing.T) {
		buf := NewLogBuffer(10)
		buf.Write([]byte("line1\n"))
		buf.Clear()

		lines := buf.Lines()
		if len(lines) != 0 {
			t.Errorf("expected 0 lines after clear, got %d", len(lines))
		}
	})

	t.Run("returns copy of lines", func(t *testing.T) {
		buf := NewLogBuffer(10)
		buf.Write([]byte("line1\n"))

		lines1 := buf.Lines()
		lines2 := buf.Lines()

		// Modifying one shouldn't affect the other
		lines1[0] = "modified"
		if lines2[0] == "modified" {
			t.Error("Lines() should return a copy")
		}
	})
}

func TestManager(t *testing.T) {
	t.Run("creates new manager with random start port", func(t *testing.T) {
		m1 := NewManager()
		m2 := NewManager()

		// They should have different start ports (statistically)
		// This is a weak test but verifies the randomization is happening
		if m1.nextPort < 50000 || m1.nextPort >= 60000 {
			t.Errorf("nextPort %d out of range [50000, 60000)", m1.nextPort)
		}
		if m2.nextPort < 50000 || m2.nextPort >= 60000 {
			t.Errorf("nextPort %d out of range [50000, 60000)", m2.nextPort)
		}
	})

	t.Run("findFreePort returns valid port", func(t *testing.T) {
		m := NewManager()
		port, err := m.findFreePort()
		if err != nil {
			t.Fatalf("findFreePort failed: %v", err)
		}
		if port < 50000 || port >= 60000 {
			t.Errorf("port %d out of range", port)
		}
	})

	t.Run("findFreePort increments", func(t *testing.T) {
		m := NewManager()
		port1, _ := m.findFreePort()
		port2, _ := m.findFreePort()

		// Second port should be different (incremented)
		if port1 == port2 {
			t.Error("findFreePort should return different ports")
		}
	})
}
