package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cherryPickCmd = &cobra.Command{
	Use:   "cherry-pick",
	Short: "git cherry-pick",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}

		mode, _ := cmd.Flags().GetString("mode")
		options := command.CherryPickOption{
			Mode: command.MergeMode(mode),
		}

		rm, _ := command.NewCherryPick(dir, args, options, stdout, stderr)
		code := rm.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(cherryPickCmd)

	cherryPickCmd.Flags().StringVar(&mergeMode, "continue", string(command.Continue), "Resume command execution from a saved state")
}
