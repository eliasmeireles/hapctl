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

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to create haproxy user/group: %v", err)
	}

	logger.Info("Updating package list...")
	if err := i.runCommand("apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package list: %w", err)
	}

	logger.Info("Installing haproxy package...")
	if err := i.runCommand("apt-get", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to verify haproxy user/group after installation: %v", err)
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

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to create haproxy user/group: %v", err)
	}

	if err := i.runCommand("yum", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to verify haproxy user/group after installation: %v", err)
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

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to create haproxy user/group: %v", err)
	}

	if err := i.runCommand("dnf", "install", "-y", "haproxy"); err != nil {
		return fmt.Errorf("failed to install haproxy: %w", err)
	}

	if err := i.ensureHAProxyUser(); err != nil {
		logger.Error("Failed to verify haproxy user/group after installation: %v", err)
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

func (i *Installer) ensureHAProxyUser() error {
	checkUserCmd := exec.Command("id", "-u", "haproxy")
	if err := checkUserCmd.Run(); err == nil {
		logger.Info("HAProxy user already exists")
		return nil
	}

	logger.Info("Creating haproxy system user and group...")

	groupCmd := exec.Command("groupadd", "--system", "haproxy")
	if err := groupCmd.Run(); err != nil {
		checkGroupCmd := exec.Command("getent", "group", "haproxy")
		if checkGroupCmd.Run() != nil {
			return fmt.Errorf("failed to create haproxy group: %w", err)
		}
		logger.Info("HAProxy group already exists")
	}

	userCmd := exec.Command("useradd",
		"--system",
		"--gid", "haproxy",
		"--home-dir", "/var/lib/haproxy",
		"--no-create-home",
		"--shell", "/usr/sbin/nologin",
		"--comment", "HAProxy system user",
		"haproxy")

	if err := userCmd.Run(); err != nil {
		checkUserCmd := exec.Command("id", "-u", "haproxy")
		if checkUserCmd.Run() != nil {
			return fmt.Errorf("failed to create haproxy user: %w", err)
		}
		logger.Info("HAProxy user already exists")
		return nil
	}

	logger.Info("✅ HAProxy user and group created successfully")
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
