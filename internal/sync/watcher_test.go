package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMonitor struct{}

func (m *mockMonitor) RegisterBind(bind *models.Bind) {} // Mock implementation
func (m *mockMonitor) UnregisterBind(bindName string) {} // Mock implementation
func (m *mockMonitor) ClearBinds()                    {} // Mock implementation

func TestWatcherConcurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hapctl-watcher-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a dummy resource file
	resourceFile := filepath.Join(tmpDir, "test.yaml")
	content := `
binds:
  - name: test
    enabled: true
    type: http
    port: 80
    backend:
      servers:
        - name: srv1
          address: 127.0.0.1:8080
`
	err = os.WriteFile(resourceFile, []byte(content), 0644)
	require.NoError(t, err)

	manager := haproxy.NewManager(tmpDir)
	watcher, err := NewWatcher(&models.SyncConfig{
		ResourcePath: tmpDir,
		Interval:     10 * time.Millisecond,
	}, manager, &mockMonitor{})
	require.NoError(t, err)

	// This test doesn't easily prove a race without -race flag,
	// but it ensures that calling the protected methods doesn't dead-lock
	// and they can be called sequentially or concurrently without crashing.

	t.Run("sequential sync calls should not deadlock", func(t *testing.T) {
		err := watcher.periodicSync()
		assert.NoError(t, err)

		err = watcher.forcedResync()
		assert.NoError(t, err)
	})

	t.Run("concurrent sync calls should be protected by mutex", func(t *testing.T) {
		// Start multiple goroutines trying to sync at the same time
		errChan := make(chan error, 10)
		for i := 0; i < 5; i++ {
			go func() {
				errChan <- watcher.periodicSync()
			}()
			go func() {
				errChan <- watcher.forcedResync()
			}()
		}

		for i := 0; i < 10; i++ {
			assert.NoError(t, <-errChan)
		}
	})
}
