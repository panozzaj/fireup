//go:build !darwin || !cgo

package server

import (
	"errors"
	"net"
)

// launchdListeners is only available on macOS; elsewhere fireup binds its
// ports itself.
func launchdListeners(name string) ([]net.Listener, error) {
	return nil, errors.New("launchd sockets are only available on macOS")
}
