package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "jit",
	Short: "this is my git",
	Long:  ``,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
