package haproxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	if err := m.ApplyBindWithoutReload(bind); err != nil {
		return err
	}
	return m.ReloadHAProxy()
}

func (m *Manager) ApplyBindWithoutReload(bind *models.Bind) error {
	if !bind.Enabled {
		logger.Info("Bind %s is disabled, removing config if exists", bind.Name)
		return m.RemoveBindWithoutReload(bind)
	}

	exists := m.generator.ConfigExists(bind)

	if exists && !bind.Override {
		logger.Info("Bind %s already exists and override is false, skipping", bind.Name)
		return nil
	}

	logger.Info("Applying bind configuration for %s", bind.Name)
	logger.Info("  Type: %s, IP: %s, Port: %d", bind.Type, bind.IP, bind.Port)
	logger.Info("  Backend servers: %d", len(bind.Backend.Servers))

	if err := m.generator.WriteBindConfig(bind); err != nil {
		return fmt.Errorf("failed to write bind config: %w", err)
	}

	config, _ := m.generator.GenerateBindConfig(bind)
	logger.Debug("Generated config:\n%s", config)

	if err := m.ValidateConfig(); err != nil {
		logger.Error("HAProxy config validation failed: %v", err)
		if err := m.generator.RemoveBindConfig(bind); err != nil {
			logger.Error("Failed to rollback invalid config: %v", err)
		}
		return fmt.Errorf("invalid HAProxy config: %w", err)
	}

	logger.Info("✓ Successfully applied bind configuration for %s", bind.Name)
	return nil
}

func (m *Manager) RemoveBind(bind *models.Bind) error {
	if err := m.RemoveBindWithoutReload(bind); err != nil {
		return err
	}
	return m.ReloadHAProxy()
}

func (m *Manager) RemoveBindWithoutReload(bind *models.Bind) error {
	if !m.generator.ConfigExists(bind) {
		logger.Debug("Bind %s config does not exist, nothing to remove", bind.Name)
		return nil
	}

	logger.Info("Removing bind configuration for %s", bind.Name)
	logger.Info("  Type: %s, IP: %s, Port: %d", bind.Type, bind.IP, bind.Port)

	if err := m.generator.RemoveBindConfig(bind); err != nil {
		return fmt.Errorf("failed to remove bind config: %w", err)
	}

	logger.Info("✓ Successfully removed bind configuration for %s", bind.Name)
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

func (m *Manager) regenerateMainConfig() error {
	baseConfigPath := "/etc/haproxy/haproxy.cfg"
	backupPath := "/etc/haproxy/haproxy.cfg.backup"

	// Read base config (up to hapctl managed section)
	baseConfig, err := os.ReadFile(baseConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("HAProxy config not found, creating default configuration")
			baseConfig = []byte(DefaultHAProxyConfig)
			if err := os.WriteFile(baseConfigPath, baseConfig, 0644); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}

			if !ErrorPagesExist() {
				logger.Info("Generating HAProxy error pages")
				if err := GenerateErrorPages(); err != nil {
					logger.Warn("Failed to generate error pages: %v", err)
				}
			}
		} else {
			return fmt.Errorf("failed to read base config: %w", err)
		}
	}

	// Find where hapctl section starts and keep only the base
	configStr := string(baseConfig)
	hapctlMarker := "# hapctl managed configurations"
	baseOnly := configStr
	if idx := strings.Index(configStr, hapctlMarker); idx != -1 {
		baseOnly = configStr[:idx]
	}

	// Trim excessive trailing blank lines from base config
	baseOnly = strings.TrimRight(baseOnly, "\n")

	// Build new config with services.d includes
	var newConfig strings.Builder
	newConfig.WriteString(baseOnly)
	newConfig.WriteString("\n# hapctl managed configurations\n")

	// Include HTTP services
	httpDir := filepath.Join(m.generator.servicesDir, HTTPServicesDir)
	if httpConfigs, err := m.readServiceConfigs(httpDir); err == nil && len(httpConfigs) > 0 {
		newConfig.WriteString("# HTTP services\n")
		newConfig.WriteString(httpConfigs)
	}

	// Include TCP services
	tcpDir := filepath.Join(m.generator.servicesDir, TCPServicesDir)
	if tcpConfigs, err := m.readServiceConfigs(tcpDir); err == nil && len(tcpConfigs) > 0 {
		newConfig.WriteString("\n# TCP services\n")
		newConfig.WriteString(tcpConfigs)
	}

	// Backup current config
	if err := os.WriteFile(backupPath, baseConfig, 0644); err != nil {
		logger.Warn("Failed to backup config: %v", err)
	}

	// Write new config
	if err := os.WriteFile(baseConfigPath, []byte(newConfig.String()), 0644); err != nil {
		return fmt.Errorf("failed to write new config: %w", err)
	}

	logger.Debug("Regenerated main HAProxy config")
	return nil
}

func (m *Manager) readServiceConfigs(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return "", err
	}

	var configs strings.Builder
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			logger.Error("Failed to read service config %s: %v", file, err)
			continue
		}
		configs.WriteString(string(content))
		configs.WriteString("\n")
	}

	return configs.String(), nil
}

func (m *Manager) ReloadHAProxy() error {
	// Regenerate main config with all services.d includes
	if err := m.regenerateMainConfig(); err != nil {
		logger.Error("Failed to regenerate main config: %v", err)
		return fmt.Errorf("failed to regenerate config: %w", err)
	}

	cmd := exec.Command("systemctl", "reload", "haproxy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reload failed: %s", string(output))
	}

	// Check HAProxy status
	statusCmd := exec.Command("systemctl", "is-active", "haproxy")
	statusOutput, _ := statusCmd.CombinedOutput()
	status := strings.TrimSpace(string(statusOutput))

	logger.Info("HAProxy reloaded successfully (status: %s)", status)
	return nil
}

func (m *Manager) ApplyBindResource(resource *models.BindResource) error {
	for i := range resource.Binds {
		bind := &resource.Binds[i]
		if err := m.ApplyBindWithoutReload(bind); err != nil {
			return fmt.Errorf("failed to apply bind %s: %w", bind.Name, err)
		}
	}
	return m.ReloadHAProxy()
}

func (m *Manager) ApplyConfig(resources map[string]*models.BindResource) error {
	for path, resource := range resources {
		for i := range resource.Binds {
			bind := &resource.Binds[i]
			if err := m.ApplyBindWithoutReload(bind); err != nil {
				logger.Error("Failed to apply bind %s from %s: %v", bind.Name, path, err)
				continue
			}
		}
	}
	return m.ReloadHAProxy()
}

func (m *Manager) GenerateBindConfig(bind *models.Bind) (string, error) {
	return m.generator.GenerateBindConfig(bind)
}

func (m *Manager) CleanupOrphanedConfigs(resources map[string]*models.BindResource) error {
	// Get all bind names from resources
	validBinds := make(map[string]bool)
	for _, resource := range resources {
		for _, bind := range resource.Binds {
			validBinds[bind.Name] = true
		}
	}

	// Check HTTP configs
	httpDir := filepath.Join(DefaultHAProxyConfigDir, DefaultServicesDir, HTTPServicesDir)
	if err := m.cleanupDir(httpDir, validBinds); err != nil {
		return err
	}

	// Check TCP configs
	tcpDir := filepath.Join(DefaultHAProxyConfigDir, DefaultServicesDir, TCPServicesDir)
	if err := m.cleanupDir(tcpDir, validBinds); err != nil {
		return err
	}

	return nil
}

func (m *Manager) cleanupDir(dir string, validBinds map[string]bool) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return err
	}

	needsReload := false
	for _, file := range files {
		filename := filepath.Base(file)

		// Only process hapctl- prefixed files
		if !strings.HasPrefix(filename, NamePrefix) {
			continue
		}

		// Extract bind name (remove hapctl- prefix and .cfg suffix)
		bindName := strings.TrimPrefix(filename, NamePrefix)
		bindName = strings.TrimSuffix(bindName, ".cfg")

		// If bind not in valid resources, remove it
		if !validBinds[bindName] {
			logger.Info("Removing orphaned config: %s", filename)
			if err := os.Remove(file); err != nil {
				logger.Error("Failed to remove orphaned config %s: %v", filename, err)
			} else {
				needsReload = true
			}
		}
	}

	// Reload HAProxy if we removed any configs
	if needsReload {
		logger.Info("Orphaned configs removed, reloading HAProxy")
		return m.ReloadHAProxy()
	}

	return nil
}
