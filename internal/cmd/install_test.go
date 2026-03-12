package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAndFixPermissions(t *testing.T) {
	t.Run("must fix incorrect file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test content"), 0600)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		err = checkAndFixPermissions(testFile, 0644)
		require.NoError(t, err)

		info, err = os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("must not change correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		err = checkAndFixPermissions(testFile, 0644)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("must handle non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "nonexistent.txt")

		err := checkAndFixPermissions(testFile, 0644)
		require.NoError(t, err)
	})
}

func TestEnsureDirectoryPermissions(t *testing.T) {
	t.Run("must fix incorrect directory permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "testdir")

		err := os.Mkdir(testDir, 0700)
		require.NoError(t, err)

		info, err := os.Stat(testDir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

		err = ensureDirectoryPermissions(testDir, 0755)
		require.NoError(t, err)

		info, err = os.Stat(testDir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})

	t.Run("must not change correct directory permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "testdir")

		err := os.Mkdir(testDir, 0755)
		require.NoError(t, err)

		err = ensureDirectoryPermissions(testDir, 0755)
		require.NoError(t, err)

		info, err := os.Stat(testDir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})

	t.Run("must handle non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "nonexistent")

		err := ensureDirectoryPermissions(testDir, 0755)
		require.NoError(t, err)
	})

	t.Run("must return error for file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		err = ensureDirectoryPermissions(testFile, 0755)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a directory")
	})
}
