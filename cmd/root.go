package cmd

import (
	"fmt"
	"os"

	"hacklab/internal/lab"
	"hacklab/internal/store"
	"hacklab/tui"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "hacklab",
	Short: "Your terminal hacking playground",
	Long: `
H   H  AAAAA  CCCC   K   K  L      AAAAA  BBBB
H   H  A   A  C      K  K   L      A   A  B   B
HHHHH  AAAAA  C      KKK    L      AAAAA  BBBB
H   H  A   A  C      K  K   L      A   A  B   B
H   H  A   A  CCCC   K   K  LLLLL  A   A  BBBB

 Your terminal hacking playground.
 Spin up vulnerable labs, exploit them, level up.
`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a subcommand was invoked, cobra handles it — this only
		// runs when no subcommand matched (bare "hacklab").
		// Check if this is a first-time user with no labs installed.
		labsDir, err := store.LabsDir()
		if err != nil {
			return err
		}

		hasLabs := false
		if _, statErr := os.Stat(labsDir); statErr == nil {
			labs, discoverErr := lab.DiscoverLabs(labsDir)
			if discoverErr == nil && len(labs) > 0 {
				hasLabs = true
			}
		}

		if !hasLabs {
			// First run or no labs — launch the tutorial
			return tui.RunTutorial()
		}

		// User has labs, show help
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
