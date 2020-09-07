package cmd

import (
	"errors"
	"path"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/utils"
	"github.com/spf13/cobra"
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:     "save <webroot> <sspak>",
	Short:   "Create .sspak backup of database and/or assets",
	Long:    `Create .sspak archive from a SilverStripe database and/or assets.`,
	Example: `  ssbak save ./ website.sspak`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := app.BoostrapEnv(args[0]); err != nil {
			return err
		}

		if app.OnlyAssets && app.OnlyDB {
			return errors.New("You cannot use --assets and --db flags together")
		}

		tmpDir := app.GetTempDir()

		sspakFiles := []string{}

		if !app.OnlyAssets {
			gzipFile := path.Join(tmpDir, "database.sql.gz")
			app.AddTempFile(gzipFile)

			// use map to determine which database function to use
			if err := utils.DBDumpWrapper[app.DB.Type](gzipFile); err != nil {
				return err
			}

			sspakFiles = append(sspakFiles, gzipFile)
		}

		if !app.OnlyDB {
			var assetsDir string

			if utils.IsDir(path.Join(app.ProjectRoot, "assets")) {
				assetsDir = path.Join(app.ProjectRoot, "assets")
			} else if utils.IsDir(path.Join(app.ProjectRoot, "public", "assets")) {
				assetsDir = path.Join(app.ProjectRoot, "public", "assets")
			} else {
				return errors.New("Could not locate assets directory")
			}
			assetsFile := path.Join(tmpDir, "assets.tar.gz")
			app.AddTempFile(assetsFile)

			if err := utils.AssetsToTarGz(assetsDir, assetsFile); err != nil {
				return err
			}

			sspakFiles = append(sspakFiles, assetsFile)
		}

		return utils.CreateSSPak(args[1], sspakFiles)
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)

	saveCmd.Flags().
		BoolVarP(&app.OnlyDB, "db", "", false, "only save the database")

	saveCmd.Flags().
		BoolVarP(&app.OnlyAssets, "assets", "", false, "only save the assets")

	saveCmd.Flags().
		BoolVarP(&app.IgnoreResampled, "ignore-resampled", "i", false, "ignore most resampled images (experimental)")

	saveCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
