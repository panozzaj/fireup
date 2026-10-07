package main

import (
	"strings"
	"testing"
)

// launchd binds HTTPS on loopback for fireup, so a port held on another
// address (e.g. by Tailscale Funnel) doesn't keep *.test off HTTPS
func TestServicePlistAsksLaunchdForLoopbackHTTPSSockets(t *testing.T) {
	plist, err := generateServicePlistContent()
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"<key>Sockets</key>",
		"<key>HTTPS</key>",
		"<string>127.0.0.1</string>",
		"<string>::1</string>",
		"<key>SockServiceName</key>\n                <string>443</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Errorf("plist missing %q:\n%s", want, plist)
		}
	}
}
