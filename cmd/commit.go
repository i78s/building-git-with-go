package cmd

import (
	"building-git/lib/command"
	"building-git/lib/command/write_commit"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "git commit",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		message, _ := cmd.Flags().GetString("message")
		file, _ := cmd.Flags().GetString("file")
		edit, _ := cmd.Flags().GetBool("edit")
		options := command.CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: message,
				File:    file,
			},
			Edit: edit,
		}
		commit, _ := command.NewCommit(dir, args, options, stdout, stderr)
		code := commit.Run(time.Now())
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringP("message", "m", "", "Specify a message to associate with the command execution")
	commitCmd.Flags().StringP("file", "F", "", "Specify a file to be used with the command")
	commitCmd.Flags().BoolP("edit", "e", false, "Edit the message taken from file, message or commit object")
}
