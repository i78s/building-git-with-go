package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var stage = "0"

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
			Stage:  stage,
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

	diffCmd.Flags().StringVar(&stage, "1", "1", "set stage to 1 (base)")
	diffCmd.Flags().StringVar(&stage, "2", "2", "set stage to 2 (ours)")
	diffCmd.Flags().StringVar(&stage, "3", "3", "set stage to 3 (theirs)")
	diffCmd.Flags().StringVar(&stage, "base", "1", "set stage to 1 (base)")
	diffCmd.Flags().StringVar(&stage, "ours", "2", "set stage to 2 (ours)")
	diffCmd.Flags().StringVar(&stage, "theirs", "3", "set stage to 3 (theirs)")

	rootCmd.AddCommand(diffCmd)
}
