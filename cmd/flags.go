package cmd

import (
	"errors"

	"github.com/axllent/ssbak/app"
	"github.com/spf13/cobra"
)

// resolveOnlyFlags validates the --db & --assets flags, and maps an explicit
// --db=false or --assets=false to the equivalent "only" flag, eg: --db=false
// is the same as --assets (skip the database).
func resolveOnlyFlags(cmd *cobra.Command) error {
	dbFalse := cmd.Flags().Changed("db") && !app.OnlyDB
	assetsFalse := cmd.Flags().Changed("assets") && !app.OnlyAssets

	if dbFalse && assetsFalse {
		return errors.New("nothing to do: both --db=false and --assets=false are set")
	}

	if dbFalse {
		app.OnlyAssets = true
	}

	if assetsFalse {
		app.OnlyDB = true
	}

	if app.OnlyAssets && app.OnlyDB {
		return errors.New("you cannot use --assets and --db flags together")
	}

	return nil
}
