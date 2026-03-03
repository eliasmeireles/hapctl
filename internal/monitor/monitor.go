package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/models"
)

type Monitor struct {
	interval time.Duration
	webhook  *models.WebhookConfig
	binds    []*models.Bind
}

func NewMonitor(cfg *models.MonitoringConfig) *Monitor {
	return &Monitor{
		interval: cfg.Interval,
		webhook:  cfg.Webhook,
		binds:    make([]*models.Bind, 0),
	}
}

func (m *Monitor) RegisterBind(bind *models.Bind) {
	m.binds = append(m.binds, bind)
	logger.Debug("Registered bind for monitoring: %s", bind.Name)
}

func (m *Monitor) UnregisterBind(bindName string) {
	for i, bind := range m.binds {
		if bind.Name == bindName {
			m.binds = append(m.binds[:i], m.binds[i+1:]...)
			logger.Debug("Unregistered bind from monitoring: %s", bindName)
			return
		}
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	logger.Info("Starting monitoring service with interval: %s", m.interval)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping monitoring service")
			return nil

		case <-ticker.C:
			m.checkBinds()
		}
	}
}

func (m *Monitor) checkBinds() {
	if len(m.binds) == 0 {
		logger.Debug("No binds to monitor")
		return
	}

	logger.Debug("Checking %d binds", len(m.binds))

	report := models.MonitoringReport{
		Timestamp: time.Now(),
		Binds:     make([]models.BindStatus, 0, len(m.binds)),
	}

	for _, bind := range m.binds {
		if !bind.Enabled {
			continue
		}

		status := m.checkBind(bind)
		report.Binds = append(report.Binds, status)

		if status.Status != "healthy" {
			logger.Error("Bind %s is unhealthy: %s", bind.Name, status.Error)
		}
	}

	if m.webhook != nil {
		if err := m.sendWebhook(&report); err != nil {
			logger.Error("Failed to send webhook: %v", err)
		}
	} else {
		if err := logger.LogMonitoring("", &report); err != nil {
			logger.Error("Failed to log monitoring report: %v", err)
		}
	}
}

func (m *Monitor) checkBind(bind *models.Bind) models.BindStatus {
	status := models.BindStatus{
		Name:      bind.Name,
		Type:      bind.Type,
		IP:        bind.IP,
		Port:      bind.Port,
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	address := fmt.Sprintf("%s:%d", m.resolveIP(bind.IP), bind.Port)

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		return status
	}
	defer conn.Close()

	return status
}

func (m *Monitor) resolveIP(ip string) string {
	if ip == "*" || ip == "" {
		return "127.0.0.1"
	}
	return ip
}

func (m *Monitor) sendWebhook(report *models.MonitoringReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	req, err := http.NewRequest("POST", m.webhook.URL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	for _, header := range m.webhook.Headers {
		req.Header.Set(header.Name, header.Value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	logger.Debug("Webhook sent successfully to %s", m.webhook.URL)
	return nil
}
