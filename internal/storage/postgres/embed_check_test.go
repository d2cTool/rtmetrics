package postgres

import (
	"io/fs"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedMigrations_Available(t *testing.T) {
	info, err := fs.Stat(embedMigrations, "migrations")
	require.NoError(t, err, "директория migrations должна быть встроена")
	require.True(t, info.IsDir())

	entries, err := fs.ReadDir(embedMigrations, "migrations")
	require.NoError(t, err)
	require.NotEmpty(t, entries, "во встроенной FS должны быть файлы миграций")

	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("postgres"))
	migrations, err := goose.CollectMigrations("migrations", 0, 9999999999)
	require.NoError(t, err)
	require.NotEmpty(t, migrations)
}
