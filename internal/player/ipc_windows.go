//go:build windows

package player

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
)

func ipcPath() string {
	return fmt.Sprintf(`\\.\pipe\subwave-tune-%d`, os.Getpid())
}

func dialIPC(path string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(path, &timeout)
}

func cleanupIPC(path string) {
	// Named pipe instances are removed when the last handle closes.
}
