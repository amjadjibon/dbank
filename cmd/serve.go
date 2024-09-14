package cmd

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/amjadjibon/dbank/handler"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		mux := http.NewServeMux()
		mux.HandleFunc("/swagger", handler.SwaggerUI)
		mux.HandleFunc("/swagger/v1/openapiv2.json", handler.SwaggerAPIv1)
		if err := http.ListenAndServe(":8080", mux); err != nil {
			cmd.PrintErr(err)
		}
	},
}
