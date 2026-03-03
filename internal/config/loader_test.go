package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("must load valid config successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
sync:
  resource-path: /tmp/resources
  interval: 10s
  enabled: true
monitoring:
  enabled: true
  interval: 5s
  webhook:
    url: http://localhost:8080/webhook
    headers:
      - name: Content-Type
        value: application/json
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, "/tmp/resources", cfg.Sync.ResourcePath)
		require.Equal(t, 10*time.Second, cfg.Sync.Interval)
		require.True(t, cfg.Sync.Enabled)
		require.True(t, cfg.Monitoring.Enabled)
		require.Equal(t, 5*time.Second, cfg.Monitoring.Interval)
		require.NotNil(t, cfg.Monitoring.Webhook)
		require.Equal(t, "http://localhost:8080/webhook", cfg.Monitoring.Webhook.URL)
	})

	t.Run("must apply defaults when values are missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
sync:
  enabled: true
monitoring:
  enabled: false
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.Equal(t, DefaultResourcePath, cfg.Sync.ResourcePath)
		require.Equal(t, DefaultSyncInterval, cfg.Sync.Interval)
		require.Equal(t, DefaultMonitorInterval, cfg.Monitoring.Interval)
	})

	t.Run("must fail when config file does not exist", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/config.yaml")
		require.Error(t, err)
	})

	t.Run("must fail with invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath)
		require.Error(t, err)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("must fail when sync interval is too short", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
sync:
  resource-path: /tmp/resources
  interval: 500ms
  enabled: true
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "interval must be at least 1 second")
	})

	t.Run("must fail when webhook URL is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
sync:
  enabled: false
monitoring:
  enabled: true
  webhook:
    headers:
      - name: Content-Type
        value: application/json
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "webhook.url is required")
	})
}
