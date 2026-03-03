package haproxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
)

const (
	DefaultHAProxyConfigDir = "/etc/haproxy"
	DefaultServicesDir      = "services.d"
	HTTPServicesDir         = "http"
	TCPServicesDir          = "tcp"
	NamePrefix              = "hapctl-"
)

type Generator struct {
	configDir     string
	servicesDir   string
	usingFallback bool
}

func NewGenerator(configDir string) *Generator {
	if configDir == "" {
		configDir = DefaultHAProxyConfigDir
	}

	g := &Generator{
		configDir:     configDir,
		servicesDir:   filepath.Join(configDir, DefaultServicesDir),
		usingFallback: false,
	}

	if err := g.ensureDirectories(); err != nil {
		logger.Info("[WARNING] Cannot write to %s: %v", configDir, err)
		logger.Info("[WARNING] Falling back to $HOME/.hapctl due to permission denied")
		g.useFallbackDir()
	}

	return g
}

func (g *Generator) useFallbackDir() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory: %v", err)
		return
	}

	fallbackDir := filepath.Join(homeDir, ".hapctl", "haproxy")
	g.configDir = fallbackDir
	g.servicesDir = filepath.Join(fallbackDir, DefaultServicesDir)
	g.usingFallback = true

	if err := g.ensureDirectories(); err != nil {
		logger.Error("Failed to create fallback directory: %v", err)
	}
}

func (g *Generator) ensureDirectories() error {
	httpDir := filepath.Join(g.servicesDir, HTTPServicesDir)
	tcpDir := filepath.Join(g.servicesDir, TCPServicesDir)

	if err := os.MkdirAll(httpDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(tcpDir, 0755); err != nil {
		return err
	}

	return nil
}

func (g *Generator) GenerateBindConfig(bind *models.Bind) (string, error) {
	var builder strings.Builder

	switch bind.Type {
	case "http":
		builder.WriteString(g.generateHTTPConfig(bind))
	case "tcp":
		builder.WriteString(g.generateTCPConfig(bind))
	default:
		return "", fmt.Errorf("unsupported bind type: %s", bind.Type)
	}

	return builder.String(), nil
}

func (g *Generator) generateHTTPConfig(bind *models.Bind) string {
	var builder strings.Builder

	frontendName := NamePrefix + bind.Name

	fmt.Fprintf(&builder, "# %s\n", bind.Description)
	fmt.Fprintf(&builder, "frontend %s\n", frontendName)

	bindAddr := g.formatBindAddress(bind)
	fmt.Fprintf(&builder, "    bind %s\n", bindAddr)
	builder.WriteString("    mode http\n")

	// Handle redirect configuration
	if bind.Redirect != nil {
		g.generateRedirectRules(&builder, bind.Redirect)
		return builder.String()
	}

	// Normal backend configuration
	backendName := NamePrefix + bind.Name + "-backend"
	fmt.Fprintf(&builder, "    default_backend %s\n\n", backendName)

	fmt.Fprintf(&builder, "backend %s\n", backendName)
	builder.WriteString("    mode http\n")
	builder.WriteString("    balance roundrobin\n")

	if len(bind.Backend.Servers) > 0 {
		for _, server := range bind.Backend.Servers {
			serverName := NamePrefix + server.Name
			fmt.Fprintf(&builder, "    server %s %s check\n", serverName, server.Address)
		}
	}

	return builder.String()
}

func (g *Generator) generateRedirectRules(builder *strings.Builder, redirect *models.Redirect) {
	code := redirect.Code
	if code == 0 {
		code = 301
	}

	scheme := redirect.Scheme
	if scheme == "" {
		scheme = "https"
	}

	if redirect.Port > 0 && redirect.Port != 443 {
		fmt.Fprintf(builder, "    redirect prefix %s://%%[req.hdr(Host)]:%d%%[capture.req.uri] code %d\n",
			scheme, redirect.Port, code)
	} else {
		fmt.Fprintf(builder, "    redirect scheme %s code %d\n", scheme, code)
	}
}

func (g *Generator) generateTCPConfig(bind *models.Bind) string {
	var builder strings.Builder

	listenName := NamePrefix + bind.Name

	fmt.Fprintf(&builder, "# %s\n", bind.Description)
	fmt.Fprintf(&builder, "listen %s\n", listenName)

	bindAddr := g.formatBindAddress(bind)
	fmt.Fprintf(&builder, "    bind %s\n", bindAddr)
	builder.WriteString("    mode tcp\n")

	if len(bind.Backend.Servers) > 0 {
		builder.WriteString("    balance roundrobin\n")
		for _, server := range bind.Backend.Servers {
			serverName := NamePrefix + server.Name
			fmt.Fprintf(&builder, "    server %s %s check\n", serverName, server.Address)
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

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", NamePrefix+bind.Name))

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if g.usingFallback {
		logger.Info("[WARNING] Config written to fallback location: %s", configPath)
		logger.Info("[WARNING] You need to manually copy configs to %s or run with sudo", DefaultHAProxyConfigDir)
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

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", NamePrefix+bind.Name))

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

	configPath := filepath.Join(serviceDir, fmt.Sprintf("%s.cfg", NamePrefix+bind.Name))
	_, err := os.Stat(configPath)
	return err == nil
}
