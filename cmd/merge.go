package cmd

import (
	"building-git/lib/command"
	"building-git/lib/command/write_commit"
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
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		mode, _ := cmd.Flags().GetString("mode")
		message, _ := cmd.Flags().GetString("message")
		file, _ := cmd.Flags().GetString("file")
		edit, _ := cmd.Flags().GetBool("edit")
		options := command.MergeOption{
			Mode: command.MergeMode(mode),
			ReadOption: write_commit.ReadOption{
				Message: message,
				File:    file,
			},
			Edit: edit,
		}
		merge, _ := command.NewMerge(dir, args, options, stdout, stderr)
		code := merge.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringVar(&mergeMode, "continue", string(command.Continue), "Resume command execution from a saved state")
	mergeCmd.Flags().StringVar(&mergeMode, "abort", string(command.Abort), "Cancel the current operation and revert to the pre-operation state")

	mergeCmd.Flags().StringP("message", "m", "", "Specify a message to associate with the command execution")
	mergeCmd.Flags().StringP("file", "F", "", "Specify a file to be used with the command")
	mergeCmd.Flags().BoolP("edit", "e", false, "Invoke an editor before committing successful mechanical merge to further edit the auto-generated merge message")
}
