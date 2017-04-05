package cmd

import (
	"github.com/spf13/cobra"
)

// RootCmd is the default gh-shorthand command, does nothing but print help.
var RootCmd = &cobra.Command{
	Use:   "gh-shorthand",
	Short: "gh-shorthand is a tool to generate Alfred autocomplete items",
	Long: `gh-shorthand parses commands and input and generates autocomplete items
in Alfred's JSON RPC format for use as an Alfred script filter.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
