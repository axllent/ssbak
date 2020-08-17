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
	Short: "SSBak: manage SilverStripe .sspak archives.",
	Long:  `SSBak - sspak database/asset backup tool for SilverStripe.`,
	// 	Example: `  ssbak load . website.sspak # restores a backup
	//   ssbak save website.sspak   # saves a backup`,
	SilenceUsage:  true, // suppress help screen on error
	SilenceErrors: true, // suppress duplicate error on error
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// delete temporary files after completion
		app.Cleanup()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	altTmpDir := os.Getenv("TMPDIR")
	if altTmpDir != "" {
		app.Log(fmt.Sprintf("Alternate tmp directory detected '%s'", altTmpDir))

		// os.Exit(0)
		app.TempDir = altTmpDir
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)

		// Clean up temporary files on error
		app.Cleanup()

		// detect if subcommand is valid
		help := "\nSee: `ssbak -h` for help"
		if len(os.Args) > 0 {
			for _, t := range rootCmd.Commands() {
				if t.Name() == os.Args[1] {
					help = "\nSee: ssbak " + os.Args[1] + " -h` for help"
				}
			}
		}
		fmt.Println(help)

		os.Exit(1)
	}
}

func init() {
	// Clean up temporary files on cancel
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigs
		app.Cleanup()
		os.Exit(0)
	}()
}
