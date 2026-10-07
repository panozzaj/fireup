package server

import (
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"
)

func testListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func closeAll(listeners []net.Listener) {
	for _, ln := range listeners {
		ln.Close()
	}
}

func TestHTTPSListenersPrefersLaunchdSockets(t *testing.T) {
	fromLaunchd := func(name string) ([]net.Listener, error) {
		if name != "HTTPS" {
			t.Fatalf("asked launchd for %q, want HTTPS", name)
		}
		return []net.Listener{testListener(t), testListener(t)}, nil
	}
	listenFn := func(network, addr string) (net.Listener, error) {
		t.Fatalf("bound %s itself despite launchd sockets", addr)
		return nil, nil
	}

	listeners, source, err := httpsListeners(fromLaunchd, listenFn, 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeAll(listeners)

	if len(listeners) != 2 || source != "launchd sockets" {
		t.Fatalf("got %d listeners from %q", len(listeners), source)
	}
}

// Run by hand, or from a plist installed by an older fireup
func TestHTTPSListenersBindsAllInterfacesWithoutLaunchd(t *testing.T) {
	fromLaunchd := func(string) ([]net.Listener, error) { return nil, syscall.ESRCH }
	var requested string
	listenFn := func(network, addr string) (net.Listener, error) {
		requested = addr
		return testListener(t), nil
	}

	listeners, source, err := httpsListeners(fromLaunchd, listenFn, 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeAll(listeners)

	if requested != "0.0.0.0:443" || source != "0.0.0.0:443" {
		t.Fatalf("bound %q (source %q), want 0.0.0.0:443", requested, source)
	}
}

func TestHTTPSListenersExplainsAPortHeldElsewhere(t *testing.T) {
	fromLaunchd := func(string) ([]net.Listener, error) { return nil, syscall.ENOENT }
	listenFn := func(network, addr string) (net.Listener, error) {
		return nil, &net.OpError{Op: "listen", Net: network, Err: syscall.EADDRINUSE}
	}

	_, _, err := httpsListeners(fromLaunchd, listenFn, 443)
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("expected EADDRINUSE, got %v", err)
	}
	if want := "fireup service install"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q should suggest %q", err, want)
	}
}
