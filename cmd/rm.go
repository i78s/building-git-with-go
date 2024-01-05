package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "git rm",
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

		cached, _ := cmd.Flags().GetBool("cached")
		force, _ := cmd.Flags().GetBool("force")
		recursive, _ := cmd.Flags().GetBool("recursive")
		options := command.RmOption{
			Cached:    cached,
			Force:     force,
			Recursive: recursive,
		}

		rm, _ := command.NewRm(dir, args, options, stdout, stderr)
		code := rm.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().Bool("cached", false, "Remove files from the index only, leaving them in the working directory")
	rmCmd.Flags().Bool("f", false, "Force removal of files, overriding checks for modifications or staging status")
	rmCmd.Flags().Bool("r", false, "Allow recursive removal when a leading directory name is given")
}
