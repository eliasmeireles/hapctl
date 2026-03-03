package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/config"
	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
	"github.com/eliasmeireles/hapctl/internal/monitor"
	"github.com/eliasmeireles/hapctl/internal/sync"
)

var (
	debugMode bool
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Start the hapctl agent (sync + monitor)",
	Long: `Start the hapctl agent which will:
- Monitor the resource directory for YAML configuration files
- Automatically apply changes to HAProxy
- Perform health checks on configured binds
- Send webhook notifications for unhealthy binds`,
	Run: runAgent,
}

func init() {
	agentCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(agentCmd)
}

func ensureLogDirectory() error {
	logDir := "/var/log/hapctl"

	// Try to create with proper permissions
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If failed, try home directory fallback
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to create log directory and get home dir: %w", err)
		}

		fallbackDir := fmt.Sprintf("%s/.hapctl/log", homeDir)
		if err := os.MkdirAll(fallbackDir, 0755); err != nil {
			return fmt.Errorf("failed to create fallback log directory: %w", err)
		}
		logger.Info("[WARNING] Using fallback log directory: %s", fallbackDir)
		return nil
	}

	// Ensure permissions are correct
	os.Chmod(logDir, 0755)
	return nil
}

func ensureResourceDirectory(resourcePath string) error {
	if resourcePath == "" {
		return fmt.Errorf("resource path is empty")
	}

	// Create resource directory with proper permissions
	if err := os.MkdirAll(resourcePath, 0755); err != nil {
		return fmt.Errorf("failed to create resource directory %s: %w", resourcePath, err)
	}

	// Ensure permissions are correct
	os.Chmod(resourcePath, 0755)
	logger.Info("Resource directory ready: %s", resourcePath)
	return nil
}

func runAgent(cmd *cobra.Command, args []string) {
	// Set debug mode
	logger.SetDebug(debugMode)

	// Ensure log directory exists with proper permissions
	if err := ensureLogDirectory(); err != nil {
		logger.Info("[WARNING] Failed to create log directory: %v", err)
	}

	installer := haproxy.NewInstaller()
	if !installer.IsInstalled() {
		exitWithError("HAProxy is not installed", fmt.Errorf("please install HAProxy first using: sudo hapctl install"))
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		exitWithError("Failed to load config", err)
	}

	logger.Info("Starting hapctl agent")
	if debugMode {
		logger.Info("Debug mode enabled")
	}
	logger.Info("Config file: %s", cfgFile)
	logger.Info("Resource path: %s", cfg.Sync.ResourcePath)

	// Ensure resource directory exists
	if err := ensureResourceDirectory(cfg.Sync.ResourcePath); err != nil {
		logger.Info("[WARNING] Failed to create resource directory: %v", err)
	}

	manager := haproxy.NewManager("")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 2)

	if cfg.Sync.Enabled {
		watcher, err := sync.NewWatcher(&cfg.Sync, manager)
		if err != nil {
			exitWithError("Failed to create watcher", err)
		}

		go func() {
			if err := watcher.Start(ctx); err != nil {
				errChan <- err
			}
		}()
	}

	if cfg.Monitoring.Enabled {
		mon := monitor.NewMonitor(&cfg.Monitoring)

		go func() {
			if err := mon.Start(ctx); err != nil {
				errChan <- err
			}
		}()
	}

	logger.Info("Agent started successfully")

	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
		cancel()
	case err := <-errChan:
		logger.Error("Agent error: %v", err)
		cancel()
	}

	logger.Info("Agent stopped")
}
