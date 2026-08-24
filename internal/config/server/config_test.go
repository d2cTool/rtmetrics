package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearServerEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CONFIG", "ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "STORE_FILE",
		"RESTORE", "DATABASE_DSN", "KEY", "CRYPTO_KEY", "AUDIT_FILE", "AUDIT_URL",
		"DATABASE_MAX_OPEN_CONNS", "DATABASE_MAX_IDLE_CONNS",
		"DATABASE_CONN_MAX_IDLE_TIME", "DATABASE_CONN_MAX_LIFETIME", "DATABASE_PING_TIMEOUT",
	} {
		t.Setenv(k, "")
		require.NoError(t, os.Unsetenv(k))
	}
}

func writeServerJSON(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestParse_Defaults(t *testing.T) {
	clearServerEnv(t)

	cfg, err := Parse(nil)
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.HTTPServer.Address)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "./tmp/data", cfg.FileStoragePath)
	assert.False(t, cfg.Restore)
	assert.Empty(t, cfg.DatabaseDSN)
	assert.Empty(t, cfg.CryptoKey)
	assert.Empty(t, cfg.Key)
	assert.Equal(t, 10, cfg.Database.MaxOpenConns)
}

func TestParse_JSONFile(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{
		"address": "127.0.0.1:9090",
		"restore": true,
		"store_interval": "1s",
		"store_file": "/path/to/file.db",
		"database_dsn": "postgres://u:p@localhost/db",
		"crypto_key": "/path/to/key.pem",
		"key": "secret",
		"audit_file": "/tmp/audit.log",
		"audit_url": "http://audit.local",
		"database_max_open_conns": 7,
		"database_conn_max_idle_time": "2m"
	}`)

	cfg, err := Parse([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.HTTPServer.Address)
	assert.True(t, cfg.Restore)
	assert.Equal(t, 1, cfg.StoreInterval)
	assert.Equal(t, "/path/to/file.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://u:p@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "/path/to/key.pem", cfg.CryptoKey)
	assert.Equal(t, "secret", cfg.Key)
	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.local", cfg.AuditURL)
	assert.Equal(t, 7, cfg.Database.MaxOpenConns)
	assert.Equal(t, 2*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestParse_ConfigLongFlag(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{"address":"from-file:1"}`)

	cfg, err := Parse([]string{"-config", path})
	require.NoError(t, err)
	assert.Equal(t, "from-file:1", cfg.HTTPServer.Address)
}

func TestParse_ConfigEnv(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{"address":"from-config-env:1"}`)
	t.Setenv("CONFIG", path)

	cfg, err := Parse(nil)
	require.NoError(t, err)
	assert.Equal(t, "from-config-env:1", cfg.HTTPServer.Address)
}

func TestParse_ConfigEnvOverridesFlagPath(t *testing.T) {
	clearServerEnv(t)
	flagPath := writeServerJSON(t, `{"address":"from-flag-file:1"}`)
	envPath := writeServerJSON(t, `{"address":"from-env-file:1"}`)
	t.Setenv("CONFIG", envPath)

	cfg, err := Parse([]string{"-c", flagPath})
	require.NoError(t, err)
	assert.Equal(t, "from-env-file:1", cfg.HTTPServer.Address)
}

func TestParse_FlagOverridesJSON(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{
		"address": "file:8080",
		"restore": true,
		"store_interval": "5s",
		"store_file": "/from/file.db",
		"database_dsn": "postgres://file",
		"crypto_key": "/from/file.pem"
	}`)

	cfg, err := Parse([]string{
		"-c", path,
		"-a", "flag:9090",
		"-r=false",
		"-i", "42",
		"-f", "/from/flag.db",
		"-d", "postgres://flag",
		"-crypto-key", "/from/flag.pem",
	})
	require.NoError(t, err)
	assert.Equal(t, "flag:9090", cfg.HTTPServer.Address)
	assert.False(t, cfg.Restore)
	assert.Equal(t, 42, cfg.StoreInterval)
	assert.Equal(t, "/from/flag.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://flag", cfg.DatabaseDSN)
	assert.Equal(t, "/from/flag.pem", cfg.CryptoKey)
}

func TestParse_EnvOverridesJSONAndFlags(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{
		"address": "file:8080",
		"store_interval": "5s",
		"store_file": "/from/file.db"
	}`)
	t.Setenv("ADDRESS", "env:7070")
	t.Setenv("STORE_INTERVAL", "99")
	t.Setenv("FILE_STORAGE_PATH", "/from/env.db")

	cfg, err := Parse([]string{"-c", path, "-a", "flag:9090", "-i", "42", "-f", "/from/flag.db"})
	require.NoError(t, err)
	assert.Equal(t, "env:7070", cfg.HTTPServer.Address)
	assert.Equal(t, 99, cfg.StoreInterval)
	assert.Equal(t, "/from/env.db", cfg.FileStoragePath)
}

func TestParse_StoreFileEnvAlias(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("STORE_FILE", "/from/store-file.db")

	cfg, err := Parse(nil)
	require.NoError(t, err)
	assert.Equal(t, "/from/store-file.db", cfg.FileStoragePath)
}

func TestParse_PartialJSONKeepsDefaults(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{"crypto_key":"/only/key.pem"}`)

	cfg, err := Parse([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.HTTPServer.Address)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "./tmp/data", cfg.FileStoragePath)
	assert.Equal(t, "/only/key.pem", cfg.CryptoKey)
}

func TestParse_MissingConfigFile(t *testing.T) {
	clearServerEnv(t)
	_, err := Parse([]string{"-c", filepath.Join(t.TempDir(), "missing.json")})
	require.Error(t, err)
}

func TestParse_InvalidStoreInterval(t *testing.T) {
	clearServerEnv(t)
	path := writeServerJSON(t, `{"store_interval":"not-a-duration"}`)
	_, err := Parse([]string{"-c", path})
	require.Error(t, err)
}
