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
	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) {
	installer := haproxy.NewInstaller()
	if !installer.IsInstalled() {
		exitWithError("HAProxy is not installed", fmt.Errorf("please install HAProxy first using: sudo hapctl install"))
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		exitWithError("Failed to load config", err)
	}

	logger.Info("Starting hapctl agent")
	logger.Info("Config file: %s", cfgFile)
	logger.Info("Resource path: %s", cfg.Sync.ResourcePath)

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
