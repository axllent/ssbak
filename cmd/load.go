package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/sspak"
	"github.com/axllent/ssbak/internal/utils"
	"github.com/spf13/cobra"
)

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load <sspak> [<webroot>]",
	Short: "Restore database and/or assets from .sspak backup",
	Long:  `Restore an .sspak file for a Silverstripe site. Deletes existing table data & assets so be careful!`,
	Example: `  ssbak load website.sspak
  ssbak load website.sspak --assets-dir /path/to/shared/assets`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.IsFile(args[0]) {
			return fmt.Errorf("'%s' does not exist", args[0])
		}

		if err := resolveOnlyFlags(cmd); err != nil {
			return err
		}

		app.ProjectRoot = "."
		if len(args) == 2 {
			app.ProjectRoot = args[1]
		}

		archive, err := sspak.Probe(args[0])
		if err != nil {
			return err
		}

		loadAssets := archive.AssetsFile != "" && !app.OnlyDB
		var assetsDir string

		// Determine & validate the assets target before restoring anything,
		// so an unsafe target fails before the database is replaced.
		if loadAssets {
			assetsDir, _ = cmd.Flags().GetString("assets-dir")
			if assetsDir != "" {
				// relative paths are relative to the webroot
				if !filepath.IsAbs(assetsDir) {
					assetsDir = filepath.Join(app.ProjectRoot, assetsDir)
				}
			} else {
				if utils.IsDir(path.Join(app.ProjectRoot, "public")) {
					assetsDir = path.Join(app.ProjectRoot, "public", "assets")
				} else {
					assetsDir = path.Join(app.ProjectRoot, "assets")
				}
			}

			cwd, _ := os.Getwd()
			if err := sspak.ValidateAssetsTarget(
				assetsDir,
				app.ProjectRoot,
				filepath.Join(app.ProjectRoot, "public"),
				args[0],
				cwd,
			); err != nil {
				return err
			}
		}

		if archive.DatabaseFile != "" && !app.OnlyAssets {
			if err := app.BootstrapEnv(app.ProjectRoot); err != nil {
				return err
			}

			dropDatabase, _ := cmd.Flags().GetBool("drop-db")
			if err := archive.LoadDatabase(dropDatabase); err != nil {
				return err
			}
		}

		if loadAssets {
			if err := archive.LoadAssets(assetsDir); err != nil {
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
		BoolVarP(&app.OnlyDB, "db", "", false, "only restore the database, not the assets")

	loadCmd.Flags().
		BoolVarP(&app.OnlyAssets, "assets", "", false, "only restore the assets, not the database")

	loadCmd.Flags().
		StringP("assets-dir", "", "", "restore assets to this directory, relative to <webroot> unless absolute (default: public/assets)")

	loadCmd.Flags().
		BoolVarP(&app.IgnoreResampled, "ignore-resampled", "i", false, "ignore most resampled images (experimental)")

	loadCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
