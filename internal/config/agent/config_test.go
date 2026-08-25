package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CONFIG", "ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL", "KEY", "RATE_LIMIT", "CRYPTO_KEY", "GRPC_ADDRESS", "GRPC_CERT",
	} {
		t.Setenv(k, "")
		require.NoError(t, os.Unsetenv(k))
	}
}

func writeAgentJSON(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestParse_Defaults(t *testing.T) {
	clearAgentEnv(t)

	cfg, err := Parse(nil)
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 1, cfg.RateLimit)
	assert.Empty(t, cfg.Key)
	assert.Empty(t, cfg.CryptoKey)
	assert.Empty(t, cfg.GRPCAddress)
}

func TestParse_JSONFile(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{
		"address": "127.0.0.1:9090",
		"report_interval": "1s",
		"poll_interval": "2s",
		"crypto_key": "/path/to/key.pem",
		"key": "secret",
		"rate_limit": 4,
		"grpc_address": "localhost:3200"
	}`)

	cfg, err := Parse([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.Address)
	assert.Equal(t, 1, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, "/path/to/key.pem", cfg.CryptoKey)
	assert.Equal(t, "secret", cfg.Key)
	assert.Equal(t, 4, cfg.RateLimit)
	assert.Equal(t, "localhost:3200", cfg.GRPCAddress)
}

func TestParse_ConfigLongFlag(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{"address":"from-file:1"}`)

	cfg, err := Parse([]string{"-config", path})
	require.NoError(t, err)
	assert.Equal(t, "from-file:1", cfg.Address)
}

func TestParse_ConfigEnv(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{"poll_interval":"7s"}`)
	t.Setenv("CONFIG", path)

	cfg, err := Parse(nil)
	require.NoError(t, err)
	assert.Equal(t, 7, cfg.PollInterval)
}

func TestParse_FlagOverridesJSON(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{
		"address": "file:8080",
		"report_interval": "5s",
		"poll_interval": "3s",
		"crypto_key": "/from/file.pem"
	}`)

	cfg, err := Parse([]string{
		"-c", path,
		"-a", "flag:9090",
		"-r", "20",
		"-p", "4",
		"-crypto-key", "/from/flag.pem",
	})
	require.NoError(t, err)
	assert.Equal(t, "flag:9090", cfg.Address)
	assert.Equal(t, 20, cfg.ReportInterval)
	assert.Equal(t, 4, cfg.PollInterval)
	assert.Equal(t, "/from/flag.pem", cfg.CryptoKey)
}

func TestParse_EnvOverridesJSONAndFlags(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{
		"address": "file:8080",
		"report_interval": "5s",
		"poll_interval": "3s"
	}`)
	t.Setenv("ADDRESS", "env:7070")
	t.Setenv("REPORT_INTERVAL", "30")
	t.Setenv("POLL_INTERVAL", "8")

	cfg, err := Parse([]string{"-c", path, "-a", "flag:9090", "-r", "20", "-p", "4"})
	require.NoError(t, err)
	assert.Equal(t, "env:7070", cfg.Address)
	assert.Equal(t, 30, cfg.ReportInterval)
	assert.Equal(t, 8, cfg.PollInterval)
}

func TestParse_GRPCAddressFlagAndEnv(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{"grpc_address":"file:3200"}`)

	cfg, err := Parse([]string{"-c", path, "-g", "flag:3300"})
	require.NoError(t, err)
	assert.Equal(t, "flag:3300", cfg.GRPCAddress)

	t.Setenv("GRPC_ADDRESS", "env:3400")
	cfg, err = Parse([]string{"-c", path, "-g", "flag:3300"})
	require.NoError(t, err)
	assert.Equal(t, "env:3400", cfg.GRPCAddress)
}

func TestParse_PartialJSONKeepsDefaults(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{"crypto_key":"/only/key.pem"}`)

	cfg, err := Parse([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, "/only/key.pem", cfg.CryptoKey)
}

func TestParse_MissingConfigFile(t *testing.T) {
	clearAgentEnv(t)
	_, err := Parse([]string{"-c", filepath.Join(t.TempDir(), "missing.json")})
	require.Error(t, err)
}

func TestParse_InvalidReportInterval(t *testing.T) {
	clearAgentEnv(t)
	path := writeAgentJSON(t, `{"report_interval":"nope"}`)
	_, err := Parse([]string{"-c", path})
	require.Error(t, err)
}
