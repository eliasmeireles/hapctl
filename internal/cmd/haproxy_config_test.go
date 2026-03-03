package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/hapctl/internal/haproxy"
)

func TestGenerateHAProxyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "haproxy.cfg")
	configPath = testConfigPath

	err := generateHAProxyConfig()
	require.NoError(t, err)

	content, err := os.ReadFile(testConfigPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "global")
	assert.Contains(t, string(content), "defaults")
	assert.Contains(t, string(content), "log /dev/log")
	assert.Equal(t, haproxy.DefaultHAProxyConfig, string(content))
}

func TestViewHAProxyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "haproxy.cfg")
	configPath = testConfigPath

	testContent := "test config content"
	err := os.WriteFile(testConfigPath, []byte(testContent), 0644)
	require.NoError(t, err)

	err = viewHAProxyConfig()
	require.NoError(t, err)
}

func TestViewHAProxyConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, "nonexistent.cfg")
	configPath = testConfigPath

	err := viewHAProxyConfig()
	require.NoError(t, err)
}
