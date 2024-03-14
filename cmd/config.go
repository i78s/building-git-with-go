package cmd

import (
	"building-git/lib/command"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var file string
var add string
var replace string
var getAll string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "git config",
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

		file, _ := cmd.Flags().GetString("file")
		add, _ := cmd.Flags().GetString("add")
		replace, _ := cmd.Flags().GetString("replace-all")
		getAll, _ := cmd.Flags().GetString("get-all")
		options := command.ConfigOption{
			File:    file,
			Add:     add,
			Replace: replace,
			GetAll:  getAll,
		}

		rm, _ := command.NewConfig(dir, args, options, stdout, stderr)
		code := rm.Run()
		os.Exit(code)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.PersistentFlags().StringVarP(&file, "file", "f", "", "Specify the config file")
	configCmd.PersistentFlags().StringVar(&add, "add", "", "Add a new name")
	configCmd.PersistentFlags().StringVar(&replace, "replace-all", "", "Replace all with a new name")
	configCmd.PersistentFlags().StringVar(&getAll, "get-all", "", "Get all names")

	configCmd.PersistentFlags().Bool("local", false, "Use local config")
	configCmd.PersistentFlags().Bool("global", false, "Use global config")
	configCmd.PersistentFlags().Bool("system", false, "Use system config")

	cobra.OnInitialize(func() {
		if rootCmd.PersistentFlags().Lookup("local").Changed {
			file = "local"
		}
		if rootCmd.PersistentFlags().Lookup("global").Changed {
			file = "global"
		}
		if rootCmd.PersistentFlags().Lookup("system").Changed {
			file = "system"
		}
	})
}
