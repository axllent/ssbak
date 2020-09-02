package cmd

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/utils"
	"github.com/spf13/cobra"
)

// saveexistingCmd represents the saveexisting command
var saveexistingCmd = &cobra.Command{
	Use:     "saveexisting <sspak>",
	Short:   "Create .sspak backup from existing database SQL dump and/or assets",
	Long:    `Create .sspak backup from an existing database SQL dump and/or assets folder.`,
	Example: `  ssbak saveexisting website.sspak --db="database.sql" --assets="public/assets"`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sqlFile, _ := cmd.Flags().GetString("db")
		assetsDir, _ := cmd.Flags().GetString("assets")

		if sqlFile == "" && assetsDir == "" {
			return errors.New("You must specify either --db or --assets, or both")
		}

		if sqlFile != "" && !utils.IsFile(sqlFile) {
			return fmt.Errorf("Database file '%s' does not exist", sqlFile)
		}

		if assetsDir != "" && !utils.IsDir(assetsDir) {
			return fmt.Errorf("Assets directory '%s' does not exist", assetsDir)
		}

		tmpDir := app.GetTempDir()

		sspakFiles := []string{}

		if sqlFile != "" {
			gzipSQL := filepath.Join(tmpDir, "database.sql.gz")
			app.AddTempFile(gzipSQL)

			if err := utils.GzipFile(sqlFile, gzipSQL); err != nil {
				return err
			}
			sspakFiles = append(sspakFiles, gzipSQL)
		}

		if assetsDir != "" {
			assetsFile := path.Join(tmpDir, "assets.tar.gz")
			app.AddTempFile(assetsFile)

			if err := utils.AssetsToTarGz(assetsDir, assetsFile); err != nil {
				return err
			}

			sspakFiles = append(sspakFiles, assetsFile)
		}

		return utils.CreateSSPak(args[0], sspakFiles)
	},
}

func init() {
	rootCmd.AddCommand(saveexistingCmd)

	saveexistingCmd.Flags().
		StringP("db", "", "", "add an existing .sql file")

	saveexistingCmd.Flags().
		StringP("assets", "", "", "add an existing assets directory")

	saveexistingCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
