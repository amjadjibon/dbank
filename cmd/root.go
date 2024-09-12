package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dbank",
	Short: "Digital Bank CLI tool to manage migrations, seeds, server and more",
	Long: `Digital Bank CLI tool to manage migrations, seeds, server and more
Complete documentation is available at https://github.com/amjadjibon/dbank`,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(migrateCmd)
}
