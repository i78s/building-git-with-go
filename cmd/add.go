package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [path to add]",
	Short: "git add",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		code := command.Add(dir, args, stdout, stderr)
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
