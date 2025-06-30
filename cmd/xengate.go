package cmd

import (
	"context"
	_ "embed"

	"xengate/internal/gui"
	logg "xengate/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (

	//go:embed version.txt
	version string

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "serve the xengate service",
		Run:   serve,
	}
)

func serve(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	cfg := resolveConfig()

	logger := logg.New(cfg.Logger).Desugar()
	zap.ReplaceGlobals(logger)

	ui := gui.NewApp("XenGate", version)
	if err := ui.Start(ctx); err != nil {
		zap.S().Error("couldn't start xengate", zap.Error(err))
	}

	zap.S().Info("shutdown complete")
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
