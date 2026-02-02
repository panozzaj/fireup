package logo

import "strings"

// Base logo - relative spacing between lines is intentional for F shape
const base = `  _____  __
 / ____\|__|_______  ____  __ __ _____
 |  __\ |  |\_  __ \/ __ \|  |  \\ __ \
 |  |   |  | |  | \/  ___/|  |  /| |_| |
 |__|   |__| |__|   \___/ |____/ |  __/
                                 |_|`

// Get returns the logo with specified leading spaces on each line
func Get(indent int) string {
	prefix := strings.Repeat(" ", indent)
	lines := strings.Split(base, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// CLI returns logo formatted for terminal output
func CLI() string {
	return Get(4)
}

// Web returns logo formatted for web/HTML output with non-breaking spaces
func Web() string {
	lines := strings.Split(base, "\n")
	for i, line := range lines {
		// Count leading spaces and replace with &nbsp;
		trimmed := strings.TrimLeft(line, " ")
		leadingSpaces := len(line) - len(trimmed)
		lines[i] = strings.Repeat("&nbsp;", leadingSpaces) + trimmed
	}
	return strings.Join(lines, "\n")
}
