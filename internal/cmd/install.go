package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
)

var (
	checkOnly         bool
	skipConfig        bool
	skipService       bool
	installConfigPath string
	installCmd        = &cobra.Command{
		Use:   "install",
		Short: "Install HAProxy on the system",
		Long: `Install HAProxy using the system package manager.
Supports apt (Debian/Ubuntu), yum (CentOS/RHEL), and dnf (Fedora).
Requires root/sudo privileges.

This command will automatically:
1. Install HAProxy package
2. Generate base HAProxy configuration
3. Install hapctl systemd service`,
		Run: runInstall,
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check if HAProxy is installed")
	installCmd.Flags().BoolVar(&skipConfig, "skip-config", false, "Skip HAProxy configuration generation")
	installCmd.Flags().BoolVar(&skipService, "skip-service", false, "Skip systemd service installation")
	installCmd.Flags().StringVar(
		&installConfigPath,
		"config-path",
		"/etc/haproxy/haproxy.cfg",
		"Path to HAProxy config file",
	)
}

func ensureHAProxyUserExists() error {
	cmd := exec.Command("id", "-u", "haproxy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("haproxy user does not exist")
	}
	return nil
}

func createHAProxyUser() error {
	groupCmd := exec.Command("groupadd", "--system", "haproxy")
	if err := groupCmd.Run(); err != nil {
		checkGroupCmd := exec.Command("getent", "group", "haproxy")
		if checkGroupCmd.Run() != nil {
			return fmt.Errorf("failed to create haproxy group: %w", err)
		}
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
	}

	logger.Info("✅ HAProxy user and group created successfully")
	return nil
}

func checkAndFixPermissions(filePath string, expectedMode os.FileMode) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	currentMode := info.Mode().Perm()
	if currentMode != expectedMode {
		logger.Warn("⚠️  File %s has incorrect permissions: %o (expected: %o)", filePath, currentMode, expectedMode)
		logger.Info("Attempting to fix permissions...")

		if err := os.Chmod(filePath, expectedMode); err != nil {
			return fmt.Errorf("failed to change permissions: %w", err)
		}

		logger.Info("✅ Permissions corrected to %o for %s", expectedMode, filePath)
	}

	return nil
}

func ensureDirectoryPermissions(dirPath string, expectedMode os.FileMode) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dirPath)
	}

	currentMode := info.Mode().Perm()
	if currentMode != expectedMode {
		logger.Warn("⚠️  Directory %s has incorrect permissions: %o (expected: %o)", dirPath, currentMode, expectedMode)
		logger.Info("Attempting to fix permissions...")

		if err := os.Chmod(dirPath, expectedMode); err != nil {
			return fmt.Errorf("failed to change directory permissions: %w", err)
		}

		logger.Info("✅ Directory permissions corrected to %o for %s", expectedMode, dirPath)
	}

	return nil
}

func generateHAProxyConfigNonInteractive() error {
	if err := ensureHAProxyUserExists(); err != nil {
		logger.Warn("⚠️  HAProxy user/group may not exist: %v", err)
		logger.Info("Attempting to create haproxy user/group...")
		if err := createHAProxyUser(); err != nil {
			logger.Error("Failed to create haproxy user/group: %v", err)
			logger.Info("You may need to create it manually: sudo useradd --system haproxy")
		}
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil {
		logger.Info("HAProxy config already exists at: %s (skipping)", configPath)
		return nil
	}

	if err := os.WriteFile(configPath, []byte(haproxy.DefaultHAProxyConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("HAProxy base configuration generated at: %s", configPath)

	if err := checkAndFixPermissions(configPath, 0644); err != nil {
		logger.Error("Warning: Failed to verify/fix permissions for config file: %v", err)
	}

	configDir := filepath.Dir(configPath)
	if err := ensureDirectoryPermissions(configDir, 0755); err != nil {
		logger.Error("Warning: Failed to verify/fix permissions for config directory: %v", err)
	}

	if !haproxy.ErrorPagesExist() {
		logger.Info("Generating error pages...")
		if err := haproxy.GenerateErrorPages(); err != nil {
			logger.Error("Warning: Failed to generate error pages: %v", err)
		} else {
			logger.Info("Error pages generated at: %s", haproxy.ErrorPagesDir)

			if err := ensureDirectoryPermissions(haproxy.ErrorPagesDir, 0755); err != nil {
				logger.Error("Warning: Failed to verify/fix permissions for error pages directory: %v", err)
			}
		}
	}

	return nil
}

func installSystemdService() error {
	systemdPath := "/etc/systemd/system/hapctl-agent.service"
	configFilePath := "/etc/hapctl/config.yaml"

	absConfigPath, err := filepath.Abs(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	serviceContent := fmt.Sprintf(defaultServiceTemplate, absConfigPath)

	if err := os.WriteFile(systemdPath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	logger.Info("Service file created: %s", systemdPath)

	if err := checkAndFixPermissions(systemdPath, 0644); err != nil {
		logger.Error("Warning: Failed to verify/fix permissions for service file: %v", err)
	}

	systemdDir := filepath.Dir(systemdPath)
	if err := ensureDirectoryPermissions(systemdDir, 0755); err != nil {
		logger.Error("Warning: Failed to verify/fix permissions for systemd directory: %v", err)
	}

	if err := systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := systemctl("enable", "hapctl-agent"); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	return nil
}

func runInstall(cmd *cobra.Command, args []string) {
	installer := haproxy.NewInstaller()

	if checkOnly {
		if installer.IsInstalled() {
			version, err := installer.GetVersion()
			if err != nil {
				logger.Error("Failed to get HAProxy version: %v", err)
			} else {
				logger.Info("HAProxy is installed:\n%s", version)
			}
			return
		}
		fmt.Println("HAProxy is not installed")
		return
	}

	if installer.IsInstalled() {
		version, _ := installer.GetVersion()
		logger.Info("HAProxy is already installed:\n%s", version)
	} else {
		logger.Info("HAProxy is not installed. Starting installation...")

		if err := installer.Install(); err != nil {
			exitWithError("Failed to install HAProxy", err)
		}

		version, err := installer.GetVersion()
		if err != nil {
			logger.Error("Installation completed but failed to verify: %v", err)
			return
		}

		logger.Info("Installation completed successfully:\n%s", version)
	}

	if !skipConfig {
		logger.Info("\n=== Generating HAProxy Configuration ===")
		configPath = installConfigPath
		if err := generateHAProxyConfigNonInteractive(); err != nil {
			logger.Error("Failed to generate HAProxy config: %v", err)
			logger.Info("You can generate it manually with: hapctl haproxy-config --generate")
		} else {
			logger.Info("✅ HAProxy configuration generated successfully")
		}
	}

	if !skipService {
		logger.Info("\n=== Installing hapctl systemd service ===")
		if err := installSystemdService(); err != nil {
			logger.Error("Failed to install systemd service: %v", err)
			logger.Info("You can install it manually with: hapctl service install")
		} else {
			logger.Info("✅ Systemd service installed successfully")
		}
	}

	logger.Info("\n=== Installation Summary ===")
	logger.Info("✅ HAProxy installed and configured")
	if !skipConfig {
		logger.Info("✅ Configuration file: %s", installConfigPath)
	}
	if !skipService {
		logger.Info("✅ Systemd service: hapctl-agent")
		logger.Info("\nNext steps:")
		logger.Info("1. Review configuration: hapctl haproxy-config --view")
		logger.Info("2. Apply bind configurations: hapctl apply -f <bind-config.yaml>")
		logger.Info("3. Start hapctl agent: sudo systemctl start hapctl-agent")
		logger.Info("4. Check status: sudo systemctl status hapctl-agent")
	}
}
