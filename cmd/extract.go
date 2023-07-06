package cmd

import (
	"errors"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/utils"
	"github.com/spf13/cobra"
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract <sspak> [<output dir>]",
	Short: "Extract .sspak backup",
	Long:  `Extract the contents of an .sspak backup.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "."
		if len(args) == 2 {
			outputDir = args[1]
		}

		if app.OnlyAssets && app.OnlyDB {
			return errors.New("You cannot use --assets and --db flags together")
		}

		if err := utils.MkDirIfNotExists(outputDir); err != nil {
			return err
		}

		return utils.ExtractSSPak(args[0], outputDir)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().
		BoolVarP(&app.OnlyDB, "db", "", false, "only extract the database.sql.gz file")

	extractCmd.Flags().
		BoolVarP(&app.OnlyAssets, "assets", "", false, "only extract the assets.tar.gz file")

	extractCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
