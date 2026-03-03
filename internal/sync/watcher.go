package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/eliasmeireles/hapctl/internal/config"
	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
)

type Watcher struct {
	resourcePath string
	interval     time.Duration
	manager      *haproxy.Manager
	monitor      Monitor
	watcher      *fsnotify.Watcher
	resources    map[string]*models.BindResource
	lastHash     string
}

type Monitor interface {
	RegisterBind(bind *models.Bind)
	UnregisterBind(bindName string)
	ClearBinds()
}

func NewWatcher(cfg *models.SyncConfig, manager *haproxy.Manager, mon Monitor) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		resourcePath: cfg.ResourcePath,
		interval:     cfg.Interval,
		manager:      manager,
		monitor:      mon,
		watcher:      fsWatcher,
		resources:    make(map[string]*models.BindResource),
	}, nil
}

func (w *Watcher) Start(ctx context.Context) error {
	logger.Info("Starting sync watcher for path: %s", w.resourcePath)

	if err := w.addWatchRecursive(w.resourcePath); err != nil {
		return fmt.Errorf("failed to add watch path: %w", err)
	}

	// Execute initial sync on startup
	logger.Info("Performing initial sync")
	if err := w.initialSync(); err != nil {
		logger.Error("Initial sync failed: %v", err)
	}

	// Periodic sync ticker (configurable interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Forced resync ticker (5 minutes fallback)
	forcedResyncTicker := time.NewTicker(5 * time.Minute)
	defer forcedResyncTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping sync watcher")
			_ = w.watcher.Close()
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			logger.Error("Watcher error: %v", err)

		case <-ticker.C:
			if err := w.periodicSync(); err != nil {
				logger.Error("Periodic sync failed: %v", err)
			}

		case <-forcedResyncTicker.C:
			logger.Info("Forced resync (5-minute fallback)")
			if err := w.forcedResync(); err != nil {
				logger.Error("Forced resync failed: %v", err)
			}
		}
	}
}

func (w *Watcher) addWatchRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info == nil {
			return nil
		}

		if info.IsDir() {
			if err := w.watcher.Add(walkPath); err != nil {
				return err
			}
			logger.Debug("Added watch for directory: %s", walkPath)
		}
		return nil
	})
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	logger.Debug("File event: %s %s", event.Op.String(), event.Name)

	if event.Op&fsnotify.Create == fsnotify.Create {
		w.handleCreate(event.Name)
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		w.handleWrite(event.Name)
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		w.handleRemove(event.Name)
	}
}

func (w *Watcher) handleCreate(path string) {
	if !isYAMLFile(path) {
		return
	}

	logger.Info("New resource file detected: %s", path)
	w.syncFile(path)
}

func (w *Watcher) handleWrite(path string) {
	if !isYAMLFile(path) {
		return
	}

	logger.Info("Resource file modified: %s", path)
	w.syncFile(path)
}

func (w *Watcher) handleRemove(path string) {
	if !isYAMLFile(path) {
		return
	}

	logger.Info("Resource file removed: %s", path)

	if resource, exists := w.resources[path]; exists {
		for i := range resource.Binds {
			bind := &resource.Binds[i]
			if err := w.manager.RemoveBind(bind); err != nil {
				logger.Error("Failed to remove bind %s: %v", bind.Name, err)
			}

			if w.monitor != nil {
				w.monitor.UnregisterBind(bind.Name)
			}
		}
		delete(w.resources, path)
	}
}

func (w *Watcher) syncFile(path string) {
	resource, err := config.LoadBindResource(path)
	if err != nil {
		logger.Error("Failed to load resource from %s: %v", path, err)
		return
	}

	for i := range resource.Binds {
		bind := &resource.Binds[i]
		if err := w.manager.ApplyBind(bind); err != nil {
			logger.Error("Failed to apply bind %s from %s: %v", bind.Name, path, err)
			continue
		}

		if w.monitor != nil && bind.Enabled {
			w.monitor.RegisterBind(bind)
		}
	}

	w.resources[path] = resource

	// Update hash after successful sync
	newHash, err := w.calculateResourcesHash()
	if err != nil {
		logger.Error("Failed to calculate hash: %v", err)
	} else {
		w.lastHash = newHash
	}

	logger.Info("Successfully synced resource from %s", path)
}

func (w *Watcher) initialSync() error {
	logger.Info("Performing initial sync")

	resources, err := config.LoadBindResourcesFromDir(w.resourcePath)
	if err != nil {
		return err
	}

	// Clear monitor binds before initial sync
	if w.monitor != nil {
		w.monitor.ClearBinds()
	}

	// Apply all configs with single reload
	if err := w.manager.ApplyConfig(resources); err != nil {
		logger.Error("Failed to apply configs: %v", err)
	}

	// Register binds for monitoring
	for path, resource := range resources {
		for i := range resource.Binds {
			bind := &resource.Binds[i]
			if w.monitor != nil && bind.Enabled {
				w.monitor.RegisterBind(bind)
			}
		}
		w.resources[path] = resource
	}

	// Calculate initial hash
	newHash, err := w.calculateResourcesHash()
	if err != nil {
		logger.Error("Failed to calculate initial hash: %v", err)
	} else {
		w.lastHash = newHash
	}

	logger.Info("Initial sync completed, loaded %d resources", len(resources))
	return nil
}

func (w *Watcher) periodicSync() error {
	logger.Debug("Performing periodic sync check")

	// Calculate current hash
	currentHash, err := w.calculateResourcesHash()
	if err != nil {
		logger.Error("Failed to calculate hash: %v", err)
	}

	// If hash hasn't changed, skip sync entirely
	if currentHash != "" && currentHash == w.lastHash {
		logger.Debug("No changes detected (hash match), skipping periodic sync")
		return nil
	}

	logger.Debug("Hash changed, performing sync")

	resources, err := config.LoadBindResourcesFromDir(w.resourcePath)
	if err != nil {
		return err
	}

	// Remove binds for files that no longer exist
	for path := range w.resources {
		if _, exists := resources[path]; !exists {
			w.handleRemove(path)
		}
	}

	// Apply all configs with single reload
	if err := w.manager.ApplyConfig(resources); err != nil {
		logger.Error("Failed to apply configs: %v", err)
	}

	// Register binds for monitoring
	for path, resource := range resources {
		for i := range resource.Binds {
			bind := &resource.Binds[i]
			if w.monitor != nil && bind.Enabled {
				w.monitor.RegisterBind(bind)
			}
		}
		w.resources[path] = resource
		logger.Info("Successfully synced resource from %s", path)
	}

	// Cleanup orphaned hapctl- configs
	if err := w.manager.CleanupOrphanedConfigs(resources); err != nil {
		logger.Error("Failed to cleanup orphaned configs: %v", err)
	}

	w.lastHash = currentHash
	return nil
}

func (w *Watcher) forcedResync() error {
	logger.Debug("Checking if forced resync is needed")

	// Calculate current hash
	currentHash, err := w.calculateResourcesHash()
	if err != nil {
		logger.Error("Failed to calculate hash: %v", err)
	}

	// If hash hasn't changed, skip forced resync entirely
	if currentHash != "" && currentHash == w.lastHash {
		logger.Debug("No changes detected (hash match), skipping forced resync")
		return nil
	}

	logger.Info("Hash changed, performing forced full resync")

	resources, err := config.LoadBindResourcesFromDir(w.resourcePath)
	if err != nil {
		return err
	}

	// Clear monitor binds to avoid duplicates
	if w.monitor != nil {
		w.monitor.ClearBinds()
	}

	// Remove binds for files that no longer exist
	for path := range w.resources {
		if _, exists := resources[path]; !exists {
			logger.Info("Removing stale resource: %s", path)
			w.handleRemove(path)
		}
	}

	// Force sync all files
	for path := range resources {
		logger.Debug("Force syncing: %s", path)
		w.syncFile(path)
	}

	// Cleanup orphaned hapctl- configs
	logger.Debug("Checking for orphaned configs")
	if err := w.manager.CleanupOrphanedConfigs(resources); err != nil {
		logger.Error("Failed to cleanup orphaned configs: %v", err)
	}

	w.lastHash = currentHash
	logger.Info("Forced resync completed, processed %d resources", len(resources))
	return nil
}

func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}

func (w *Watcher) calculateResourcesHash() (string, error) {
	var files []string

	err := filepath.Walk(w.resourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isYAMLFile(path) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	// Sort files for consistent hash
	sort.Strings(files)

	hasher := sha256.New()
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			logger.Error("Failed to open file for hashing: %s: %v", file, err)
			continue
		}

		if _, err := io.Copy(hasher, f); err != nil {
			_ = f.Close()
			logger.Error("Failed to hash file: %s: %v", file, err)
			continue
		}
		_ = f.Close()
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
