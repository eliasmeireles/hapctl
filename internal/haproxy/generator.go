package haproxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliasmeireles/hapctl/internal/models"
)

const (
	DefaultHAProxyConfigDir = "/etc/haproxy"
	DefaultServicesDir      = "services.d"
	HTTPServicesDir         = "http"
	TCPServicesDir          = "tcp"
)

type Generator struct {
	configDir   string
	servicesDir string
}

func NewGenerator(configDir string) *Generator {
	if configDir == "" {
		configDir = DefaultHAProxyConfigDir
	}

	return &Generator{
		configDir:   configDir,
		servicesDir: filepath.Join(configDir, DefaultServicesDir),
	}
}

func (g *Generator) GenerateBindConfig(bind *models.Bind) (string, error) {
	var builder strings.Builder

	if bind.Type == "http" {
		builder.WriteString(g.generateHTTPConfig(bind))
	} else if bind.Type == "tcp" {
		builder.WriteString(g.generateTCPConfig(bind))
	} else {
		return "", fmt.Errorf("unsupported bind type: %s", bind.Type)
	}

	return builder.String(), nil
}

func (g *Generator) generateHTTPConfig(bind *models.Bind) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s\n", bind.Description))
	builder.WriteString(fmt.Sprintf("frontend %s\n", bind.Name))
	
	bindAddr := g.formatBindAddress(bind)
	builder.WriteString(fmt.Sprintf("    bind %s\n", bindAddr))
	builder.WriteString("    mode http\n")
	builder.WriteString(fmt.Sprintf("    default_backend %s_backend\n\n", bind.Name))

	builder.WriteString(fmt.Sprintf("backend %s_backend\n", bind.Name))
	builder.WriteString("    mode http\n")
	builder.WriteString("    balance roundrobin\n")

	if len(bind.Backend.Servers) > 0 {
		for _, server := range bind.Backend.Servers {
			builder.WriteString(fmt.Sprintf("    server %s %s check\n", server.Name, server.Address))
		}
	}

	return builder.String()
}

func (g *Generator) generateTCPConfig(bind *models.Bind) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s\n", bind.Description))
	builder.WriteString(fmt.Sprintf("listen %s\n", bind.Name))
	
	bindAddr := g.formatBindAddress(bind)
	builder.WriteString(fmt.Sprintf("    bind %s\n", bindAddr))
	builder.WriteString("    mode tcp\n")

	if len(bind.Backend.Servers) > 0 {
		builder.WriteString("    balance roundrobin\n")
		for _, server := range bind.Backend.Servers {
			builder.WriteString(fmt.Sprintf("    server %s %s check\n", server.Name, server.Address))
		}
	}

	return builder.String()
}

func (g *Generator) formatBindAddress(bind *models.Bind) string {
	ip := bind.IP
	if ip == "" || ip == "*" {
		ip = "*"
	}
	return fmt.Sprintf("%s:%d", ip, bind.Port)
}

func (g *Generator) WriteBindConfig(bind *models.Bind) error {
	config, err := g.GenerateBindConfig(bind)
	if err != nil {
		return err
	}

	var serviceDir string
	if bind.Type == "http" {
		serviceDir = filepath.Join(g.servicesDir, HTTPServicesDir)
	} else {
		serviceDir = filepath.Join(g.servicesDir, TCPServicesDir)
	}

	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", bind.Name))
	
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (g *Generator) RemoveBindConfig(bind *models.Bind) error {
	var serviceDir string
	if bind.Type == "http" {
		serviceDir = filepath.Join(g.servicesDir, HTTPServicesDir)
	} else {
		serviceDir = filepath.Join(g.servicesDir, TCPServicesDir)
	}

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", bind.Name))
	
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

func (g *Generator) ConfigExists(bind *models.Bind) bool {
	var serviceDir string
	if bind.Type == "http" {
		serviceDir = filepath.Join(g.servicesDir, HTTPServicesDir)
	} else {
		serviceDir = filepath.Join(g.servicesDir, TCPServicesDir)
	}

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", bind.Name))
	_, err := os.Stat(configPath)
	return err == nil
}
