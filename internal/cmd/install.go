package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
)

var (
	checkOnly bool
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install HAProxy on the system",
		Long: `Install HAProxy using the system package manager.
Supports apt (Debian/Ubuntu), yum (CentOS/RHEL), and dnf (Fedora).
Requires root/sudo privileges.`,
		Run: runInstall,
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check if HAProxy is installed")
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
		return
	}

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
