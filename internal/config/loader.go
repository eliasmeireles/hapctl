package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/eliasmeireles/hapctl/internal/models"
)

const (
	DefaultResourcePath      = "/etc/hapctl/resources"
	DefaultSyncInterval      = 5 * time.Second
	DefaultMonitorInterval   = 5 * time.Second
	DefaultSyncEnabled       = true
	DefaultMonitoringEnabled = true
)

func LoadConfig(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg models.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyDefaults(&cfg)

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func applyDefaults(cfg *models.Config) {
	if cfg.Sync.ResourcePath == "" {
		cfg.Sync.ResourcePath = DefaultResourcePath
	}
	if cfg.Sync.Interval == 0 {
		cfg.Sync.Interval = DefaultSyncInterval
	}
	if cfg.Monitoring.Interval == 0 {
		cfg.Monitoring.Interval = DefaultMonitorInterval
	}
}

func validateConfig(cfg *models.Config) error {
	if cfg.Sync.Enabled && cfg.Sync.ResourcePath == "" {
		return fmt.Errorf("sync.resource-path is required when sync is enabled")
	}
	if cfg.Sync.Interval < time.Second {
		return fmt.Errorf("sync.interval must be at least 1 second")
	}
	if cfg.Monitoring.Enabled && cfg.Monitoring.Interval < time.Second {
		return fmt.Errorf("monitoring.interval must be at least 1 second")
	}
	if cfg.Monitoring.Webhook != nil && cfg.Monitoring.Webhook.URL == "" {
		return fmt.Errorf("monitoring.webhook.url is required when webhook is configured")
	}
	return nil
}
