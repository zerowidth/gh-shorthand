package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(completeCommand)
}

var completeCommand = &cobra.Command{
	Use:   "complete ['input string']",
	Short: "Completion mode",
	Long: `Parse the given input and generate matching Alfred items.

Parses an input string as directly provided by an Alfred script filter innput.
It expects a leading space for the default mode (that is, "space optional" in
the script filter), and uses the first character of the input as a mode string:

  ' ' is default completion mode, for opening repos and issues.
  'i' is issue listing or search
  'n' is new issue in a repo`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}
