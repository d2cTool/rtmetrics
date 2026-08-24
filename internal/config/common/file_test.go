package common

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPath_FlagOnly(t *testing.T) {
	t.Setenv("CONFIG", "")
	require.NoError(t, os.Unsetenv("CONFIG"))

	assert.Equal(t, "/tmp/cfg.json", ResolveConfigPath("/tmp/cfg.json"))
	assert.Empty(t, ResolveConfigPath(""))
}

func TestResolveConfigPath_EnvOverridesFlag(t *testing.T) {
	t.Setenv("CONFIG", "/from/env.json")
	assert.Equal(t, "/from/env.json", ResolveConfigPath("/from/flag.json"))
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"address":"127.0.0.1:9090"}`), 0o600))

	var got struct {
		Address string `json:"address"`
	}
	require.NoError(t, LoadJSON(path, &got))
	assert.Equal(t, "127.0.0.1:9090", got.Address)
}

func TestLoadJSON_MissingFile(t *testing.T) {
	var dest struct{}
	err := LoadJSON(filepath.Join(t.TempDir(), "missing.json"), &dest)
	require.Error(t, err)
}

func TestLoadJSON_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	var dest struct{}
	err := LoadJSON(path, &dest)
	require.Error(t, err)
}

func TestVisitedFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var a, b string
	fs.StringVar(&a, "a", "", "")
	fs.StringVar(&b, "b", "", "")
	require.NoError(t, fs.Parse([]string{"-a", "x"}))

	visited := VisitedFlags(fs)
	assert.True(t, FlagPassed(visited, "a"))
	assert.False(t, FlagPassed(visited, "b"))
	assert.True(t, FlagPassed(visited, "b", "a"))
}

func TestParseIntervalSeconds(t *testing.T) {
	sec, err := ParseIntervalSeconds("1s")
	require.NoError(t, err)
	assert.Equal(t, 1, sec)

	sec, err = ParseIntervalSeconds("2m")
	require.NoError(t, err)
	assert.Equal(t, 120, sec)

	sec, err = ParseIntervalSeconds("300")
	require.NoError(t, err)
	assert.Equal(t, 300, sec)

	_, err = ParseIntervalSeconds("")
	require.Error(t, err)

	_, err = ParseIntervalSeconds("foo")
	require.Error(t, err)
}
