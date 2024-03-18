package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var topics []string

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "git remote",
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

		verbose, _ := cmd.Flags().GetBool("verbose")
		topics, _ := cmd.Flags().GetStringSlice("topic")
		options := command.RemoteOption{
			Verbose: verbose,
			Tracked: topics,
		}

		rm, _ := command.NewRemote(dir, args, options, stdout, stderr)
		code := rm.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(remoteCmd)

	remoteCmd.Flags().BoolP("verbose", "v", true, "Enable verbose output")
	remoteCmd.Flags().StringSliceVarP(&topics, "t", "t", []string{}, "Topics for the item (can specify multiple)")
}
