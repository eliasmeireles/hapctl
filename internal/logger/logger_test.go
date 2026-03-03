package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("must initialize with custom path", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		err := Init(logPath)
		require.NoError(t, err)

		Info("test log message")

		require.FileExists(t, logPath)
	})

	t.Run("must use fallback when default path is not accessible", func(t *testing.T) {
		err := Init("/root/restricted/hapctl.log")

		if os.Geteuid() != 0 {
			require.NoError(t, err)

			homeDir, _ := os.UserHomeDir()
			fallbackPath := filepath.Join(homeDir, FallbackLogPath)
			fallbackDir := filepath.Dir(fallbackPath)

			_, err := os.Stat(fallbackDir)
			require.NoError(t, err)
		}
	})
}

func TestLogMonitoring(t *testing.T) {
	t.Run("must log monitoring report to custom path", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "monitoring.log")

		report := map[string]string{"status": "healthy"}
		err := LogMonitoring(logPath, report)
		require.NoError(t, err)

		require.FileExists(t, logPath)
	})

	t.Run("must use fallback when default path is not accessible", func(t *testing.T) {
		report := map[string]string{"status": "healthy"}
		err := LogMonitoring("/root/restricted/monitoring.log", report)

		if os.Geteuid() != 0 {
			require.NoError(t, err)
		}
	})
}
