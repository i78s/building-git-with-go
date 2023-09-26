package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "git diff",
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

		cached, _ := cmd.Flags().GetBool("cached")
		staged, _ := cmd.Flags().GetBool("staged")

		patch, _ := cmd.Flags().GetBool("patch")
		noPatch, _ := cmd.Flags().GetBool("no-patch")
		if noPatch {
			patch = false
		}
		options := command.DiffOption{
			Cached: cached || staged,
			Patch:  patch,
		}

		diff, _ := command.NewDiff(dir, args, options, stdout, stderr)
		code := diff.Run()
		os.Exit(code)
	},
}

func init() {
	diffCmd.Flags().Bool("cached", false, "prints the changes staged for commit")
	diffCmd.Flags().Bool("staged", false, "alias for --cached; prints the changes staged for commit")
	diffCmd.Flags().Bool("patch", true, "generate patch (default is true)")
	diffCmd.Flags().Bool("no-patch", false, "do not generate patch (default is false)")
	rootCmd.AddCommand(diffCmd)
}
