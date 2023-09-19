package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "git branch",
	Long:  ``,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		verboseFlag, _ := cmd.Flags().GetBool("verbose")
		options := command.BranchOption{
			Verbose: verboseFlag,
		}

		diff, _ := command.NewBranch(dir, args, options, stdout, stderr)
		code := diff.Run()
		os.Exit(code)
	},
}

func init() {
	branchCmd.Flags().BoolVar(&verbose, "verbose", false, "display additional details about each branch.")
	rootCmd.AddCommand(branchCmd)
}
