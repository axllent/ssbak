package cmd

import (
	"fmt"
	"os"

	"github.com/axllent/ghru/v2"
	"github.com/axllent/ssbak/app"
	"github.com/spf13/cobra"
)

var (
	// Version is the default application version, updated on release
	Version = "dev"

	// Repo on Github for updater
	Repo = "axllent/ssbak"

	// RepoBinaryName on Github for updater
	RepoBinaryName = "ssbak"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the app version & update information",
	Long:  `Displays the ssbak version & update information.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		conf := ghru.Config{
			Repo:           "axllent/ssbak",
			ArchiveName:    "ssbak_{{.OS}}_{{.Arch}}",
			BinaryName:     "ssbak",
			CurrentVersion: Version,
		}

		update, _ := cmd.Flags().GetBool("update")

		if update {
			// Update the app
			rel, err := conf.SelfUpdate()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Printf("Updated %s to version %s\n", os.Args[0], rel.Tag)
			return nil
		}

		fmt.Printf("Version: %s\n", Version)

		release, err := conf.Latest()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// The latest version is the same version
		if release.Tag == Version {
			return nil
		}

		// A newer release is available
		fmt.Printf(
			"Update available: %s\nRun `%s version -u` to update (requires read/write access to install directory).\n",
			release.Tag,
			os.Args[0],
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().
		BoolP("update", "u", false, "update to latest version")

	versionCmd.Flags().
		BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output")
}
