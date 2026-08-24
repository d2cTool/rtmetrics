//go:build windows

package common

import (
	"os"
	"syscall"
)

// ShutdownSignals — сигналы штатной остановки. SIGQUIT в Windows нет.
func ShutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}
