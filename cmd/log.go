package cmd

import (
	"building-git/lib/command"
	"building-git/lib/pager"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var LogFormat = map[string]bool{
	"oneline": true,
	"short":   true,
	"medium":  true,
}

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

		abbrevCommit, _ := cmd.Flags().GetBool("abbrev-commit")
		pretty, _ := cmd.Flags().GetString("pretty")
		if _, exists := LogFormat[pretty]; !exists {
			pretty = "medium"
		}
		oneline, _ := cmd.Flags().GetBool("oneline")
		if oneline {
			pretty = "oneline"
			abbrevCommit = true
		}

		options := command.LogOption{
			Abbrev: abbrevCommit,
			Format: pretty,
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
	logCmd.Flags().Bool("abbrev-commit", false, "Show only the first few characters of the SHA-1 checksum.")
	logCmd.Flags().String("pretty", "medium", "Set log message format")
	// logCmd.Flags().String("format", "medium", "Alias for --pretty")
	logCmd.Flags().Bool("oneline", false, "Shorthand for --pretty=oneline --abbrev-commit")

	rootCmd.AddCommand(logCmd)
}
