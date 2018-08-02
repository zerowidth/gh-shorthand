package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/internal/pkg/completion"
	"github.com/zerowidth/gh-shorthand/internal/pkg/server"
	"github.com/zerowidth/gh-shorthand/internal/pkg/snippets"
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

var markdownCommand = &cobra.Command{
	Use: "markdown-link",
	Run: func(cmd *cobra.Command, args []string) {
		link := snippets.MarkdownLink(strings.Join(args, " "))
		fmt.Fprintf(os.Stdout, link)
	},
}

var issueReferenceCommand = &cobra.Command{
	Use: "issue-reference",
	Run: func(cmd *cobra.Command, args []string) {
		ref := snippets.IssueReference(strings.Join(args, " "))
		fmt.Fprintf(os.Stdout, ref)
	},
}

func init() {
	rootCmd.AddCommand(completeCommand)
	rootCmd.AddCommand(serverCommand)
	rootCmd.AddCommand(markdownCommand)
	rootCmd.AddCommand(issueReferenceCommand)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
