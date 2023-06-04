package cmd

import (
	"building-git/lib/command"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [path to init]",
	Short: "git init",
	Long:  ``,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		command.Init(args)
	},
}
