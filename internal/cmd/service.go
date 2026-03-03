package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/logger"
)

const defaultServiceTemplate = `[Unit]
Description=HAProxy Control Agent
Documentation=https://github.com/eliasmeireles/hapctl
After=network.target haproxy.service
Wants=haproxy.service

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/hapctl agent --config %s
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=hapctl-agent

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/haproxy /var/log/hapctl /var/run

[Install]
WantedBy=multi-user.target
`

var (
	serviceConfigPath string
	serviceName       string
	serviceFilePath   string
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage hapctl systemd service",
	Long:  `Install, uninstall, or check status of hapctl systemd service`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install hapctl systemd service",
	Long:  `Generate and install hapctl systemd service file`,
	RunE:  runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall hapctl systemd service",
	Long:  `Stop, disable and remove hapctl systemd service`,
	RunE:  runServiceUninstall,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check hapctl systemd service status",
	Long:  `Show the status of hapctl systemd service`,
	RunE:  runServiceStatus,
}

func init() {
	serviceInstallCmd.Flags().StringVar(&serviceConfigPath, "config", "/etc/hapctl/config.yaml", "Path to hapctl config file")
	serviceInstallCmd.Flags().StringVar(&serviceName, "name", "hapctl-agent", "Service name")
	serviceInstallCmd.Flags().StringVar(&serviceFilePath, "service-file", "", "Custom service file path (optional)")

	serviceUninstallCmd.Flags().StringVar(&serviceName, "name", "hapctl-agent", "Service name")
	serviceStatusCmd.Flags().StringVar(&serviceName, "name", "hapctl-agent", "Service name")

	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}

func runServiceInstall(cmd *cobra.Command, args []string) error {
	logger.Info("Installing hapctl systemd service...")

	systemdPath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	var serviceContent string
	if serviceFilePath != "" {
		logger.Info("Using custom service file: %s", serviceFilePath)
		content, err := os.ReadFile(serviceFilePath)
		if err != nil {
			return fmt.Errorf("failed to read service file: %w", err)
		}
		serviceContent = string(content)
	} else {
		logger.Info("Using default service template")
		absConfigPath, err := filepath.Abs(serviceConfigPath)
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		serviceContent = fmt.Sprintf(defaultServiceTemplate, absConfigPath)
	}

	if err := os.WriteFile(systemdPath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	logger.Info("Service file created: %s", systemdPath)

	if err := systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := systemctl("enable", serviceName); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	logger.Info("Service installed and enabled successfully")
	logger.Info("Start the service with: sudo systemctl start %s", serviceName)
	logger.Info("Check status with: sudo systemctl status %s", serviceName)
	logger.Info("View logs with: sudo journalctl -u %s -f", serviceName)

	return nil
}

func runServiceUninstall(cmd *cobra.Command, args []string) error {
	logger.Info("Uninstalling hapctl systemd service...")

	if err := systemctl("stop", serviceName); err != nil {
		logger.Error("Failed to stop service (may not be running): %v", err)
	}

	if err := systemctl("disable", serviceName); err != nil {
		logger.Error("Failed to disable service: %v", err)
	}

	systemdPath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := os.Remove(systemdPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file: %w", err)
		}
		logger.Info("Service file not found: %s", systemdPath)
	} else {
		logger.Info("Service file removed: %s", systemdPath)
	}

	if err := systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	logger.Info("Service uninstalled successfully")
	return nil
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	output, err := exec.Command("systemctl", "status", serviceName).CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	return nil
}

func systemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
