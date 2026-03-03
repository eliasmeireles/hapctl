package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/config"
	"github.com/eliasmeireles/hapctl/internal/haproxy"
	"github.com/eliasmeireles/hapctl/internal/logger"
)

var (
	validateFile string
	validateCmd  = &cobra.Command{
		Use:   "validate",
		Short: "Validate a bind configuration file",
		Long:  `Validate a bind configuration YAML file without applying it.`,
		Run:   runValidate,
	}
)

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&validateFile, "file", "f", "", "bind configuration file (required)")
	_ = validateCmd.MarkFlagRequired("file")
}

func runValidate(cmd *cobra.Command, args []string) {
	resource, err := config.LoadBindResource(validateFile)
	if err != nil {
		exitWithError("Validation failed", err)
	}

	logger.Info("Validating bind resource from: %s", validateFile)

	manager := haproxy.NewManager("")

	for i, bind := range resource.Binds {
		configStr, err := manager.GenerateBindConfig(&bind)
		if err != nil {
			exitWithError(fmt.Sprintf("Failed to generate config for bind[%d] (%s)", i, bind.Name), err)
		}

		logger.Info("Bind[%d] %s: valid", i, bind.Name)
		logger.Debug("Generated config:\n%s", configStr)
	}

	logger.Info("Validation successful: %d bind(s) are valid", len(resource.Binds))
}
