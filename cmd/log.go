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
		oneline, _ := cmd.Flags().GetBool("oneline")
		if oneline {
			pretty = "oneline"
			abbrevCommit = true
		}

		decorate, _ := cmd.Flags().GetString("decorate")
		noDecorate, _ := cmd.Flags().GetBool("no-decorate")
		if noDecorate {
			decorate = "no"
		}

		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		writer, cleanup := pager.SetupPager(isTTY, stdout, stderr)
		defer cleanup()

		options := command.LogOption{
			Abbrev:   abbrevCommit,
			Format:   pretty,
			Decorate: decorate,
			IsTty:    isTTY,
		}

		cc, _ := cmd.Flags().GetBool("cc")
		if cc {
			options.Combined = true
			options.Patch = true
		}

		status, _ := command.NewLog(dir, args, options, writer, stderr)
		code := status.Run()
		os.Exit(code)
	},
}

func init() {
	logCmd.Flags().Bool("abbrev-commit", false, "Show only the first few characters of the SHA-1 checksum.")
	logCmd.Flags().String("pretty", "medium", "Set log message format")
	logCmd.Flags().Lookup("pretty").NoOptDefVal = "medium"

	// logCmd.Flags().String("format", "medium", "Alias for --pretty")
	logCmd.Flags().Bool("oneline", false, "Shorthand for --pretty=oneline --abbrev-commit")
	logCmd.Flags().String("decorate", "auto", "Decorate log format")
	logCmd.Flags().Lookup("decorate").NoOptDefVal = "short"
	logCmd.Flags().Bool("no-decorate", false, "Disable decorate")
	logCmd.Flags().Bool("cc", false, "Produce dense combined diff output for merge commits")

	rootCmd.AddCommand(logCmd)
}
