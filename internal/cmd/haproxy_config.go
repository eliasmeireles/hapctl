package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/hapctl/internal/haproxy"
)

var (
	generateConfig bool
	viewConfig     bool
	configPath     string
)

var haproxyConfigCmd = &cobra.Command{
	Use:   "haproxy-config",
	Short: "Manage HAProxy base configuration",
	Long: `Generate or view the HAProxy base configuration file.
This command helps manage the global and defaults sections of HAProxy config.`,
	RunE: runHAProxyConfig,
}

func init() {
	rootCmd.AddCommand(haproxyConfigCmd)
	haproxyConfigCmd.Flags().BoolVar(&generateConfig, "generate", false, "Generate default HAProxy base configuration")
	haproxyConfigCmd.Flags().BoolVar(&viewConfig, "view", false, "View current HAProxy configuration")
	haproxyConfigCmd.Flags().StringVar(&configPath, "path", "/etc/haproxy/haproxy.cfg", "Path to HAProxy config file")
}

func runHAProxyConfig(cmd *cobra.Command, args []string) error {
	if generateConfig && viewConfig {
		return fmt.Errorf("cannot use --generate and --view together")
	}

	if !generateConfig && !viewConfig {
		return fmt.Errorf("must specify either --generate or --view")
	}

	if generateConfig {
		return generateHAProxyConfig()
	}

	if viewConfig {
		return viewHAProxyConfig()
	}

	return nil
}

func generateHAProxyConfig() error {
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("⚠️  HAProxy config already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := os.WriteFile(configPath, []byte(haproxy.DefaultHAProxyConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ HAProxy base configuration generated at: %s\n", configPath)

	if !haproxy.ErrorPagesExist() {
		fmt.Println("\nGenerating error pages...")
		if err := haproxy.GenerateErrorPages(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to generate error pages: %v\n", err)
		} else {
			fmt.Printf("✅ Error pages generated at: %s\n", haproxy.ErrorPagesDir)
		}
	} else {
		fmt.Printf("✓ Error pages already exist at: %s\n", haproxy.ErrorPagesDir)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("1. Review and customize the configuration if needed")
	fmt.Println("2. Test the configuration: haproxy -c -f", configPath)
	fmt.Println("3. Apply bind configurations: hapctl apply -f <bind-config.yaml>")
	fmt.Println("4. Start hapctl agent: hapctl agent --config /etc/hapctl/config.yaml")

	return nil
}

func viewHAProxyConfig() error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("❌ HAProxy config not found at: %s\n", configPath)
			fmt.Println("\nGenerate default config with: hapctl haproxy-config --generate")
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	fmt.Printf("HAProxy configuration at: %s\n", configPath)
	fmt.Println("========================================")
	fmt.Print(string(content))
	fmt.Println("========================================")

	return nil
}
