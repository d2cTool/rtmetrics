//go:build unix

package common

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShutdownSignalsIncludeQUIT(t *testing.T) {
	t.Parallel()
	assert.Contains(t, ShutdownSignals(), syscall.SIGQUIT)
}
