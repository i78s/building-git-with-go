package cmd

import (
	"building-git/lib/command"
	"building-git/lib/pager"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	abbrevCommit bool
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

		abbrevCommitFlag, _ := cmd.Flags().GetBool("abbrev-commit")
		options := command.LogOption{
			Abbrev: abbrevCommitFlag,
		}

		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		writer, cleanup := pager.SetupPager(isTTY, stdout, stderr)
		defer cleanup()

		status, _ := command.NewLog(dir, args, options, writer, stderr)
		code := status.Run()
		os.Exit(code)
	},
}

func init() {
	logCmd.Flags().BoolVar(&abbrevCommit, "abbrev-commit", false, "Show only the first few characters of the SHA-1 checksum.")
	rootCmd.AddCommand(logCmd)
}
