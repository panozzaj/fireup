package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReverseProxy handles proxying requests to backend services
type ReverseProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// NewReverseProxy creates a new reverse proxy to the given port
func NewReverseProxy(port int, theme string) *ReverseProxy {
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Preserve the original Host header
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Keep original host header so apps can detect subdomains
		req.Host = req.Header.Get("X-Forwarded-Host")
		if req.Host == "" {
			req.Host = req.URL.Host
		}
	}

	// Handle errors gracefully with a styled page that auto-retries
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Connection Error</title>
    <script>
        (function() {
            var theme = '%s';
            if (theme && theme !== 'system') {
                document.documentElement.setAttribute('data-theme', theme);
            }
        })();
    </script>
    <style>
        :root { --bg: #1a1a2e; --text: #eee; --muted: #9ca3af; --border: #374151; }
        @media (prefers-color-scheme: light) {
            :root:not([data-theme="dark"]) { --bg: #f5f5f5; --text: #1a1a1a; --muted: #6b7280; --border: #e5e7eb; }
        }
        [data-theme="light"] { --bg: #f5f5f5; --text: #1a1a1a; --muted: #6b7280; --border: #e5e7eb; }
        [data-theme="dark"] { --bg: #1a1a2e; --text: #eee; --muted: #9ca3af; --border: #374151; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--bg); color: var(--text); margin: 0; padding: 60px 40px; min-height: 100vh; display: flex; flex-direction: column; align-items: center; }
        .container { text-align: center; max-width: 500px; }
        h1 { font-size: 24px; margin: 0 0 16px; }
        .message { color: var(--muted); margin-bottom: 24px; }
        .spinner { width: 40px; height: 40px; border: 3px solid var(--border); border-top-color: #f59e0b; border-radius: 50%%%%; animation: spin 1s linear infinite; margin: 0 auto; }
        @keyframes spin { to { transform: rotate(360deg); } }
    </style>
</head>
<body>
    <div class="container">
        <h1>Connecting...</h1>
        <p class="message">The service isn't responding. Retrying automatically...</p>
        <div class="spinner"></div>
    </div>
    <script>setTimeout(() => location.reload(), 1000);</script>
</body>
</html>`, theme)
	}

	// Add cache-busting headers to prevent browser from caching proxied responses
	// This ensures users see the interstitial when services restart
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Only modify HTML responses (the main document)
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "text/html") {
			resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			resp.Header.Set("Pragma", "no-cache")
			resp.Header.Set("Expires", "0")
		}
		return nil
	}

	return &ReverseProxy{
		target: target,
		proxy:  proxy,
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// ServeHTTP implements http.Handler
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Store original host for the backend
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-Proto", "http")
	if r.TLS != nil {
		r.Header.Set("X-Forwarded-Proto", "https")
	}

	// WebSocket requests need raw TCP relay since httputil.ReverseProxy
	// doesn't handle the Connection: Upgrade protocol
	if isWebSocketUpgrade(r) {
		p.serveWebSocket(w, r)
		return
	}

	p.proxy.ServeHTTP(w, r)
}

// serveWebSocket handles WebSocket upgrade requests by hijacking the client
// connection and relaying bytes bidirectionally to the backend
func (p *ReverseProxy) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	// Dial the backend
	backendAddr := p.target.Host
	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket hijack not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "WebSocket hijack failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Rewrite the request URL to point to the backend and forward it
	r.URL.Scheme = "http"
	r.URL.Host = p.target.Host
	r.Header.Set("Host", p.target.Host)
	if err := r.Write(backendConn); err != nil {
		return
	}

	// Flush any buffered data from the client
	if clientBuf.Reader.Buffered() > 0 {
		buffered := make([]byte, clientBuf.Reader.Buffered())
		if _, err := clientBuf.Read(buffered); err == nil {
			backendConn.Write(buffered)
		}
	}

	// Bidirectional relay
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()

	// Wait for either direction to finish, then close both
	<-done
}

// StaticHandler serves static files
type StaticHandler struct {
	path string
}

// NewStaticHandler creates a handler for serving static files
func NewStaticHandler(path string) *StaticHandler {
	return &StaticHandler{path: path}
}

// ServeHTTP implements http.Handler
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	info, err := os.Stat(h.path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		// Serve directory with http.FileServer
		fs := http.FileServer(http.Dir(h.path))
		fs.ServeHTTP(w, r)
		return
	}

	// Serve single file
	// If requesting root, serve the file
	if r.URL.Path == "/" {
		http.ServeFile(w, r, h.path)
		return
	}

	// For other paths, try to serve from same directory
	dir := filepath.Dir(h.path)
	requestedPath := filepath.Join(dir, r.URL.Path)

	// Security: ensure path is within directory
	if !strings.HasPrefix(requestedPath, dir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
		// Fall back to serving the main file (SPA support)
		http.ServeFile(w, r, h.path)
		return
	}

	http.ServeFile(w, r, requestedPath)
}
