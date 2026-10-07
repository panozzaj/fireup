//go:build darwin && cgo

package server

/*
#include <launch.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// launchdListeners returns the sockets launchd bound for `name` under the
// Sockets key of fireup's LaunchAgent plist. launchd binds them as root, so
// fireup can serve a privileged port on a specific address (127.0.0.1:443),
// which macOS otherwise only allows unprivileged processes on 0.0.0.0.
//
// Fails with ESRCH when fireup wasn't started by launchd, and ENOENT when the
// plist has no socket by that name (e.g. installed by an older fireup).
func launchdListeners(name string) ([]net.Listener, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var fds *C.int
	var count C.size_t
	if rc := C.launch_activate_socket(cName, &fds, &count); rc != 0 {
		return nil, syscall.Errno(rc)
	}
	defer C.free(unsafe.Pointer(fds))

	var listeners []net.Listener
	for _, fd := range unsafe.Slice(fds, int(count)) {
		file := os.NewFile(uintptr(fd), name)
		ln, err := net.FileListener(file)
		file.Close() // FileListener dups the descriptor
		if err != nil {
			for _, l := range listeners {
				l.Close()
			}
			return nil, fmt.Errorf("launchd socket %s: %w", name, err)
		}
		listeners = append(listeners, ln)
	}
	return listeners, nil
}
