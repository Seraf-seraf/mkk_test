package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	require.FailNow(t, "repo root not found")
	return ""
}

func TestLoadDefault(t *testing.T) {
	const methodCtx = "config.TestLoadDefault"

	root := findRepoRoot(t)
	oldWD, err := os.Getwd()
	require.NoError(t, err, methodCtx)
	defer func() { _ = os.Chdir(oldWD) }()

	require.NoError(t, os.Chdir(root), methodCtx)

	cfg, err := LoadDefault()
	require.NoError(t, err, methodCtx)
	require.NotNil(t, cfg, methodCtx)
	require.Equal(t, 8080, cfg.Server.Port, methodCtx)
	require.Equal(t, "mysql", cfg.MySQL.Host, methodCtx)
}

func TestLoadEmptyPathUsesDefault(t *testing.T) {
	const methodCtx = "config.TestLoadEmptyPathUsesDefault"

	root := findRepoRoot(t)
	oldWD, err := os.Getwd()
	require.NoError(t, err, methodCtx)
	defer func() { _ = os.Chdir(oldWD) }()

	require.NoError(t, os.Chdir(root), methodCtx)

	cfg, err := Load("")
	require.NoError(t, err, methodCtx)
	require.NotNil(t, cfg, methodCtx)
}

func TestLoadMissingFile(t *testing.T) {
	const methodCtx = "config.TestLoadMissingFile"

	root := findRepoRoot(t)
	oldWD, err := os.Getwd()
	require.NoError(t, err, methodCtx)
	defer func() { _ = os.Chdir(oldWD) }()

	require.NoError(t, os.Chdir(root), methodCtx)

	_, err = Load("configs/missing.yml")
	require.Error(t, err, methodCtx)
}
