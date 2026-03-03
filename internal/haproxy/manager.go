package haproxy

import (
	"fmt"
	"os/exec"

	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
)

type Manager struct {
	generator *Generator
}

func NewManager(configDir string) *Manager {
	return &Manager{
		generator: NewGenerator(configDir),
	}
}

func (m *Manager) ApplyBind(bind *models.Bind) error {
	if !bind.Enabled {
		logger.Info("Bind %s is disabled, removing config if exists", bind.Name)
		return m.RemoveBind(bind)
	}

	exists := m.generator.ConfigExists(bind)

	if exists && !bind.Override {
		logger.Info("Bind %s already exists and override is false, skipping", bind.Name)
		return nil
	}

	logger.Info("Applying bind configuration for %s", bind.Name)

	if err := m.generator.WriteBindConfig(bind); err != nil {
		return fmt.Errorf("failed to write bind config: %w", err)
	}

	if err := m.ValidateConfig(); err != nil {
		logger.Error("HAProxy config validation failed: %v", err)
		if err := m.generator.RemoveBindConfig(bind); err != nil {
			logger.Error("Failed to rollback invalid config: %v", err)
		}
		return fmt.Errorf("invalid HAProxy config: %w", err)
	}

	if err := m.ReloadHAProxy(); err != nil {
		return fmt.Errorf("failed to reload HAProxy: %w", err)
	}

	logger.Info("Successfully applied bind configuration for %s", bind.Name)
	return nil
}

func (m *Manager) RemoveBind(bind *models.Bind) error {
	if !m.generator.ConfigExists(bind) {
		logger.Debug("Bind %s config does not exist, nothing to remove", bind.Name)
		return nil
	}

	logger.Info("Removing bind configuration for %s", bind.Name)

	if err := m.generator.RemoveBindConfig(bind); err != nil {
		return fmt.Errorf("failed to remove bind config: %w", err)
	}

	if err := m.ReloadHAProxy(); err != nil {
		return fmt.Errorf("failed to reload HAProxy: %w", err)
	}

	logger.Info("Successfully removed bind configuration for %s", bind.Name)
	return nil
}

func (m *Manager) ValidateConfig() error {
	cmd := exec.Command("haproxy", "-c", "-f", "/etc/haproxy/haproxy.cfg")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validation failed: %s", string(output))
	}
	return nil
}

func (m *Manager) ReloadHAProxy() error {
	cmd := exec.Command("systemctl", "reload", "haproxy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reload failed: %s", string(output))
	}
	logger.Info("HAProxy reloaded successfully")
	return nil
}

func (m *Manager) ApplyBindResource(resource *models.BindResource) error {
	for i := range resource.Binds {
		bind := &resource.Binds[i]
		if err := m.ApplyBind(bind); err != nil {
			return fmt.Errorf("failed to apply bind %s: %w", bind.Name, err)
		}
	}
	return nil
}

func (m *Manager) GenerateBindConfig(bind *models.Bind) (string, error) {
	return m.generator.GenerateBindConfig(bind)
}
