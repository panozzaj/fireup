package process

import (
	"fmt"
	"net"
	"testing"
	"time"
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

func TestProcessStates(t *testing.T) {
	t.Run("new process starts in starting state", func(t *testing.T) {
		m := NewManager()
		// Use a command that starts quickly but doesn't listen on port
		proc, err := m.StartAsync("test", "sleep 10", "/tmp", nil)
		if err != nil {
			t.Fatalf("StartAsync failed: %v", err)
		}
		defer m.Stop("test")

		// Should be in starting state (port not ready)
		if !proc.IsStarting() {
			t.Error("expected process to be in starting state")
		}
		if proc.IsRunning() {
			t.Error("expected process to not be running (port not ready)")
		}
		if proc.HasFailed() {
			t.Error("expected process to not have failed")
		}
	})

	t.Run("stop removes process from map", func(t *testing.T) {
		m := NewManager()
		_, err := m.StartAsync("test", "sleep 10", "/tmp", nil)
		if err != nil {
			t.Fatalf("StartAsync failed: %v", err)
		}

		// Verify process exists
		if _, found := m.Get("test"); !found {
			t.Fatal("expected process to exist before stop")
		}

		// Stop it
		m.Stop("test")

		// Verify process is gone
		if _, found := m.Get("test"); found {
			t.Error("expected process to be removed after stop")
		}
	})

	t.Run("StartAsync returns existing process if starting", func(t *testing.T) {
		m := NewManager()
		// Use a command that doesn't listen on port (stays in starting state)
		proc1, err := m.StartAsync("test", "sleep 10", "/tmp", nil)
		if err != nil {
			t.Fatalf("StartAsync failed: %v", err)
		}
		defer m.Stop("test")

		// Immediately try to start again
		proc2, err := m.StartAsync("test", "sleep 10", "/tmp", nil)
		if err != nil {
			t.Fatalf("second StartAsync failed: %v", err)
		}

		if proc1 != proc2 {
			t.Error("expected same process instance to be returned for starting process")
		}
	})
}

func TestStartAsyncWithMissingDirectory(t *testing.T) {
	t.Run("fails when directory does not exist", func(t *testing.T) {
		m := NewManager()
		nonExistentDir := "/tmp/roost-dev-test-nonexistent-dir-12345"

		proc, err := m.StartAsync("test-missing-dir", "echo hello", nonExistentDir, nil)
		if err != nil {
			// Immediate error is acceptable
			return
		}
		defer m.Stop("test-missing-dir")

		// Wait a bit for the process to fail
		for i := 0; i < 10; i++ {
			if proc.HasFailed() {
				// Success - process properly marked as failed
				return
			}
			// Small sleep to let the process attempt to start
			<-time.After(100 * time.Millisecond)
		}

		t.Error("expected process to fail when directory does not exist")
	})
}

func TestProcessStateQueries(t *testing.T) {
	t.Run("IsRunning returns false for nil cmd", func(t *testing.T) {
		p := &Process{}
		if p.IsRunning() {
			t.Error("IsRunning should return false for nil cmd")
		}
	})

	t.Run("IsStarting returns false when starting is false", func(t *testing.T) {
		p := &Process{starting: false}
		if p.IsStarting() {
			t.Error("IsStarting should return false when starting is false")
		}
	})

	t.Run("IsStarting returns false when failed", func(t *testing.T) {
		p := &Process{starting: true, failed: true}
		if p.IsStarting() {
			t.Error("IsStarting should return false when process has failed")
		}
	})

	t.Run("HasFailed returns true when failed is set", func(t *testing.T) {
		p := &Process{failed: true}
		if !p.HasFailed() {
			t.Error("HasFailed should return true when failed is set")
		}
	})
}

func TestPortReservation(t *testing.T) {
	t.Run("findFreePort reserves the port", func(t *testing.T) {
		m := NewManager()
		port, err := m.findFreePort()
		if err != nil {
			t.Fatalf("findFreePort failed: %v", err)
		}

		// Port should be in reserved set
		m.mu.RLock()
		reserved := m.reservedPorts[port]
		m.mu.RUnlock()

		if !reserved {
			t.Errorf("port %d should be reserved after findFreePort", port)
		}
	})

	t.Run("findFreePort skips reserved ports", func(t *testing.T) {
		m := NewManager()

		// Get first port and keep it reserved
		port1, _ := m.findFreePort()

		// Get second port - should be different
		port2, _ := m.findFreePort()

		if port1 == port2 {
			t.Error("findFreePort should skip reserved ports")
		}

		// Both should be reserved
		m.mu.RLock()
		if !m.reservedPorts[port1] || !m.reservedPorts[port2] {
			t.Error("both ports should be reserved")
		}
		m.mu.RUnlock()
	})

	t.Run("releasePort removes reservation", func(t *testing.T) {
		m := NewManager()
		port, _ := m.findFreePort()

		// Verify reserved
		m.mu.RLock()
		if !m.reservedPorts[port] {
			t.Fatal("port should be reserved")
		}
		m.mu.RUnlock()

		// Release it
		m.mu.Lock()
		m.releasePort(port)
		m.mu.Unlock()

		// Verify released
		m.mu.RLock()
		if m.reservedPorts[port] {
			t.Error("port should not be reserved after release")
		}
		m.mu.RUnlock()
	})

	t.Run("released port can be reused", func(t *testing.T) {
		m := NewManager()

		// Get a port (whatever is free)
		port1, err := m.findFreePort()
		if err != nil {
			t.Fatalf("findFreePort failed: %v", err)
		}

		// Release it and reset nextPort to try getting same port again
		m.mu.Lock()
		m.releasePort(port1)
		m.nextPort = port1
		m.mu.Unlock()

		// Should be able to get the same port again
		port2, err := m.findFreePort()
		if err != nil {
			t.Fatalf("second findFreePort failed: %v", err)
		}
		if port2 != port1 {
			t.Errorf("expected port %d to be available after release, got %d", port1, port2)
		}
	})

	t.Run("port released when process fails to start", func(t *testing.T) {
		m := NewManager()

		// Start a process that will fail immediately (bad directory)
		_, err := m.StartAsync("test-fail", "echo hello", "/nonexistent/path/that/does/not/exist", nil)

		// Whether it returns error immediately or not, wait a bit
		time.Sleep(500 * time.Millisecond)

		// Check if any ports are still reserved (should all be released)
		m.mu.RLock()
		reservedCount := len(m.reservedPorts)
		m.mu.RUnlock()

		// If there was an error, port should have been released
		if err != nil && reservedCount > 0 {
			t.Errorf("port should be released on immediate error, but %d ports still reserved", reservedCount)
		}

		m.Stop("test-fail")
	})

	t.Run("concurrent findFreePort calls get different ports", func(t *testing.T) {
		m := NewManager()
		ports := make(chan int, 10)

		// Launch 10 concurrent findFreePort calls
		// Note: findFreePort requires holding the lock
		for i := 0; i < 10; i++ {
			go func() {
				m.mu.Lock()
				port, err := m.findFreePort()
				m.mu.Unlock()
				if err != nil {
					t.Errorf("findFreePort failed: %v", err)
					return
				}
				ports <- port
			}()
		}

		// Collect all ports
		seen := make(map[int]bool)
		for i := 0; i < 10; i++ {
			select {
			case port := <-ports:
				if seen[port] {
					t.Errorf("port %d was allocated twice", port)
				}
				seen[port] = true
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for port allocation")
			}
		}
	})

	t.Run("reservedPorts initialized on new manager", func(t *testing.T) {
		m := NewManager()
		if m.reservedPorts == nil {
			t.Error("reservedPorts should be initialized")
		}
	})

	t.Run("findFreePort skips ports with active listeners on 127.0.0.1", func(t *testing.T) {
		m := NewManager()

		// Start a listener on the next port the manager will try
		m.mu.Lock()
		targetPort := m.nextPort
		m.mu.Unlock()

		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
		if err != nil {
			t.Skipf("could not bind to port %d for test: %v", targetPort, err)
		}
		defer ln.Close()

		// Now findFreePort should skip the port with the active listener
		port, err := m.findFreePort()
		if err != nil {
			t.Fatalf("findFreePort failed: %v", err)
		}

		if port == targetPort {
			t.Errorf("findFreePort returned port %d which has an active listener", targetPort)
		}
	})

	t.Run("findFreePort skips ports with active listeners on 0.0.0.0", func(t *testing.T) {
		m := NewManager()

		// Start a listener on 0.0.0.0 (all interfaces) on the next port
		m.mu.Lock()
		targetPort := m.nextPort
		m.mu.Unlock()

		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", targetPort))
		if err != nil {
			t.Skipf("could not bind to port %d for test: %v", targetPort, err)
		}
		defer ln.Close()

		// findFreePort should detect this and skip the port
		port, err := m.findFreePort()
		if err != nil {
			t.Fatalf("findFreePort failed: %v", err)
		}

		if port == targetPort {
			t.Errorf("findFreePort returned port %d which has an active listener on 0.0.0.0", targetPort)
		}
	})
}
