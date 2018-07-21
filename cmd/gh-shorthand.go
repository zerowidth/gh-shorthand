package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/internal/pkg/completion"
)

var rootCmd = &cobra.Command{
	Use: "gh-shorthand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Usage()
	},
}

var completeCommand = &cobra.Command{
	Use: "complete 'input string'",
	Run: func(cmd *cobra.Command, args []string) {
		input := strings.Join(args, " ")
		result := completion.Complete(input)
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "could not generate JSON: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(completeCommand)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
