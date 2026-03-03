package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliasmeireles/hapctl/internal/models"
	"gopkg.in/yaml.v3"
)

func LoadBindResource(path string) (*models.BindResource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read bind resource file: %w", err)
	}

	var resource models.BindResource
	if err := yaml.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse bind resource file: %w", err)
	}

	applyBindDefaults(&resource)

	if err := validateBindResource(&resource); err != nil {
		return nil, fmt.Errorf("invalid bind resource: %w", err)
	}

	return &resource, nil
}

func LoadBindResourcesFromDir(dirPath string) (map[string]*models.BindResource, error) {
	resources := make(map[string]*models.BindResource)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !isYAMLFile(path) {
			return nil
		}

		resource, err := LoadBindResource(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		resources[path] = resource
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func applyBindDefaults(resource *models.BindResource) {
	for i := range resource.Binds {
		bind := &resource.Binds[i]
		if bind.IP == "" {
			bind.IP = "*"
		}
		if !bind.Override {
			bind.Override = false
		}
	}
}

func validateBindResource(resource *models.BindResource) error {
	if len(resource.Binds) == 0 {
		return fmt.Errorf("at least one bind must be defined")
	}

	names := make(map[string]bool)
	for i, bind := range resource.Binds {
		if bind.Name == "" {
			return fmt.Errorf("bind[%d]: name is required", i)
		}
		if names[bind.Name] {
			return fmt.Errorf("bind[%d]: duplicate name '%s'", i, bind.Name)
		}
		names[bind.Name] = true

		if bind.Type != "tcp" && bind.Type != "http" {
			return fmt.Errorf("bind[%d] (%s): type must be 'tcp' or 'http'", i, bind.Name)
		}
		if bind.Port <= 0 || bind.Port > 65535 {
			return fmt.Errorf("bind[%d] (%s): port must be between 1 and 65535", i, bind.Name)
		}
	}

	return nil
}
