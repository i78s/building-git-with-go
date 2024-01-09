package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var resetMode = string(command.Mixed)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "git reset",
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

		mode, _ := cmd.Flags().GetString("mode")
		options := command.ResetOption{
			Mode: command.ResetMode(mode),
		}

		reset, _ := command.NewReset(dir, args, options, stdout, stderr)
		code := reset.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().StringVar(&resetMode, "soft", string(command.Soft), "Perform a 'soft' reset, keeping changes in the working directory.")
	resetCmd.Flags().StringVar(&resetMode, "hard", string(command.Hard), "Perform a 'hard' reset, discarding changes in the working directory.")
}
