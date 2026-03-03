package ssl

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/eliasmeireles/hapctl/internal/models"
)

func LoadSSLConfig(path string) (*models.SSLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config models.SSLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}

func LoadSSLConfigsFromDir(dir string) (map[string]*models.SSLConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*models.SSLConfig), nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	configs := make(map[string]*models.SSLConfig)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !isYAMLFile(entry.Name()) {
			continue
		}

		path := fmt.Sprintf("%s/%s", dir, entry.Name())
		config, err := LoadSSLConfig(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
		}

		configs[path] = config
	}

	return configs, nil
}

func isYAMLFile(filename string) bool {
	return len(filename) > 5 && (filename[len(filename)-5:] == ".yaml" || filename[len(filename)-4:] == ".yml")
}
