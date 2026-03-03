package haproxy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadServiceConfigsFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hapctl-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)

	t.Run("should filter out shell variables but keep valid HAProxy config", func(t *testing.T) {
		// Create a mock .cfg file with mixed content (valid HAProxy + shell vars)
		content := `
# Valid comment
frontend hapctl-test
    bind *:80
    mode http
    default_backend hapctl-test-backend

# Malicious/Old shell variables
backend_port="6443"
backend_host="127.0.0.1"
path_prefix="/"

backend hapctl-test-backend
    mode http
    server srv1 127.0.0.1:8080 check
`
		httpDir := filepath.Join(m.generator.servicesDir, HTTPServicesDir)
		err := os.MkdirAll(httpDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(httpDir, "test.cfg"), []byte(content), 0644)
		require.NoError(t, err)

		result, err := m.readServiceConfigs(httpDir)
		require.NoError(t, err)

		// Assertions
		assert.Contains(t, result, "frontend hapctl-test")
		assert.Contains(t, result, "bind *:80")
		assert.Contains(t, result, "backend hapctl-test-backend")
		assert.Contains(t, result, "# Valid comment")

		// Should NOT contain shell variables
		assert.NotContains(t, result, "backend_port=\"6443\"")
		assert.NotContains(t, result, "backend_host=\"127.0.0.1\"")
		assert.NotContains(t, result, "path_prefix=\"/\"")

		// Ensure it doesn't accidentally filter out legitimate backend lines
		assert.Contains(t, result, "backend hapctl-test-backend")
	})

	t.Run("should handle multiple files and keep indentation", func(t *testing.T) {
		tcpDir := filepath.Join(m.generator.servicesDir, TCPServicesDir)
		err := os.MkdirAll(tcpDir, 0755)
		require.NoError(t, err)

		cfg1 := `listen tcp-1
    bind *:3306
    mode tcp`
		cfg2 := `listen tcp-2
    bind *:5672
    mode tcp`

		err = os.WriteFile(filepath.Join(tcpDir, "tcp1.cfg"), []byte(cfg1), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tcpDir, "tcp2.cfg"), []byte(cfg2), 0644)
		require.NoError(t, err)

		result, err := m.readServiceConfigs(tcpDir)
		require.NoError(t, err)

		assert.Contains(t, result, "listen tcp-1")
		assert.Contains(t, result, "listen tcp-2")
		assert.Contains(t, result, "    bind *:3306")
		assert.Contains(t, result, "    bind *:5672")
	})
}
