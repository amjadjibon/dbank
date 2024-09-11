package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

const (
	// Version of the CLI
	Version = "0.0.1"
)

func getCommitHash() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value[:10]
			}
		}
	}

	return ""
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of dbank",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dbank: %s\n", Version)
		fmt.Printf("git: %s\n", getCommitHash())
		fmt.Printf("golang: %s\n", runtime.Version())
	},
}
