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

	// wa := webapp.NewWebApp(cfg.App.WebApp, true)
	ui := gui.NewFyneUI(version)
	// graceful.AddCallback(func() error {
	// 	zap.S().Info("shutting down webapp")
	// 	return ui.Shutdown(ctx)
	// })

	// go func() {
	if err := gui.Start(ctx, ui); err != nil {
		zap.S().Error("couldn't start xengate", zap.Error(err))
	}

	// graceful.ShutdownNow()
	// }()

	// if err := graceful.WaitShutdown(); err != nil {
	// 	zap.S().Error("unable to shutdown xengate gracefully", zap.Error(err))
	// 	return
	// }

	zap.S().Info("shutdown complete")
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
