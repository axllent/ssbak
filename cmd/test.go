package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/axllent/ssbak/app"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:    "test <webroot>",
	Short:  "Test DB environment, returns detected values",
	Long:   `Tests your database environment and returns the values.`,
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := app.BoostrapEnv(args[0]); err != nil {
			return err
		}
		app.GetTempDir()

		d, err := json.MarshalIndent(app.DB, "", "\t")
		if err != nil {
			return err
		}

		fmt.Println("Detected database environment variables:")
		fmt.Println(string(d))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
