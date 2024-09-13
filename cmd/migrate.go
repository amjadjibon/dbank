package cmd

import (
	"fmt"
	"os"

	"github.com/amjadjibon/dbank/db"
	"github.com/spf13/cobra"
)

var dbURL string

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate the database",
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Migrate the database up",
	Run: func(cmd *cobra.Command, _ []string) {
		if err := db.MigrateUp(cmd.Context(), dbURL); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Database migrated up")
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Migrate the database down",
	Run: func(cmd *cobra.Command, _ []string) {
		if err := db.MigrateDown(cmd.Context(), dbURL); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Database migrated down")
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)

	migrateUpCmd.Flags().StringVarP(&dbURL, "db-url", "d", "", "Database URL")
	migrateDownCmd.Flags().StringVarP(&dbURL, "db-url", "d", "", "Database URL")

	migrateUpCmd.PreRun = checkAndSetDBURL
	migrateDownCmd.PreRun = checkAndSetDBURL
}

func checkAndSetDBURL(*cobra.Command, []string) {
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
		if dbURL == "" {
			fmt.Println("set DB_URL environment variable or pass it as a flag --db-url=<db-url>")
			os.Exit(1)
		}
	}
}
