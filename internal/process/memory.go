package process

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetProcessTreeRSS returns the total RSS in bytes for a process and all its
// descendants. Uses ps and pgrep (available on macOS and Linux).
func GetProcessTreeRSS(pid int) (int64, error) {
	if pid <= 0 {
		return 0, nil
	}

	var total int64

	// Get RSS for this process (ps returns KB)
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		// Process may have exited
		return 0, nil
	}
	if rssKB, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
		total += rssKB * 1024
	}

	// Find child processes
	out, err = exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		// No children or pgrep failed — that's fine
		return total, nil
	}

	for _, pidStr := range strings.Fields(string(out)) {
		childPid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		childRSS, _ := GetProcessTreeRSS(childPid)
		total += childRSS
	}

	return total, nil
}

// FormatBytes formats bytes into a human-readable string (e.g. "1.2 GB", "342 MB").
func FormatBytes(bytes int64) string {
	const (
		gb = 1024 * 1024 * 1024
		mb = 1024 * 1024
	)
	switch {
	case bytes >= gb:
		return strconv.FormatFloat(float64(bytes)/float64(gb), 'f', 1, 64) + " GB"
	default:
		return strconv.FormatInt(bytes/mb, 10) + " MB"
	}
}
