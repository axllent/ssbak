package cmd

import (
	"errors"
	"path"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/sspak"
	"github.com/axllent/ssbak/utils"
	"github.com/spf13/cobra"
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:     "save <webroot> <sspak>",
	Short:   "Create .sspak backup of database and/or assets",
	Long:    `Create .sspak archive from a Silverstripe database and/or assets.`,
	Example: `  ssbak save ./ website.sspak`,
	Args:    cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := app.BootstrapEnv(args[0]); err != nil {
			return err
		}

		if app.OnlyAssets && app.OnlyDB {
			return errors.New("you cannot use --assets and --db flags together")
		}

		archive := sspak.New()

		if !app.OnlyAssets {
			if err := archive.AddDatabase(); err != nil {
				return err
			}
		}

		if !app.OnlyDB {
			var assetsDir string

			if utils.IsDir(path.Join(app.ProjectRoot, "assets")) {
				assetsDir = app.RealPath(path.Join(app.ProjectRoot, "assets"))
			} else if utils.IsDir(path.Join(app.ProjectRoot, "public", "assets")) {
				assetsDir = app.RealPath(path.Join(app.ProjectRoot, "public", "assets"))
			} else {
				return errors.New("could not locate assets directory")
			}

			if err := archive.AddAssets(assetsDir); err != nil {
				return err
			}
		}

		return archive.Write(args[1])
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)

	saveCmd.Flags().
		BoolVarP(&app.OnlyDB, "db", "", false, "only save the database")

	saveCmd.Flags().
		BoolVarP(&app.OnlyAssets, "assets", "", false, "only save the assets")

	saveCmd.Flags().
		BoolVarP(&app.IgnoreResampled, "ignore-resampled", "i", false, "ignore most resampled images")

	saveCmd.Flags().
		BoolVarP(&sspak.UseZSTD, "zstd", "z", false, "use zstd compression (experimental)")

	saveCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
