package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed examples/*.md
var examplesFS embed.FS

func cmdExamples(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`fireup examples - Framework config examples

USAGE:
    fireup examples              List available frameworks
    fireup examples <framework>  Show config example for a framework`)
			os.Exit(0)
		}
	}

	if len(args) == 0 {
		listExamples()
		return
	}

	showExample(args[0])
}

func listExamples() {
	entries, err := examplesFS.ReadDir("examples")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading examples")
		os.Exit(1)
	}

	fmt.Println("Available framework examples:")
	fmt.Println()
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		fmt.Printf("  fireup examples %s\n", name)
	}
}

func showExample(name string) {
	filename := name + ".md"
	data, err := examplesFS.ReadFile("examples/" + filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No example for %q. Run 'fireup examples' to see available frameworks.\n", name)
		os.Exit(1)
	}

	fmt.Print(renderMarkdown(string(data)))
}

func renderMarkdown(s string) string {
	var out strings.Builder
	lines := strings.Split(s, "\n")
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				out.WriteString(colorDim)
			} else {
				out.WriteString(colorReset)
			}
			continue
		}

		if inCodeBlock {
			out.WriteString("  " + line + "\n")
			continue
		}

		if strings.HasPrefix(line, "# ") {
			out.WriteString(fmt.Sprintf("\n%s%s%s\n", colorCyan, line[2:], colorReset))
		} else if strings.HasPrefix(line, "## ") {
			out.WriteString(fmt.Sprintf("\n%s%s%s\n", colorYellow, line[3:], colorReset))
		} else {
			out.WriteString(line + "\n")
		}
	}

	if inCodeBlock {
		out.WriteString(colorReset)
	}

	return out.String()
}
