package server

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

type listenFunc func(network, addr string) (net.Listener, error)

// httpsListeners returns the sockets to serve HTTPS on, and a description of
// where they came from for the startup log.
//
// Under the LaunchAgent, launchd binds 127.0.0.1:443 and [::1]:443 for us (see
// the Sockets key in `fireup service install`). Otherwise fireup binds
// 0.0.0.0 itself, which macOS Mojave+ allows unprivileged for any port — but
// only while nothing else holds the port on a specific address. Tailscale
// Serve/Funnel holds :443 on the tailnet address, for one.
func httpsListeners(fromLaunchd func(string) ([]net.Listener, error), listenFn listenFunc, port int) ([]net.Listener, string, error) {
	listeners, launchdErr := fromLaunchd("HTTPS")
	if launchdErr == nil && len(listeners) > 0 {
		return listeners, "launchd sockets", nil
	}

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := listenFn("tcp", addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			err = fmt.Errorf("%w (another process holds port %d, possibly only on one address such as Tailscale Serve/Funnel; "+
				"run `fireup service install` so launchd binds 127.0.0.1:%d for fireup; launchd sockets unavailable: %v)",
				err, port, port, launchdErr)
		}
		return nil, "", err
	}
	return []net.Listener{ln}, addr, nil
}

// serveAll serves srv on every listener and returns the first error.
func serveAll(listeners []net.Listener, serve func(net.Listener) error) error {
	errs := make(chan error, len(listeners))
	for _, ln := range listeners {
		go func(ln net.Listener) { errs <- serve(ln) }(ln)
	}
	return <-errs
}
