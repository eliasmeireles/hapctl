package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/eliasmeireles/hapctl/internal/models"
)

const (
	DefaultResourcePath    = "/etc/hapctl/resources"
	DefaultSyncInterval    = 5 * time.Second
	DefaultMonitorInterval = 30 * time.Second
	DefaultSSLConfigPath   = "/etc/hapctl/ssl"
	DefaultSSLCertPath     = "/etc/haproxy/certs"
	DefaultSSLRenewalCheck = 12 * time.Hour
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
	if cfg.SSL.ConfigPath == "" {
		cfg.SSL.ConfigPath = DefaultSSLConfigPath
	}
	if cfg.SSL.CertPath == "" {
		cfg.SSL.CertPath = DefaultSSLCertPath
	}
	if cfg.SSL.RenewalCheck == 0 {
		cfg.SSL.RenewalCheck = DefaultSSLRenewalCheck
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
	if cfg.SSL.Enabled {
		if cfg.SSL.ConfigPath == "" {
			return fmt.Errorf("ssl.config-path is required when SSL is enabled")
		}
		if cfg.SSL.CertPath == "" {
			return fmt.Errorf("ssl.cert-path is required when SSL is enabled")
		}
		if cfg.SSL.Email == "" {
			return fmt.Errorf("ssl.email is required when SSL is enabled")
		}
		if cfg.SSL.RenewalCheck < time.Hour {
			return fmt.Errorf("ssl.renewal-check must be at least 1 hour")
		}
	}
	return nil
}
