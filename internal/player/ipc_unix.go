//go:build !windows

package player

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func ipcPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("subwave-tune-%d.sock", os.Getpid()))
}

func dialIPC(path string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", path, timeout)
}

func cleanupIPC(path string) {
	_ = os.Remove(path)
}
