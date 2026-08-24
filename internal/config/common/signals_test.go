package common

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShutdownSignalsIncludeINTAndTERM(t *testing.T) {
	t.Parallel()

	got := ShutdownSignals()
	assert.Contains(t, got, syscall.SIGINT)
	assert.Contains(t, got, syscall.SIGTERM)
}
