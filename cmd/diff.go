package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cached bool
	staged bool
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

		cachedFlag, _ := cmd.Flags().GetBool("cached")
		stagedFlag, _ := cmd.Flags().GetBool("staged")
		options := command.DiffOption{
			Cached: cachedFlag || stagedFlag,
		}

		diff, _ := command.NewDiff(dir, args, options, stdout, stderr)
		code := diff.Run()
		os.Exit(code)
	},
}

func init() {
	diffCmd.Flags().BoolVar(&cached, "cached", false, "prints the changes staged for commit")
	diffCmd.Flags().BoolVar(&staged, "staged", false, "alias for --cached; prints the changes staged for commit")
	rootCmd.AddCommand(diffCmd)
}
