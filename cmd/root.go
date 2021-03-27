package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/axllent/ssbak/app"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ssbak",
	Short: "SSBak: manage Silverstripe .sspak archives.",
	Long: `SSBak - sspak database/asset backup & restore tool for Silverstripe.

Support/Documentation
  https://github.com/axllent/ssbak`,
	SilenceUsage:  true, // suppress help screen on error
	SilenceErrors: true, // suppress duplicate error on error
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// delete temporary files after completion
		return app.Cleanup()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	altTmpDir := os.Getenv("TMPDIR")
	if altTmpDir != "" {
		app.Log(fmt.Sprintf("Alternative tmp directory detected '%s'", altTmpDir))

		app.TempDir = altTmpDir
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)

		// Clean up temporary files on error, don't print any cleanup errors
		// as they would have already been returned above
		app.Cleanup() // #nosec

		// detect if subcommand is valid
		help := "\nSee: `ssbak -h` for help"
		if len(os.Args) > 1 {
			for _, t := range rootCmd.Commands() {
				if t.Name() == os.Args[1] {
					help = "\nSee: `ssbak " + os.Args[1] + " -h` for help"
				}
			}
		}

		fmt.Println(help)

		os.Exit(1)
	}
}

func init() {
	// hide the `help` command
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	// Clean up temporary files on cancel
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigs
		if err := app.Cleanup(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(0)
	}()
}
