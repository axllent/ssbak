package cmd

import (
	"fmt"
	"os"

	"github.com/axllent/ssbak/utils"
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

		update, _ := cmd.Flags().GetBool("update")

		// Allow prereleases
		utils.AllowPrereleases = true

		if update {
			return updateApp()
		}

		fmt.Printf("Version: %s\n", Version)
		latest, _, _, err := utils.GithubLatest(Repo, RepoBinaryName)
		if err == nil && utils.GreaterThan(latest, Version) {
			fmt.Printf("Update available: %s\nRun `%s version --update to update (required read/write access to installed directory).\n", latest, os.Args[0])
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().
		BoolP("update", "", false, "update to latest version")
}

func displayVersion() {
	latest, _, _, err := utils.GithubLatest(Repo, RepoBinaryName)
	if err == nil && utils.GreaterThan(latest, Version) {
		fmt.Printf("Update available: %s\nRun `%s -u` to update.\n", latest, os.Args[0])
	}
}

func updateApp() error {
	rel, err := utils.GithubUpdate(Repo, RepoBinaryName, Version)
	if err != nil {
		return err
	}
	fmt.Printf("Updated %s to version %s\n", os.Args[0], rel)
	return nil
}