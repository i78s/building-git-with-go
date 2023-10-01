package cmd

import (
	"building-git/lib/command"
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
		stdin := cmd.InOrStdin()
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		options := command.CommitOption{}
		commit, _ := command.NewCommit(dir, args, options, stdin, stdout, stderr)
		code := commit.Run(time.Now())
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}
