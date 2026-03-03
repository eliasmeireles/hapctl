package models

import "time"

type Config struct {
	Sync       SyncConfig       `yaml:"sync"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

type SyncConfig struct {
	ResourcePath string        `yaml:"resource-path"`
	Interval     time.Duration `yaml:"interval"`
	Enabled      bool          `yaml:"enabled"`
}

type MonitoringConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Webhook  *WebhookConfig `yaml:"webhook,omitempty"`
}

type WebhookConfig struct {
	URL     string   `yaml:"url"`
	Headers []Header `yaml:"headers,omitempty"`
}

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
