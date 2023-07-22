package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var porcelain bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "git status",
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

		porcelain, _ := cmd.Flags().GetBool("porcelain")
		options := command.StatusOption{
			Porcelain: porcelain,
		}

		status, _ := command.NewStatus(dir, args, options, stdout, stderr)
		code := status.Run()
		os.Exit(code)
	},
}

func init() {
	statusCmd.Flags().BoolVar(&porcelain, "porcelain", false, "use porcelain format")
	rootCmd.AddCommand(statusCmd)
}
