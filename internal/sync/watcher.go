package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eliasmeireles/hapctl/internal/config"
	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	resourcePath string
	interval     time.Duration
	manager      *haproxy.Manager
	watcher      *fsnotify.Watcher
	resources    map[string]*models.BindResource
}

func NewWatcher(cfg *models.SyncConfig, manager *haproxy.Manager) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		resourcePath: cfg.ResourcePath,
		interval:     cfg.Interval,
		manager:      manager,
		watcher:      fsWatcher,
		resources:    make(map[string]*models.BindResource),
	}, nil
}

func (w *Watcher) Start(ctx context.Context) error {
	logger.Info("Starting sync watcher for path: %s", w.resourcePath)

	if err := w.addWatchRecursive(w.resourcePath); err != nil {
		return fmt.Errorf("failed to add watch path: %w", err)
	}

	if err := w.initialSync(); err != nil {
		logger.Error("Initial sync failed: %v", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping sync watcher")
			w.watcher.Close()
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

	if err := w.manager.ApplyBindResource(resource); err != nil {
		logger.Error("Failed to apply resource from %s: %v", path, err)
		return
	}

	w.resources[path] = resource
	logger.Info("Successfully synced resource from %s", path)
}

func (w *Watcher) initialSync() error {
	logger.Info("Performing initial sync")

	resources, err := config.LoadBindResourcesFromDir(w.resourcePath)
	if err != nil {
		return err
	}

	for path, resource := range resources {
		if err := w.manager.ApplyBindResource(resource); err != nil {
			logger.Error("Failed to apply resource from %s: %v", path, err)
			continue
		}
		w.resources[path] = resource
	}

	logger.Info("Initial sync completed, loaded %d resources", len(resources))
	return nil
}

func (w *Watcher) periodicSync() error {
	logger.Debug("Performing periodic sync check")

	resources, err := config.LoadBindResourcesFromDir(w.resourcePath)
	if err != nil {
		return err
	}

	for path := range w.resources {
		if _, exists := resources[path]; !exists {
			w.handleRemove(path)
		}
	}

	for path := range resources {
		if _, exists := w.resources[path]; !exists {
			w.syncFile(path)
		}
	}

	return nil
}

func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}
