package cmd

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/config"
	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
)

var (
	applyFile string
	applyCmd  = &cobra.Command{
		Use:   "apply",
		Short: "Apply a bind configuration file",
		Long:  `Apply a bind configuration from a YAML file to HAProxy.`,
		Run:   runApply,
	}
)

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "bind configuration file (required)")
	applyCmd.MarkFlagRequired("file")
}

func runApply(cmd *cobra.Command, args []string) {
	resource, err := config.LoadBindResource(applyFile)
	if err != nil {
		exitWithError("Failed to load bind resource", err)
	}

	logger.Info("Applying bind resource from: %s", applyFile)

	manager := haproxy.NewManager("")

	if err := manager.ApplyBindResource(resource); err != nil {
		exitWithError("Failed to apply bind resource", err)
	}

	logger.Info("Successfully applied %d bind(s)", len(resource.Binds))
}
