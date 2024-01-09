package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var mergeMode = string(command.Run)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "git merge",
	Long:  ``,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		stdin := cmd.InOrStdin()
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		mode, _ := cmd.Flags().GetString("mode")
		options := command.MergeOption{
			Mode: command.MergeMode(mode),
		}
		merge, _ := command.NewMerge(dir, args, options, stdin, stdout, stderr)
		code := merge.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringVar(&mergeMode, "continue", string(command.Continue), "Resume command execution from a saved state")
	mergeCmd.Flags().StringVar(&mergeMode, "abort", string(command.Abort), "Cancel the current operation and revert to the pre-operation state")
}
