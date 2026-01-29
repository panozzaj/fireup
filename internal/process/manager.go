package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// getUserShell returns the current user's default shell.
// On macOS, it uses dscl to query the DirectoryService.
// Falls back to SHELL env var, then /bin/zsh on macOS or /bin/bash elsewhere.
func getUserShell() string {
	// Try SHELL env var first
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// On macOS, query DirectoryService for the user's shell
	if runtime.GOOS == "darwin" {
		if u, err := user.Current(); err == nil {
			out, err := exec.Command("dscl", ".", "-read", "/Users/"+u.Username, "UserShell").Output()
			if err == nil {
				// Output is "UserShell: /bin/zsh"
				if parts := strings.SplitN(strings.TrimSpace(string(out)), ": ", 2); len(parts) == 2 {
					return parts[1]
				}
			}
		}
		return "/bin/zsh" // macOS default
	}

	return "/bin/bash" // Linux/other default
}

// Process represents a running process
type Process struct {
	Name    string
	Command string
	Dir     string
	Port    int
	Env     map[string]string

	cmd       *exec.Cmd
	cancel    context.CancelFunc
	logs      *LogBuffer
	started   time.Time
	starting  bool // true while waiting for port to be ready
	failed    bool
	exitError string
	mu        sync.Mutex
}

// LogBuffer stores recent log output
type LogBuffer struct {
	mu    sync.RWMutex
	lines []string
	max   int
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(maxLines int) *LogBuffer {
	return &LogBuffer{
		lines: make([]string, 0, maxLines),
		max:   maxLines,
	}
}

// Write implements io.Writer
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Filter out shell warnings that aren't relevant to users
		if strings.Contains(line, "can't change option: zle") {
			continue
		}
		b.lines = append(b.lines, line)
		if len(b.lines) > b.max {
			b.lines = b.lines[1:]
		}
	}
	return len(p), nil
}

// Lines returns all stored log lines
func (b *LogBuffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	return result
}

// Clear clears the log buffer
func (b *LogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = b.lines[:0]
}

// Manager manages running processes
type Manager struct {
	mu            sync.RWMutex
	processes     map[string]*Process
	reservedPorts map[int]bool // ports allocated but not yet bound
	portStart     int
	portEnd       int
	nextPort      int
}

// NewManager creates a new process manager
func NewManager() *Manager {
	portStart := 50000
	portEnd := 60000
	// Start from a random port to avoid conflicts with orphaned processes
	nextPort := portStart + int(time.Now().UnixNano()%int64(portEnd-portStart))
	return &Manager{
		processes:     make(map[string]*Process),
		reservedPorts: make(map[int]bool),
		portStart:     portStart,
		portEnd:       portEnd,
		nextPort:      nextPort,
	}
}

// findFreePort finds an available port and reserves it.
// Caller must call releasePort if the port won't be used.
func (m *Manager) findFreePort() (int, error) {
	for i := 0; i < m.portEnd-m.portStart; i++ {
		port := m.nextPort
		m.nextPort++
		if m.nextPort >= m.portEnd {
			m.nextPort = m.portStart
		}

		// Skip ports we've already reserved for other processes
		if m.reservedPorts[port] {
			fmt.Printf("[roost-dev] Port %d is reserved, skipping\n", port)
			continue
		}

		// First check if anything is already LISTENING on this port
		// This catches processes bound to 0.0.0.0 that wouldn't block our 127.0.0.1 bind
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			fmt.Printf("[roost-dev] Port %d has something listening, skipping\n", port)
			continue
		}

		// Also check 0.0.0.0 binding to be thorough
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("0.0.0.0:%d", port), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			fmt.Printf("[roost-dev] Port %d has something listening on 0.0.0.0, skipping\n", port)
			continue
		}

		// Now check if we can bind to it
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			// Reserve this port until the process binds or fails
			m.reservedPorts[port] = true
			fmt.Printf("[roost-dev] Allocated and reserved port %d\n", port)
			return port, nil
		}
		fmt.Printf("[roost-dev] Port %d bind failed: %v\n", port, err)
	}
	return 0, fmt.Errorf("no free ports available in range %d-%d", m.portStart, m.portEnd)
}

// releasePort removes a port reservation
func (m *Manager) releasePort(port int) {
	fmt.Printf("[roost-dev] Released port reservation: %d\n", port)
	delete(m.reservedPorts, port)
}

// Start starts a process
func (m *Manager) Start(name, command, dir string, env map[string]string) (*Process, error) {
	m.mu.Lock()

	// Check if already running
	if p, exists := m.processes[name]; exists && p.IsRunning() {
		m.mu.Unlock()
		return p, nil
	}

	// Clean up stale Rails PID file if this looks like a Rails server
	if strings.Contains(command, "rails server") || strings.Contains(command, "rails s") {
		cleanupRailsPID(dir)
	}

	// Check if working directory exists
	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			m.mu.Unlock()
			return nil, fmt.Errorf("working directory does not exist: %s", dir)
		}
	}

	// Find a free port
	port, err := m.findFreePort()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	fmt.Printf("[roost-dev] Starting %s on port %d\n", name, port)

	// Create process
	ctx, cancel := context.WithCancel(context.Background())

	// Build environment
	procEnv := os.Environ()
	procEnv = append(procEnv, fmt.Sprintf("PORT=%d", port))
	procEnv = append(procEnv, "FORCE_COLOR=1")
	portStr := fmt.Sprintf("%d", port)
	for k, v := range env {
		// Expand $PORT in env values
		v = strings.ReplaceAll(v, "$PORT", portStr)
		procEnv = append(procEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Parse command (handle shell execution)
	// Use interactive login shell to ensure user's environment (rvm, rbenv, nvm, etc.) is loaded
	// -l (login) sources .zprofile; -i (interactive) sources .zshrc/.bashrc
	shell := getUserShell()
	cmd := exec.CommandContext(ctx, shell, "-i", "-l", "-c", command)
	cmd.Dir = dir
	cmd.Env = procEnv
	// Run in own process group so we can kill the entire tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set up logging
	logs := NewLogBuffer(1000)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	proc := &Process{
		Name:    name,
		Command: command,
		Dir:     dir,
		Port:    port,
		Env:     env,
		cmd:     cmd,
		cancel:  cancel,
		logs:    logs,
		started: time.Now(),
	}

	// Start process
	if err := cmd.Start(); err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("start process: %w", err)
	}

	// Stream logs
	go streamLogs(stdout, logs, name)
	go streamLogs(stderr, logs, name)

	// Monitor for exit
	go func() {
		err := cmd.Wait()
		// Write log BEFORE setting failed flag to avoid race condition
		// where status shows "failed" but logs are empty
		if err != nil {
			proc.logs.Write([]byte("[roost-dev] Process exited\n"))
		}
		proc.mu.Lock()
		if err != nil {
			proc.failed = true
			if exitErr, ok := err.(*exec.ExitError); ok {
				proc.exitError = fmt.Sprintf("exit code %d", exitErr.ExitCode())
			} else {
				proc.exitError = err.Error()
			}
		}
		proc.mu.Unlock()
		// Don't delete failed processes so we can show their status
		// They'll be replaced if started again
	}()

	proc.starting = true
	m.processes[name] = proc

	// Release lock BEFORE waiting for port - this can take a while and would block all requests
	m.mu.Unlock()

	// Wait for port to be ready (keep checking until port ready or process exits)
	go func() {
		// Release port reservation when done (process bound or exited)
		defer func() {
			m.mu.Lock()
			m.releasePort(port)
			m.mu.Unlock()
		}()

		for {
			// Check if process has exited
			if proc.cmd.ProcessState != nil {
				proc.mu.Lock()
				proc.starting = false
				proc.mu.Unlock()
				return
			}

			// Check if port is ready
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				proc.mu.Lock()
				proc.starting = false
				proc.mu.Unlock()
				return
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Wait up to 30s for initial startup, then return
	// Process stays in "starting" state until port is actually ready
	waitForPort(port, 30*time.Second)

	return proc, nil
}

// StartAsync starts a process without waiting for the port to be ready.
// Returns immediately after the process is spawned.
func (m *Manager) StartAsync(name, command, dir string, env map[string]string) (*Process, error) {
	m.mu.Lock()

	// Check if already running or starting
	if p, exists := m.processes[name]; exists && (p.IsRunning() || p.IsStarting()) {
		m.mu.Unlock()
		return p, nil
	}

	// Clean up stale Rails PID file if this looks like a Rails server
	if strings.Contains(command, "rails server") || strings.Contains(command, "rails s") {
		cleanupRailsPID(dir)
	}

	// Check if working directory exists
	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			m.mu.Unlock()
			return nil, fmt.Errorf("working directory does not exist: %s", dir)
		}
	}

	// Find a free port
	port, err := m.findFreePort()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	fmt.Printf("[roost-dev] Starting %s on port %d\n", name, port)

	// Create process
	ctx, cancel := context.WithCancel(context.Background())

	// Build environment
	procEnv := os.Environ()
	procEnv = append(procEnv, fmt.Sprintf("PORT=%d", port))
	procEnv = append(procEnv, "FORCE_COLOR=1")
	portStr := fmt.Sprintf("%d", port)
	for k, v := range env {
		// Expand $PORT in env values
		v = strings.ReplaceAll(v, "$PORT", portStr)
		procEnv = append(procEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Parse command (handle shell execution)
	// Use interactive login shell to ensure user's environment (rvm, rbenv, nvm, etc.) is loaded
	// -l (login) sources .zprofile; -i (interactive) sources .zshrc/.bashrc
	shell := getUserShell()
	cmd := exec.CommandContext(ctx, shell, "-i", "-l", "-c", command)
	cmd.Dir = dir
	cmd.Env = procEnv
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set up logging
	logs := NewLogBuffer(1000)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	proc := &Process{
		Name:    name,
		Command: command,
		Dir:     dir,
		Port:    port,
		Env:     env,
		cmd:     cmd,
		cancel:  cancel,
		logs:    logs,
		started: time.Now(),
	}

	// Start process
	if err := cmd.Start(); err != nil {
		cancel()
		m.releasePort(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("start process: %w", err)
	}

	// Stream logs
	go streamLogs(stdout, logs, name)
	go streamLogs(stderr, logs, name)

	// Monitor for exit
	go func() {
		err := cmd.Wait()
		// Write log BEFORE setting failed flag to avoid race condition
		// where status shows "failed" but logs are empty
		if err != nil {
			proc.logs.Write([]byte("[roost-dev] Process exited\n"))
		}
		proc.mu.Lock()
		if err != nil {
			proc.failed = true
			if exitErr, ok := err.(*exec.ExitError); ok {
				proc.exitError = fmt.Sprintf("exit code %d", exitErr.ExitCode())
			} else {
				proc.exitError = err.Error()
			}
		}
		proc.mu.Unlock()
	}()

	proc.starting = true
	m.processes[name] = proc
	m.mu.Unlock()

	// Wait for port in background (keep checking until port ready or process exits)
	go func() {
		for {
			// Check if process has exited
			if proc.cmd.ProcessState != nil {
				proc.mu.Lock()
				proc.starting = false
				proc.mu.Unlock()
				return
			}

			// Check if port is ready
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				proc.mu.Lock()
				proc.starting = false
				proc.mu.Unlock()
				return
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	return proc, nil
}

// streamLogs reads from a reader and writes to the log buffer
func streamLogs(r io.Reader, logs *LogBuffer, name string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		logs.Write([]byte(line + "\n"))
		// Also print to stdout for debugging
		fmt.Printf("[%s] %s\n", name, line)
	}
}

// cleanupRailsPID removes Rails PID files and kills orphaned processes
func cleanupRailsPID(dir string) {
	pidFile := filepath.Join(dir, "tmp", "pids", "server.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return // No PID file, nothing to clean up
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Invalid PID file, remove it
		os.Remove(pidFile)
		fmt.Printf("[roost-dev] Removed invalid PID file: %s\n", pidFile)
		return
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		fmt.Printf("[roost-dev] Removed stale PID file (pid %d): %s\n", pid, pidFile)
		return
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, remove the stale PID file
		os.Remove(pidFile)
		fmt.Printf("[roost-dev] Removed stale PID file (pid %d not running): %s\n", pid, pidFile)
		return
	}

	// Process IS running - this is likely an orphaned process from a previous roost-dev
	// Kill it so we can start fresh
	fmt.Printf("[roost-dev] Killing orphaned Rails process (pid %d)\n", pid)
	process.Signal(syscall.SIGTERM)
	time.Sleep(100 * time.Millisecond)
	process.Signal(syscall.SIGKILL)
	os.Remove(pidFile)
}

// waitForPort waits for a port to become available
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}

// Stop stops a process
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc, exists := m.processes[name]
	if !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	proc.Kill()
	delete(m.processes, name)
	return nil
}

// Kill terminates the process and all its children
func (p *Process) Kill() {
	p.mu.Lock()
	var pid int
	var pgid int
	var hasPid bool
	var hasPgid bool
	if p.cmd != nil && p.cmd.Process != nil {
		pid = p.cmd.Process.Pid
		hasPid = true
		var err error
		pgid, err = syscall.Getpgid(pid)
		if err != nil {
			fmt.Printf("[roost-dev] Kill %s: Getpgid(%d) failed: %v\n", p.Name, pid, err)
		} else {
			hasPgid = true
		}
	}
	p.mu.Unlock()

	if hasPgid {
		// Kill the entire process group
		fmt.Printf("[roost-dev] Kill %s: sending SIGTERM to process group -%d\n", p.Name, pgid)
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			fmt.Printf("[roost-dev] Kill %s: SIGTERM to group -%d failed: %v\n", p.Name, pgid, err)
		}
		// Give it a moment to clean up, then force kill
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("[roost-dev] Kill %s: sending SIGKILL to process group -%d\n", p.Name, pgid)
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
			fmt.Printf("[roost-dev] Kill %s: SIGKILL to group -%d failed: %v\n", p.Name, pgid, err)
		}
	} else if hasPid {
		// Fallback: kill by PID if we couldn't get PGID
		fmt.Printf("[roost-dev] Kill %s: falling back to PID kill for %d\n", p.Name, pid)
		syscall.Kill(pid, syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(pid, syscall.SIGKILL)
	}

	// Also kill any child processes we can find (belt and suspenders)
	if hasPid {
		killChildProcesses(pid, p.Name)
	}

	p.cancel()
}

// killChildProcesses finds and kills all child processes of the given PID
func killChildProcesses(parentPid int, name string) {
	// Use pgrep to find children (works on macOS and Linux)
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(parentPid))
	output, err := cmd.Output()
	if err != nil {
		// No children found or pgrep failed - that's fine
		return
	}

	childPids := strings.Fields(string(output))
	for _, pidStr := range childPids {
		childPid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		fmt.Printf("[roost-dev] Kill %s: killing orphaned child PID %d\n", name, childPid)
		// Recursively kill grandchildren first
		killChildProcesses(childPid, name)
		// Then kill this child
		syscall.Kill(childPid, syscall.SIGTERM)
		time.Sleep(50 * time.Millisecond)
		syscall.Kill(childPid, syscall.SIGKILL)
	}
}

// Restart restarts a process
func (m *Manager) Restart(name string) (*Process, error) {
	m.mu.RLock()
	proc, exists := m.processes[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process not found: %s", name)
	}

	// Capture config before stopping
	command := proc.Command
	dir := proc.Dir
	env := proc.Env

	// Stop
	m.Stop(name)

	// Brief wait for port release
	time.Sleep(100 * time.Millisecond)

	// Start again
	return m.Start(name, command, dir, env)
}

// RestartAsync restarts a process without blocking
func (m *Manager) RestartAsync(name string) {
	m.mu.RLock()
	proc, exists := m.processes[name]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Capture config before stopping
	command := proc.Command
	dir := proc.Dir
	env := proc.Env

	// Stop
	m.Stop(name)

	// Start again asynchronously after brief delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.Start(name, command, dir, env)
	}()
}

// Get returns a process by name
func (m *Manager) Get(name string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	proc, ok := m.processes[name]
	return proc, ok
}

// All returns all running processes
func (m *Manager) All() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	procs := make([]*Process, 0, len(m.processes))
	for _, proc := range m.processes {
		procs = append(procs, proc)
	}
	return procs
}

// StopAll stops all running processes
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.processes) == 0 {
		fmt.Println("[roost-dev] StopAll: no processes to stop")
		return
	}

	fmt.Printf("[roost-dev] StopAll: stopping %d processes\n", len(m.processes))
	for name, proc := range m.processes {
		fmt.Printf("[roost-dev] StopAll: stopping %s\n", name)
		proc.Kill()
		delete(m.processes, name)
	}
	fmt.Println("[roost-dev] StopAll: all processes stopped")
}

// IsRunning returns true if the process is still running
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// Still starting (waiting for port) - not "running" yet
	if p.starting {
		return false
	}

	// Check if process has exited
	if p.cmd.ProcessState != nil {
		return false
	}

	fmt.Printf("[roost-dev] IsRunning: %s = true (port %d)\n", p.Name, p.Port)
	return true
}

// Logs returns the log buffer
func (p *Process) Logs() *LogBuffer {
	return p.logs
}

// Uptime returns how long the process has been running
func (p *Process) Uptime() time.Duration {
	return time.Since(p.started)
}

// HasFailed returns true if the process exited with an error
func (p *Process) HasFailed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.failed
}

// IsStarting returns true if the process is starting but port not yet ready
func (p *Process) IsStarting() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.starting && !p.failed
}

// ExitError returns the exit error message if the process failed
func (p *Process) ExitError() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitError
}
