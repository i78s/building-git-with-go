package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cached bool

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "git diff",
	Long:  ``,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, _args []string) {
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		cached, _ := cmd.Flags().GetBool("cached")
		args := command.DiffOption{
			Cached: cached,
		}

		diff, _ := command.NewDiff(dir, args, stdout, stderr)
		code := diff.Run()
		os.Exit(code)
	},
}

func init() {
	diffCmd.Flags().BoolVar(&porcelain, "cached", false, "prints the changes staged for commit")
	rootCmd.AddCommand(diffCmd)
}
