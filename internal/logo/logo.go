package logo

import "strings"

// Base logo with no leading spaces on first line
const base = `    _____
   / __(_)_______  __  ______
  / /_/ / ___/ _ \/ / / / __ \
 / __/ / /  /  __/ /_/ / /_/ /
/_/ /_/_/   \___/\__,_/ .___/
                     /_/`

// Get returns the logo with specified leading spaces on the first line
func Get(indent int) string {
	return strings.Repeat(" ", indent) + base
}

// CLI returns logo formatted for terminal output (26 spaces)
func CLI() string {
	return Get(26)
}

// Web returns logo formatted for web/HTML output (17 spaces)
func Web() string {
	return Get(17)
}
