package haproxy

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/eliasmeireles/hapctl/internal/logger"
)

type Installer struct{}

func NewInstaller() *Installer {
	return &Installer{}
}

func (i *Installer) IsInstalled() bool {
	cmd := exec.Command("haproxy", "-v")
	err := cmd.Run()
	return err == nil
}

func (i *Installer) GetVersion() (string, error) {
	cmd := exec.Command("haproxy", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get HAProxy version: %w", err)
	}
	return string(output), nil
}

func (i *Installer) Install() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("automatic installation is only supported on Linux")
	}

	logger.Info("Detecting package manager...")

	if i.commandExists("apt-get") {
		return i.installWithApt()
	} else if i.commandExists("yum") {
		return i.installWithYum()
	} else if i.commandExists("dnf") {
		return i.installWithDnf()
	}

	return fmt.Errorf("no supported package manager found (apt, yum, dnf)")
}

func (i *Installer) installWithApt() error {
	logger.Info("Installing HAProxy using apt...")

	logger.Info("Updating package list...")
	if err := i.runCommand("apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package list: %w", err)
	}

	logger.Info("Installing haproxy package...")
	if err := i.runCommand("apt-get", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	logger.Info("Enabling HAProxy service...")
	if err := i.runCommand("systemctl", "enable", "haproxy"); err != nil {
		logger.Error("Failed to enable haproxy service: %v", err)
	}

	logger.Info("Configuring HAProxy to include hapctl configs...")
	if err := i.ConfigureHAProxy(); err != nil {
		logger.Error("Failed to configure HAProxy: %v", err)
	}

	logger.Info("HAProxy installed successfully")
	return nil
}

func (i *Installer) installWithYum() error {
	logger.Info("Installing HAProxy using yum...")

	if err := i.runCommand("yum", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	logger.Info("Enabling HAProxy service...")
	if err := i.runCommand("systemctl", "enable", "haproxy"); err != nil {
		logger.Error("Failed to enable haproxy service: %v", err)
	}

	logger.Info("HAProxy installed successfully")
	return nil
}

func (i *Installer) installWithDnf() error {
	logger.Info("Installing HAProxy using dnf...")

	if err := i.runCommand("dnf", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	logger.Info("Enabling HAProxy service...")
	if err := i.runCommand("systemctl", "enable", "haproxy"); err != nil {
		logger.Error("Failed to enable haproxy service: %v", err)
	}

	logger.Info("HAProxy installed successfully")
	return nil
}

func (i *Installer) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (i *Installer) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func (i *Installer) ConfigureHAProxy() error {
	configPath := "/etc/haproxy/haproxy.cfg"

	// Check if already configured
	checkCmd := exec.Command("grep", "-q", "services.d", configPath)
	if checkCmd.Run() == nil {
		logger.Info("HAProxy already configured to include services.d")
		return nil
	}

	// Append configuration to include services.d configs
	includeConfig := "\n# hapctl managed configurations\n"
	includeConfig += "# Include HTTP services\n"
	includeConfig += "# Include TCP services\n"

	appendCmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' >> %s", includeConfig, configPath))
	if err := appendCmd.Run(); err != nil {
		return fmt.Errorf("failed to append config: %w", err)
	}

	logger.Info("HAProxy configuration updated")
	return nil
}
