package cmd

import (
	"fmt"
	"os"

	"xengate/internal/config"

	"github.com/spf13/cobra"
)

var (
	configPath = "config.yml"
	skipConfig = false
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xengate",
	Short: "Payment Lab.",
	Long:  "Payment Lab is set of tools for integration test of components in the payment transaction domain.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// resolveConfig or exit with error
func resolveConfig() *config.Config {
	cfg, err := config.New(configPath, skipConfig)
	if err != nil {
		fmt.Printf("unable to initialize config: %s\n", err.Error())
		os.Exit(1)
	}

	if skipConfig {
		fmt.Println("Skipped file-based configuration, using only ENV")
	}

	return cfg
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yml", "path to yml config")
	rootCmd.PersistentFlags().BoolVar(&skipConfig, "skip-config", false, "skips config and uses ENV only")
}
