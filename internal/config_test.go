package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("GoodCase_loads_repos_and_expands_env", func(t *testing.T) {
		t.Setenv("DV_TEST_TOKEN", "secret-token")

		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.yaml")
		content := `repos:
  - name: github.com/example/one
    repo_url: https://github.com/example/one
    token: ${DV_TEST_TOKEN}
  - name: github.com/example/two
    repo_url: https://github.com/example/two
`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		cfg, err := LoadConfig(path)
		require.NoError(t, err)
		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "github.com/example/one", cfg.Repos[0].Name)
		assert.Equal(t, "https://github.com/example/one", cfg.Repos[0].RepoURL)
		assert.Equal(t, "secret-token", cfg.Repos[0].Token)
		assert.Equal(t, "github.com/example/two", cfg.Repos[1].Name)
		assert.Empty(t, cfg.Repos[1].Token)
	})

	t.Run("BadCase_file_not_found", func(t *testing.T) {
		_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
		assert.Error(t, err)
	})

	t.Run("BadCase_invalid_yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.yaml")
		require.NoError(t, os.WriteFile(path, []byte("repos: [invalid: ["), 0o644))

		_, err := LoadConfig(path)
		assert.Error(t, err)
	})
}
