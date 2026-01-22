// Package diff provides utilities for showing file change previews and executing them.
package diff

import (
	"fmt"
	"os"
	"strings"
)

// FileOp represents a planned file operation
type FileOp struct {
	Path      string
	Operation string                 // "create", "delete"
	Content   func() (string, error) // Function that returns content (lazy evaluation)
}

// Plan represents a collection of planned file operations.
// The same Plan is used for both preview and execution, ensuring they stay in sync.
type Plan struct {
	ops []FileOp
}

// NewPlan creates a new Plan
func NewPlan() *Plan {
	return &Plan{}
}

// Create adds a file creation operation
func (p *Plan) Create(path string, content func() (string, error)) {
	p.ops = append(p.ops, FileOp{
		Path:      path,
		Operation: "create",
		Content:   content,
	})
}

// CreateStatic adds a file creation with static content
func (p *Plan) CreateStatic(path, content string) {
	p.Create(path, func() (string, error) { return content, nil })
}

// Delete adds a file deletion operation
func (p *Plan) Delete(path string) {
	p.ops = append(p.ops, FileOp{
		Path:      path,
		Operation: "delete",
	})
}

// HasChanges returns true if there are any operations
func (p *Plan) HasChanges() bool {
	return len(p.ops) > 0
}

// ChangeSummary represents a summary of planned changes
type ChangeSummary struct {
	Creates  []string // Files to create (new)
	Modifies []string // Files to modify (existing, different content)
	Deletes  []string // Files to delete
}

// Summary returns a summary of what will change, checking actual file contents.
// Returns nil if there are no actual changes.
func (p *Plan) Summary() *ChangeSummary {
	if !p.HasChanges() {
		return nil
	}

	summary := &ChangeSummary{}

	for _, op := range p.ops {
		existingBytes, err := os.ReadFile(op.Path)
		exists := err == nil

		switch op.Operation {
		case "create":
			newContent := ""
			if op.Content != nil {
				content, err := op.Content()
				if err == nil {
					newContent = content
				}
			}

			// Skip if content is identical
			if exists && string(existingBytes) == newContent {
				continue
			}

			if exists {
				summary.Modifies = append(summary.Modifies, op.Path)
			} else {
				summary.Creates = append(summary.Creates, op.Path)
			}

		case "delete":
			if exists {
				summary.Deletes = append(summary.Deletes, op.Path)
			}
		}
	}

	// Return nil if no actual changes
	if len(summary.Creates) == 0 && len(summary.Modifies) == 0 && len(summary.Deletes) == 0 {
		return nil
	}

	return summary
}

// Print displays the summary in a compact format
func (s *ChangeSummary) Print() {
	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	reset := "\033[0m"

	fmt.Println()
	for _, path := range s.Creates {
		fmt.Printf("  %s+%s %s (new)\n", green, reset, path)
	}
	for _, path := range s.Modifies {
		fmt.Printf("  %s~%s %s (modified)\n", yellow, reset, path)
	}
	for _, path := range s.Deletes {
		fmt.Printf("  %s-%s %s (delete)\n", red, reset, path)
	}
	fmt.Println()
}

// HasChanges returns true if there are any changes in the summary
func (s *ChangeSummary) HasChanges() bool {
	return len(s.Creates) > 0 || len(s.Modifies) > 0 || len(s.Deletes) > 0
}

// Preview displays the planned changes in unified diff format.
// Returns true if there are actual changes to show, false if all files are unchanged.
func (p *Plan) Preview() bool {
	if !p.HasChanges() {
		return false
	}

	cyan := "\033[36m"
	green := "\033[32m"
	red := "\033[31m"
	reset := "\033[0m"

	hasChanges := false
	var output strings.Builder

	for _, op := range p.ops {
		// Read existing file content if it exists
		existingContent := ""
		existingBytes, err := os.ReadFile(op.Path)
		exists := err == nil
		if exists {
			existingContent = string(existingBytes)
		}

		switch op.Operation {
		case "create":
			newContent := ""
			if op.Content != nil {
				content, err := op.Content()
				if err == nil {
					newContent = content
				}
			}

			// Skip if content is identical
			if exists && existingContent == newContent {
				continue
			}

			hasChanges = true

			// Show unified diff
			output.WriteString(fmt.Sprintf("%s--- %s%s\n", red, op.Path, reset))
			output.WriteString(fmt.Sprintf("%s+++ %s%s\n", green, op.Path, reset))

			if !exists {
				// New file - show all lines as additions
				output.WriteString(fmt.Sprintf("%s@@ -0,0 +1,%d @@%s\n", cyan, countLines(newContent), reset))
				for _, line := range strings.Split(newContent, "\n") {
					if line != "" || !strings.HasSuffix(newContent, "\n") {
						output.WriteString(fmt.Sprintf("%s+%s%s\n", green, line, reset))
					}
				}
			} else {
				// Existing file - show unified diff
				diff := unifiedDiff(existingContent, newContent)
				output.WriteString(diff)
			}
			output.WriteString("\n")

		case "delete":
			if exists {
				hasChanges = true
				output.WriteString(fmt.Sprintf("%s--- %s%s\n", red, op.Path, reset))
				output.WriteString(fmt.Sprintf("%s+++ /dev/null%s\n", green, reset))
				lines := strings.Split(existingContent, "\n")
				output.WriteString(fmt.Sprintf("%s@@ -1,%d +0,0 @@%s\n", cyan, len(lines), reset))
				for _, line := range lines {
					output.WriteString(fmt.Sprintf("%s-%s%s\n", red, line, reset))
				}
				output.WriteString("\n")
			}
		}
	}

	if hasChanges {
		fmt.Print(output.String())
	}

	return hasChanges
}

// unifiedDiff generates a unified diff between old and new content
func unifiedDiff(old, new string) string {
	cyan := "\033[36m"
	green := "\033[32m"
	red := "\033[31m"
	reset := "\033[0m"

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	// Simple line-by-line diff with context
	var result strings.Builder
	contextLines := 3

	// Find changed regions
	type hunk struct {
		oldStart, oldCount int
		newStart, newCount int
		lines              []string // prefixed with ' ', '+', or '-'
	}

	var hunks []hunk
	var currentHunk *hunk

	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			// Lines match - context line
			if currentHunk != nil {
				currentHunk.lines = append(currentHunk.lines, " "+oldLines[i])
				currentHunk.oldCount++
				currentHunk.newCount++
			}
			i++
			j++
		} else {
			// Lines differ - start or continue a hunk
			if currentHunk == nil {
				// Start new hunk with context
				start := max(0, i-contextLines)
				currentHunk = &hunk{
					oldStart: start + 1,
					newStart: max(0, j-contextLines) + 1,
				}
				// Add leading context
				for k := start; k < i; k++ {
					currentHunk.lines = append(currentHunk.lines, " "+oldLines[k])
					currentHunk.oldCount++
					currentHunk.newCount++
				}
			}

			// Find how many lines differ
			if i < len(oldLines) && (j >= len(newLines) || !containsAt(newLines, j, oldLines[i])) {
				// Line removed from old
				currentHunk.lines = append(currentHunk.lines, "-"+oldLines[i])
				currentHunk.oldCount++
				i++
			} else if j < len(newLines) {
				// Line added to new
				currentHunk.lines = append(currentHunk.lines, "+"+newLines[j])
				currentHunk.newCount++
				j++
			}
		}

		// Check if we should close the hunk (enough trailing context)
		if currentHunk != nil {
			trailingContext := 0
			for k := len(currentHunk.lines) - 1; k >= 0 && currentHunk.lines[k][0] == ' '; k-- {
				trailingContext++
			}
			if trailingContext >= contextLines && i < len(oldLines) {
				// Close hunk
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
			}
		}
	}

	// Close final hunk
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	// Format hunks
	for _, h := range hunks {
		result.WriteString(fmt.Sprintf("%s@@ -%d,%d +%d,%d @@%s\n",
			cyan, h.oldStart, h.oldCount, h.newStart, h.newCount, reset))
		for _, line := range h.lines {
			switch line[0] {
			case '+':
				result.WriteString(fmt.Sprintf("%s%s%s\n", green, line, reset))
			case '-':
				result.WriteString(fmt.Sprintf("%s%s%s\n", red, line, reset))
			default:
				result.WriteString(line + "\n")
			}
		}
	}

	return result.String()
}

func containsAt(lines []string, start int, target string) bool {
	for i := start; i < len(lines) && i < start+5; i++ {
		if lines[i] == target {
			return true
		}
	}
	return false
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Execute runs all the planned operations
func (p *Plan) Execute() error {
	for _, op := range p.ops {
		switch op.Operation {
		case "create":
			if op.Content != nil {
				content, err := op.Content()
				if err != nil {
					return fmt.Errorf("generating content for %s: %w", op.Path, err)
				}
				if err := os.WriteFile(op.Path, []byte(content), 0644); err != nil {
					return fmt.Errorf("writing %s: %w", op.Path, err)
				}
			}
		case "delete":
			if _, err := os.Stat(op.Path); err == nil {
				if err := os.Remove(op.Path); err != nil {
					return fmt.Errorf("removing %s: %w", op.Path, err)
				}
			}
		}
	}
	return nil
}

// Paths returns all file paths in the plan
func (p *Plan) Paths() []string {
	var paths []string
	for _, op := range p.ops {
		paths = append(paths, op.Path)
	}
	return paths
}
