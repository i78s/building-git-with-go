package cmd

import (
	"building-git/lib/command"
	"building-git/lib/pager"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "git log",
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

		options := command.LogOption{}

		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		writer, cleanup := pager.SetupPager(isTTY, stdout, stderr)
		defer cleanup()

		status, _ := command.NewLog(dir, args, options, writer, stderr)
		code := status.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
