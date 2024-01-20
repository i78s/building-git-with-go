package cmd

import (
	"building-git/lib/command"
	"building-git/lib/command/write_commit"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var reuse string

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
		reuseMessage, _ := cmd.Flags().GetString("reuse-message")
		reeditMessage, _ := cmd.Flags().GetString("reedit-message")
		if edit && reuseMessage == "" && reeditMessage == "" {
			edit = false
		}
		amend, _ := cmd.Flags().GetBool("amend")
		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		options := command.CommitOption{
			ReadOption: write_commit.ReadOption{
				Message: message,
				File:    file,
			},
			Edit:  edit,
			Reuse: reuse,
			Amend: amend,
			IsTTY: isTTY,
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
	commitCmd.Flags().StringVarP(&reuse, "reuse-message", "C", "", "Reuse the message from the specified commit without launching an editor")
	commitCmd.Flags().StringVarP(&reuse, "reedit-message", "c", "", "Use the message from the specified commit as the starting point for the new commit message in the editor")
	commitCmd.Flags().Bool("amend", false, "Replace the tip of the current branch by creating a new commit")
}
