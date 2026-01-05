package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const asciiLogo = `
    ___  ___  ___  ____ _____      ___  ____ _  _
    |__| |  | |  | [__   |   ____ |  \ |___ |  |
    |  \ |__| |__| ___]  |        |__/ |___  \/
`

// ReverseProxy handles proxying requests to backend services
type ReverseProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// NewReverseProxy creates a new reverse proxy to the given port
func NewReverseProxy(port int) *ReverseProxy {
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

	// Handle errors gracefully
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("%s\nProxy error: %v\n", asciiLogo, err), http.StatusBadGateway)
	}

	return &ReverseProxy{
		target: target,
		proxy:  proxy,
	}
}

// ServeHTTP implements http.Handler
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Store original host for the backend
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-Proto", "http")
	if r.TLS != nil {
		r.Header.Set("X-Forwarded-Proto", "https")
	}

	p.proxy.ServeHTTP(w, r)
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
