package cmd

import (
	"building-git/lib/command"
	"os"

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
		code := command.Commit(args, stdout, stderr)
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}
