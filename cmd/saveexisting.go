package cmd

import (
	"errors"
	"fmt"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/sspak"
	"github.com/axllent/ssbak/utils"
	"github.com/spf13/cobra"
)

// saveExistingCmd represents the saveexisting command
var saveExistingCmd = &cobra.Command{
	Use:     "saveexisting <sspak>",
	Short:   "Create .sspak backup from existing database SQL dump and/or assets",
	Long:    `Create .sspak backup from an existing database SQL dump and/or assets folder.`,
	Example: `  ssbak saveexisting website.sspak --db="database.sql" --assets="public/assets"`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sqlFile, _ := cmd.Flags().GetString("db")
		assetsDir, _ := cmd.Flags().GetString("assets")

		if sqlFile == "" && assetsDir == "" {
			return errors.New("you must specify either --db or --assets, or both")
		}

		if sqlFile != "" && !utils.IsFile(sqlFile) {
			return fmt.Errorf("database file '%s' does not exist", sqlFile)
		}

		if assetsDir != "" && !utils.IsDir(assetsDir) {
			return fmt.Errorf("assets directory '%s' does not exist", assetsDir)
		}

		archive := sspak.New()

		if sqlFile != "" {
			if err := archive.AddDatabaseFromFile(sqlFile); err != nil {
				return err
			}
		}

		if assetsDir != "" {
			if err := archive.AddAssets(assetsDir); err != nil {
				return err
			}
		}

		return archive.Write(args[0])
	},
}

func init() {
	rootCmd.AddCommand(saveExistingCmd)

	saveExistingCmd.Flags().
		StringP("db", "", "", "add an existing .sql file")

	saveExistingCmd.Flags().
		StringP("assets", "", "", "add an existing assets directory")

	saveExistingCmd.Flags().
		BoolVarP(&sspak.UseZSTD, "zstd", "z", false, "use zstd compression (experimental)")

	saveExistingCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
