package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose     bool
	delete      bool
	force       bool
	forceDelete bool
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
		deleteFlag, _ := cmd.Flags().GetBool("delete")
		forceFlag, _ := cmd.Flags().GetBool("force")
		forceDeleteFlag, _ := cmd.Flags().GetBool("forceDelete")
		if forceDeleteFlag {
			deleteFlag = true
			forceFlag = true
		}
		options := command.BranchOption{
			Verbose: verboseFlag,
			Delete:  deleteFlag,
			Force:   forceFlag,
		}

		diff, _ := command.NewBranch(dir, args, options, stdout, stderr)
		code := diff.Run()
		os.Exit(code)
	},
}

func init() {
	branchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "display additional details about each branch.")
	branchCmd.Flags().BoolVarP(&delete, "delete", "d", false, "Delete the specified branch.")
	branchCmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion of the branch, even if it has unmerged changes.")
	branchCmd.Flags().BoolVar(&forceDelete, "D", false, "Force deletion of the branch, even if it has unmerged changes.")
	rootCmd.AddCommand(branchCmd)
}
