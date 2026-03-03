package haproxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandExists(t *testing.T) {
	installer := NewInstaller()

	t.Run("must detect existing command", func(t *testing.T) {
		exists := installer.commandExists("ls")
		require.True(t, exists)
	})

	t.Run("must not detect non-existing command", func(t *testing.T) {
		exists := installer.commandExists("nonexistentcommand12345")
		require.False(t, exists)
	})
}

func TestIsInstalled(t *testing.T) {
	installer := NewInstaller()

	t.Run("must check if haproxy is installed", func(t *testing.T) {
		isInstalled := installer.IsInstalled()
		require.IsType(t, false, isInstalled)
	})
}

func TestGetVersion(t *testing.T) {
	installer := NewInstaller()

	t.Run("must return error when haproxy not installed", func(t *testing.T) {
		if !installer.IsInstalled() {
			_, err := installer.GetVersion()
			require.Error(t, err)
		}
	})

	t.Run("must return version when haproxy is installed", func(t *testing.T) {
		if installer.IsInstalled() {
			version, err := installer.GetVersion()
			require.NoError(t, err)
			require.NotEmpty(t, version)
			require.Contains(t, version, "HAProxy")
		}
	})
}
