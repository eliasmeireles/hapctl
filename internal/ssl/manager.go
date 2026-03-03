package ssl

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
)

const (
	RenewalThresholdDays = 30
	CertbotPath          = "/usr/bin/certbot"
	LetsEncryptLivePath  = "/etc/letsencrypt/live"
)

type Manager struct {
	config       *models.SSLManagerConfig
	certificates map[string]*models.Certificate
}

func NewManager(cfg *models.SSLManagerConfig) *Manager {
	return &Manager{
		config:       cfg,
		certificates: make(map[string]*models.Certificate),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	logger.Info("Starting SSL certificate manager")
	logger.Info("Renewal check interval: %s", m.config.RenewalCheck)

	if err := m.ensureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	if err := m.checkCertbot(); err != nil {
		logger.Error("Certbot not found: %v", err)
		logger.Info("Install certbot: sudo apt-get install certbot python3-certbot-dns-cloudflare")
		return err
	}

	if err := m.loadSSLConfigs(); err != nil {
		logger.Error("Failed to load SSL configs: %v", err)
	}

	if err := m.initialCertificateCheck(); err != nil {
		logger.Error("Initial certificate check failed: %v", err)
	}

	ticker := time.NewTicker(m.config.RenewalCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping SSL certificate manager")
			return nil

		case <-ticker.C:
			m.checkAndRenewCertificates()
		}
	}
}

func (m *Manager) ensureDirectories() error {
	dirs := []string{
		m.config.ConfigPath,
		m.config.CertPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (m *Manager) checkCertbot() error {
	if _, err := os.Stat(CertbotPath); err != nil {
		return fmt.Errorf("certbot not installed at %s", CertbotPath)
	}
	return nil
}

func (m *Manager) loadSSLConfigs() error {
	logger.Info("Loading SSL configurations from %s", m.config.ConfigPath)

	files, err := filepath.Glob(filepath.Join(m.config.ConfigPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob SSL configs: %w", err)
	}

	if len(files) == 0 {
		logger.Info("No SSL configuration files found")
		return nil
	}

	for _, file := range files {
		if err := m.loadSSLConfig(file); err != nil {
			logger.Error("Failed to load SSL config from %s: %v", file, err)
			continue
		}
	}

	logger.Info("Loaded %d SSL domain configurations", len(m.certificates))
	return nil
}

func (m *Manager) loadSSLConfig(path string) error {
	config, err := LoadSSLConfig(path)
	if err != nil {
		return err
	}

	email := config.Config.Mail
	if email == "" {
		email = m.config.Email
	}

	if email == "" {
		logger.Warn("SSL config %s ignored: email is required but not set", path)
		logger.Warn("Configure 'mail' in SSL config or 'ssl.email' in main config")
		return nil
	}

	if len(config.Config.Domain) == 0 {
		logger.Warn("SSL config %s ignored: no domains specified", path)
		return nil
	}

	for _, domain := range config.Config.Domain {
		cert := &models.Certificate{
			Domain:     domain,
			IsWildcard: strings.HasPrefix(domain, "*."),
		}
		m.certificates[domain] = cert
		logger.Debug("Registered domain for SSL management: %s", domain)
	}

	return nil
}

func (m *Manager) initialCertificateCheck() error {
	logger.Info("Performing initial certificate check")

	for domain := range m.certificates {
		status := m.checkCertificate(domain)

		if !status.IsValid {
			logger.Info("Certificate for %s is not valid, requesting new certificate", domain)
			if err := m.requestCertificate(domain); err != nil {
				logger.Error("Failed to request certificate for %s: %v", domain, err)
				continue
			}
		} else if status.NeedsRenewal {
			logger.Info("Certificate for %s needs renewal (%d days remaining)", domain, status.DaysRemaining)
			if err := m.renewCertificate(domain); err != nil {
				logger.Error("Failed to renew certificate for %s: %v", domain, err)
			}
		} else {
			logger.Info("Certificate for %s is valid (%d days remaining)", domain, status.DaysRemaining)
		}
	}

	return nil
}

func (m *Manager) checkAndRenewCertificates() {
	logger.Debug("Checking certificates for renewal")

	for domain := range m.certificates {
		status := m.checkCertificate(domain)

		if !status.IsValid {
			logger.Info("Certificate for %s is invalid, requesting new certificate", domain)
			if err := m.requestCertificate(domain); err != nil {
				logger.Error("Failed to request certificate for %s: %v", domain, err)
			}
		} else if status.NeedsRenewal {
			logger.Info("Certificate for %s needs renewal (%d days remaining)", domain, status.DaysRemaining)
			if err := m.renewCertificate(domain); err != nil {
				logger.Error("Failed to renew certificate for %s: %v", domain, err)
			}
		}
	}
}

func (m *Manager) checkCertificate(domain string) models.CertificateStatus {
	status := models.CertificateStatus{
		Domain:  domain,
		IsValid: false,
	}

	certPath := m.getCertificatePath(domain, "fullchain.pem")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		status.Error = "certificate file not found"
		return status
	}

	certData, err := os.ReadFile(certPath)
	if err != nil {
		status.Error = fmt.Sprintf("failed to read certificate: %v", err)
		return status
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		status.Error = "failed to decode PEM block"
		return status
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		status.Error = fmt.Sprintf("failed to parse certificate: %v", err)
		return status
	}

	status.ExpiryDate = cert.NotAfter
	status.DaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
	status.IsValid = time.Now().Before(cert.NotAfter)
	status.NeedsRenewal = status.DaysRemaining <= RenewalThresholdDays

	return status
}

func (m *Manager) requestCertificate(domain string) error {
	logger.Info("Requesting new certificate for %s", domain)

	cert := m.certificates[domain]
	if cert == nil {
		return fmt.Errorf("domain %s not registered", domain)
	}

	var args []string
	args = append(args, "certonly")
	args = append(args, "--non-interactive")
	args = append(args, "--agree-tos")
	args = append(args, "--email", m.config.Email)

	if cert.IsWildcard {
		args = append(args, "--dns-cloudflare")
		args = append(args, "--dns-cloudflare-credentials", "/etc/letsencrypt/cloudflare.ini")
		args = append(args, "-d", domain)

		baseDomain := strings.TrimPrefix(domain, "*.")
		args = append(args, "-d", baseDomain)
	} else {
		args = append(args, "--standalone")
		args = append(args, "-d", domain)
	}

	cmd := exec.Command(CertbotPath, args...)
	output, err := cmd.CombinedOutput()

	logger.Debug("Certbot output: %s", string(output))

	if err != nil {
		return fmt.Errorf("certbot failed: %w, output: %s", err, string(output))
	}

	if err := m.createHAProxyPEM(domain); err != nil {
		return fmt.Errorf("failed to create HAProxy PEM: %w", err)
	}

	logger.Info("Successfully obtained certificate for %s", domain)
	return nil
}

func (m *Manager) renewCertificate(domain string) error {
	logger.Info("Renewing certificate for %s", domain)

	cmd := exec.Command(CertbotPath, "renew", "--cert-name", m.getCertName(domain), "--non-interactive")
	output, err := cmd.CombinedOutput()

	logger.Debug("Certbot renew output: %s", string(output))

	if err != nil {
		return fmt.Errorf("certbot renew failed: %w, output: %s", err, string(output))
	}

	if err := m.createHAProxyPEM(domain); err != nil {
		return fmt.Errorf("failed to create HAProxy PEM: %w", err)
	}

	logger.Info("Successfully renewed certificate for %s", domain)
	return nil
}

func (m *Manager) createHAProxyPEM(domain string) error {
	certName := m.getCertName(domain)
	liveDir := filepath.Join(LetsEncryptLivePath, certName)

	fullchainPath := filepath.Join(liveDir, "fullchain.pem")
	privkeyPath := filepath.Join(liveDir, "privkey.pem")

	fullchain, err := os.ReadFile(fullchainPath)
	if err != nil {
		return fmt.Errorf("failed to read fullchain: %w", err)
	}

	privkey, err := os.ReadFile(privkeyPath)
	if err != nil {
		return fmt.Errorf("failed to read privkey: %w", err)
	}

	pemPath := filepath.Join(m.config.CertPath, fmt.Sprintf("%s.pem", certName))

	pemData := append(fullchain, privkey...)
	if err := os.WriteFile(pemPath, pemData, 0600); err != nil {
		return fmt.Errorf("failed to write PEM file: %w", err)
	}

	logger.Info("Created HAProxy PEM file: %s", pemPath)
	return nil
}

func (m *Manager) getCertName(domain string) string {
	if strings.HasPrefix(domain, "*.") {
		return strings.TrimPrefix(domain, "*.")
	}
	return domain
}

func (m *Manager) getCertificatePath(domain, file string) string {
	certName := m.getCertName(domain)
	return filepath.Join(LetsEncryptLivePath, certName, file)
}

func (m *Manager) GetCertificatePath(domain string) string {
	certName := m.getCertName(domain)
	return filepath.Join(m.config.CertPath, fmt.Sprintf("%s.pem", certName))
}
