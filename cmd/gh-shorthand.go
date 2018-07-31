package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/internal/pkg/completion"
	"github.com/zerowidth/gh-shorthand/internal/pkg/server"
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
		env := completion.AlfredEnvironment(input)
		result := completion.Complete(env)
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "could not generate JSON: %s\n", err)
		}
	},
}

var serverCommand = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(completeCommand)
	rootCmd.AddCommand(serverCommand)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
