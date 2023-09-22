package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var checkOutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "git checkout",
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

		options := command.CheckOutOption{}

		checkout, _ := command.NewCheckOut(dir, args, options, stdout, stderr)
		code := checkout.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(checkOutCmd)
}
