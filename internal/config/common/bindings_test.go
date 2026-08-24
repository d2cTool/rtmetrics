package common

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings_VisitedOverridesZeroAndUnvisitedKeepsDest(t *testing.T) {
	address := "localhost:8080"
	interval := 300
	restore := false
	idle := 60 * time.Second

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	b := NewBindings()
	b.String(fs, "a", address, "", &address)
	b.Int(fs, "i", interval, "", &interval)
	b.Bool(fs, "r", restore, "", &restore)
	b.Duration(fs, "idle", idle, "", &idle)

	require.NoError(t, fs.Parse([]string{"-a", "flag:9090", "-r=false", "-idle", "0s"}))

	// как после JSON: файл уже записал значения, непройденные флаги их не трогают
	address = "from-file:1"
	interval = 5
	restore = true
	idle = time.Minute

	b.Apply(fs)

	assert.Equal(t, "flag:9090", address)
	assert.Equal(t, 5, interval)
	assert.False(t, restore)
	assert.Equal(t, time.Duration(0), idle)
}

func TestApplyVisited_NilMap(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	require.NoError(t, fs.Parse(nil))
	ApplyVisited(fs, nil)
}
