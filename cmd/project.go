package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(projectCommand)
}

var projectCommand = &cobra.Command{
	Use:   "project <action>",
	Short: "Project mode",
	Long: `Generates a list of Alfred items with the given <action>, for every
directory in the configured project paths ("project_dirs" key in the config
file). This is intended to be used as a single-shot Alfred script filter,
allowing Alfred to filter the results.

Action can be one of:

  edit   - for opening an editor with the project directory
  finder - for opening a project directory in Finder
  term   - for opening a terminal in a project directory`,

	Run: func(cmd *cobra.Command, args []string) {
	},
}
