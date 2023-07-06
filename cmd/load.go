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

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:     "load <sspak> [<webroot>]",
	Short:   "Restore database and/or assets from .sspak backup",
	Long:    `Restore an .sspak file for a Silverstripe site. Deletes existing table data & assets so be careful!`,
	Example: `  ssbak load website.sspak`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.IsFile(args[0]) {
			return fmt.Errorf("'%s' does not exist", args[0])
		}

		if app.OnlyAssets && app.OnlyDB {
			return errors.New("You cannot use --assets and --db flags together")
		}

		app.ProjectRoot = "."
		if len(args) == 2 {
			app.ProjectRoot = args[1]
		}

		var assetsBase string
		if utils.IsDir(path.Join(app.ProjectRoot, "public")) {
			assetsBase = path.Join(app.ProjectRoot, "public")
		} else {
			assetsBase = app.ProjectRoot
		}

		tmpDir := app.GetTempDir()

		if err := utils.ExtractSSPak(args[0], tmpDir); err != nil {
			return err
		}

		gzipSQLFile := filepath.Join(tmpDir, "database.sql.gz")
		app.AddTempFile(gzipSQLFile)
		assetsFile := filepath.Join(tmpDir, "assets.tar.gz")
		app.AddTempFile(assetsFile)

		if utils.IsFile(gzipSQLFile) && !app.OnlyAssets {
			if err := app.BootstrapEnv(app.ProjectRoot); err != nil {
				return err
			}

			dropDatabase, _ := cmd.Flags().GetBool("drop-db")
			// use map to determine which database function to use
			if err := utils.DBCreateWrapper[app.DB.Type](dropDatabase); err != nil {
				return err
			}

			// use map to determine which database function to use
			if err := utils.DBLoadWrapper[app.DB.Type](gzipSQLFile); err != nil {
				return err
			}
		}

		if utils.IsFile(assetsFile) && !app.OnlyDB {
			if err := utils.AssetsFromTarGz(tmpDir, assetsBase); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)

	loadCmd.Flags().
		BoolP("drop-db", "", false, "drop existing database (if exists)")

	loadCmd.Flags().
		BoolVarP(&app.OnlyDB, "db", "", false, "only restore the database")

	loadCmd.Flags().
		BoolVarP(&app.OnlyAssets, "assets", "", false, "only restore the assets")

	loadCmd.Flags().
		BoolVarP(&app.IgnoreResampled, "ignore-resampled", "i", false, "ignore most resampled images (experimental)")

	loadCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
