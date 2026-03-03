package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "hapctl",
		Short: "HAProxy Control CLI and Agent",
		Long: `hapctl is a CLI and agent tool for managing HAProxy configurations dynamically.
It monitors YAML configuration files and automatically applies changes to HAProxy,
with built-in health monitoring and webhook notifications.`,
		Version: Version,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/hapctl/config.yaml", "config file path")
}

func exitWithError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
