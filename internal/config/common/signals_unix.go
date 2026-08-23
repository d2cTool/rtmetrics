//go:build unix

package common

import (
	"os"
	"syscall"
)

// ShutdownSignals — сигналы штатной остановки: SIGINT, SIGTERM, SIGQUIT.
func ShutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
}
