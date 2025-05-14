package cmd

import (
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/amjadjibon/dbank/app"
	"github.com/amjadjibon/dbank/conf"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the HTTP server",
	Run: func(cmd *cobra.Command, _ []string) {
		cfg := conf.NewConfig()

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		server, err := app.NewServer(ctx, cfg)
		if err != nil {
			cmd.PrintErr(err)
		}

		if err := server.Start(ctx); err != nil {
			cmd.PrintErr(err)
		}
	},
}
