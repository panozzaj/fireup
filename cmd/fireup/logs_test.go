package main

import (
	"bytes"
	"testing"
)

func newTestPrinter(colorize bool) (*logPrinter, *bytes.Buffer) {
	var buf bytes.Buffer
	return &logPrinter{
		colorize: colorize,
		colors:   make(map[string]string),
		w:        &buf,
	}, &buf
}

func TestLogPrinterNoColor(t *testing.T) {
	lp, buf := newTestPrinter(false)

	lp.Println("[web] some output")
	if got := buf.String(); got != "[web] some output\n" {
		t.Errorf("expected plain passthrough, got %q", got)
	}
}

func TestLogPrinterColorizePrefix(t *testing.T) {
	lp, buf := newTestPrinter(true)

	lp.Println("[web] server started")
	got := buf.String()

	// Should start with reset to clear any prior ANSI state
	if got[0:len(colorReset)] != colorReset {
		t.Error("expected line to start with colorReset")
	}

	// Should contain the prefix color, then [web], then reset, then content
	expected := colorReset + colorCyan + "[web]" + colorReset + " server started\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLogPrinterNoTrailingReset(t *testing.T) {
	lp, buf := newTestPrinter(true)

	// Simulate app output with unclosed bold
	lp.Println("[web] \033[1mbold text")
	got := buf.String()

	// Should NOT end with colorReset before the newline — app's bold flows through
	if got[len(got)-2:len(got)-1] == colorReset[len(colorReset)-1:] {
		// More precise check: the content after the second reset should be raw
		afterPrefix := colorReset + " \033[1mbold text\n"
		if got[len(got)-len(afterPrefix):] != afterPrefix {
			t.Errorf("trailing reset should not be added after content, got %q", got)
		}
	}

	// Verify no double-reset at end
	expected := colorReset + colorCyan + "[web]" + colorReset + " \033[1mbold text\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLogPrinterResetClearsLeakFromPriorLine(t *testing.T) {
	lp, buf := newTestPrinter(true)

	// Line 1: app leaves bold open
	lp.Println("[web] \033[1mbold start")
	// Line 2: prefix should still render cleanly
	lp.Println("[web] normal line")

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	// Line 2 (index 1) should start with reset
	line2 := string(lines[1])
	if line2[:len(colorReset)] != colorReset {
		t.Error("second line should start with reset to clear prior bold")
	}
}

func TestLogPrinterDistinctColorsPerPrefix(t *testing.T) {
	lp, buf := newTestPrinter(true)

	lp.Println("[web] line1")
	lp.Println("[assets] line2")
	lp.Println("[web] line3")

	lines := bytes.Split(buf.Bytes(), []byte("\n"))

	// web gets first color (cyan), assets gets second (magenta)
	line1 := string(lines[0])
	line2 := string(lines[1])
	line3 := string(lines[2])

	webExpected := colorReset + colorCyan + "[web]" + colorReset
	assetsExpected := colorReset + colorMagenta + "[assets]" + colorReset

	if line1[:len(webExpected)] != webExpected {
		t.Errorf("web should get cyan, got %q", line1)
	}
	if line2[:len(assetsExpected)] != assetsExpected {
		t.Errorf("assets should get magenta, got %q", line2)
	}
	// web should keep same color on repeat
	if line3[:len(webExpected)] != webExpected {
		t.Errorf("web should keep cyan on repeat, got %q", line3)
	}
}

func TestLogPrinterNoPrefixPassthrough(t *testing.T) {
	lp, buf := newTestPrinter(true)

	lp.Println("no prefix here")
	got := buf.String()

	// Lines without [prefix] should pass through unmodified
	if got != "no prefix here\n" {
		t.Errorf("expected plain passthrough for non-prefixed line, got %q", got)
	}
}

func TestLogPrinterBracketButNoSpace(t *testing.T) {
	lp, buf := newTestPrinter(true)

	// [timestamp] without trailing space+content pattern
	lp.Println("[notaprefix]no space after bracket")
	got := buf.String()

	// Should not be treated as a prefix (no "] " match)
	if got != "[notaprefix]no space after bracket\n" {
		t.Errorf("expected passthrough for malformed prefix, got %q", got)
	}
}
