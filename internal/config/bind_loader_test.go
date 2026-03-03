package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadBindResource(t *testing.T) {
	t.Run("must load valid TCP bind resource", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - name: test-server
    override: true
    enabled: true
    description: Test server
    type: tcp
    port: 8080
    backend:
      servers:
        - name: server1
          address: 127.0.0.1:8080
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		resource, err := LoadBindResource(bindPath)
		require.NoError(t, err)
		require.NotNil(t, resource)
		require.Len(t, resource.Binds, 1)
		
		bind := resource.Binds[0]
		require.Equal(t, "test-server", bind.Name)
		require.True(t, bind.Override)
		require.True(t, bind.Enabled)
		require.Equal(t, "tcp", bind.Type)
		require.Equal(t, 8080, bind.Port)
		require.Equal(t, "*", bind.IP)
	})

	t.Run("must load valid HTTP bind resource", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - name: web-server
    enabled: true
    type: http
    ip: "10.0.0.1"
    port: 80
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		resource, err := LoadBindResource(bindPath)
		require.NoError(t, err)
		require.Len(t, resource.Binds, 1)
		
		bind := resource.Binds[0]
		require.Equal(t, "web-server", bind.Name)
		require.Equal(t, "http", bind.Type)
		require.Equal(t, "10.0.0.1", bind.IP)
		require.Equal(t, 80, bind.Port)
	})

	t.Run("must fail when bind name is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - type: tcp
    port: 8080
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		_, err = LoadBindResource(bindPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "name is required")
	})

	t.Run("must fail with invalid bind type", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - name: test
    type: invalid
    port: 8080
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		_, err = LoadBindResource(bindPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "type must be 'tcp' or 'http'")
	})

	t.Run("must fail with invalid port", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - name: test
    type: tcp
    port: 99999
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		_, err = LoadBindResource(bindPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "port must be between 1 and 65535")
	})

	t.Run("must fail with duplicate bind names", func(t *testing.T) {
		tmpDir := t.TempDir()
		bindPath := filepath.Join(tmpDir, "bind.yaml")

		bindContent := `
binds:
  - name: test
    type: tcp
    port: 8080
  - name: test
    type: tcp
    port: 8081
`
		err := os.WriteFile(bindPath, []byte(bindContent), 0644)
		require.NoError(t, err)

		_, err = LoadBindResource(bindPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate name")
	})
}

func TestLoadBindResourcesFromDir(t *testing.T) {
	t.Run("must load multiple resources from directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		bind1Content := `
binds:
  - name: server1
    type: tcp
    port: 8080
`
		bind2Content := `
binds:
  - name: server2
    type: http
    port: 80
`
		err := os.WriteFile(filepath.Join(tmpDir, "bind1.yaml"), []byte(bind1Content), 0644)
		require.NoError(t, err)
		
		err = os.WriteFile(filepath.Join(tmpDir, "bind2.yml"), []byte(bind2Content), 0644)
		require.NoError(t, err)

		resources, err := LoadBindResourcesFromDir(tmpDir)
		require.NoError(t, err)
		require.Len(t, resources, 2)
	})

	t.Run("must skip non-YAML files", func(t *testing.T) {
		tmpDir := t.TempDir()

		bindContent := `
binds:
  - name: server1
    type: tcp
    port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "bind.yaml"), []byte(bindContent), 0644)
		require.NoError(t, err)
		
		err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not yaml"), 0644)
		require.NoError(t, err)

		resources, err := LoadBindResourcesFromDir(tmpDir)
		require.NoError(t, err)
		require.Len(t, resources, 1)
	})
}
