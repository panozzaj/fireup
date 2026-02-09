package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestReverseProxy_HTTP(t *testing.T) {
	// Backend returns a simple response
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "hello from backend, path=%s", r.URL.Path)
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	// Proxy server
	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/test-path")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if string(body) != "hello from backend, path=/test-path" {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestReverseProxy_ForwardsHeaders(t *testing.T) {
	var receivedHost, receivedProto string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Header.Get("X-Forwarded-Host")
		receivedProto = r.Header.Get("X-Forwarded-Proto")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	req, _ := http.NewRequest("GET", proxy.URL+"/", nil)
	req.Host = "myapp.test"
	http.DefaultClient.Do(req)

	if receivedHost != "myapp.test" {
		t.Errorf("expected X-Forwarded-Host=myapp.test, got %q", receivedHost)
	}
	if receivedProto != "http" {
		t.Errorf("expected X-Forwarded-Proto=http, got %q", receivedProto)
	}
}

func TestReverseProxy_ErrorHandler(t *testing.T) {
	// Proxy to a port with nothing listening
	rp := NewReverseProxy(19999, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Connecting...") {
		t.Error("error page should contain 'Connecting...'")
	}
}

func TestReverseProxy_CacheBusting(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html>hello</html>")
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if !strings.Contains(cc, "no-store") {
		t.Errorf("expected Cache-Control to contain no-store, got %q", cc)
	}
}

func TestReverseProxy_NoCacheBustingForNonHTML(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/api")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if strings.Contains(cc, "no-store") {
		t.Errorf("non-HTML response should not have cache-busting, got %q", cc)
	}
}

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name       string
		upgrade    string
		connection string
		want       bool
	}{
		{"valid websocket", "websocket", "Upgrade", true},
		{"case insensitive upgrade", "WebSocket", "upgrade", true},
		{"connection with keep-alive", "websocket", "keep-alive, Upgrade", true},
		{"no upgrade header", "", "Upgrade", false},
		{"no connection header", "websocket", "", false},
		{"wrong upgrade value", "h2c", "Upgrade", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/ws", nil)
			if tt.upgrade != "" {
				r.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.connection != "" {
				r.Header.Set("Connection", tt.connection)
			}
			if got := isWebSocketUpgrade(r); got != tt.want {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReverseProxy_WebSocketRelay(t *testing.T) {
	// Backend: accept WebSocket upgrade, echo messages back
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketUpgrade(r) {
			http.Error(w, "expected websocket", 400)
			return
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", 500)
			return
		}
		conn, buf, err := hijacker.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send 101 Switching Protocols
		buf.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
		buf.WriteString("Upgrade: websocket\r\n")
		buf.WriteString("Connection: Upgrade\r\n")
		buf.WriteString("\r\n")
		buf.Flush()

		// Echo loop: read a line, send it back
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "CLOSE" {
				break
			}
			fmt.Fprintf(conn, "echo:%s\n", line)
		}
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	// Connect through the proxy with an upgrade request
	proxyURL, _ := url.Parse(proxy.URL)
	conn, err := net.DialTimeout("tcp", proxyURL.Host, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy failed: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send WebSocket upgrade request
	req := fmt.Sprintf("GET /ws/voice HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"+
		"Sec-WebSocket-Version: 13\r\n"+
		"\r\n", proxyURL.Host)
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Fatalf("write upgrade request failed: %v", err)
	}

	// Read 101 response
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status line failed: %v", err)
	}
	if !strings.Contains(statusLine, "101") {
		t.Fatalf("expected 101 Switching Protocols, got %q", strings.TrimSpace(statusLine))
	}

	// Read remaining headers until empty line
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers failed: %v", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	// Send a message through the relay
	fmt.Fprintf(conn, "hello-proxy\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read echo failed: %v", err)
	}
	if strings.TrimSpace(response) != "echo:hello-proxy" {
		t.Errorf("expected echo:hello-proxy, got %q", strings.TrimSpace(response))
	}

	// Send another message to confirm ongoing relay
	fmt.Fprintf(conn, "second-message\n")
	response, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read second echo failed: %v", err)
	}
	if strings.TrimSpace(response) != "echo:second-message" {
		t.Errorf("expected echo:second-message, got %q", strings.TrimSpace(response))
	}

	// Clean close
	fmt.Fprintf(conn, "CLOSE\n")
}

func TestReverseProxy_WebSocketBackendDown(t *testing.T) {
	// Proxy to a port with nothing listening — WebSocket upgrade should fail gracefully
	rp := NewReverseProxy(19999, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	proxyURL, _ := url.Parse(proxy.URL)
	conn, err := net.DialTimeout("tcp", proxyURL.Host, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy failed: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send WebSocket upgrade request to dead backend
	req := fmt.Sprintf("GET /ws HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"\r\n", proxyURL.Host)
	conn.Write([]byte(req))

	// Should get 502 Bad Gateway
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status failed: %v", err)
	}
	if !strings.Contains(statusLine, "502") {
		t.Errorf("expected 502 for dead backend, got %q", strings.TrimSpace(statusLine))
	}
}

func TestReverseProxy_RegularUpgradeNotIntercepted(t *testing.T) {
	// A non-websocket Upgrade header should go through normal proxy
	var receivedUpgrade string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUpgrade = r.Header.Get("Upgrade")
		w.WriteHeader(200)
	}))
	defer backend.Close()

	port := portFromURL(t, backend.URL)
	rp := NewReverseProxy(port, "dark")

	proxy := httptest.NewServer(rp)
	defer proxy.Close()

	req, _ := http.NewRequest("GET", proxy.URL+"/", nil)
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("Connection", "Upgrade")
	http.DefaultClient.Do(req)

	if receivedUpgrade != "h2c" {
		t.Errorf("non-websocket upgrade should pass through, got Upgrade=%q", receivedUpgrade)
	}
}

// portFromURL extracts the port number from an httptest server URL
func portFromURL(t *testing.T, rawURL string) int {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL %q: %v", rawURL, err)
	}
	_, portStr, _ := net.SplitHostPort(u.Host)
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}
