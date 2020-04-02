package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/pkg/completion"
	"github.com/zerowidth/gh-shorthand/pkg/config"
	"github.com/zerowidth/gh-shorthand/pkg/rpc"
	"github.com/zerowidth/gh-shorthand/pkg/server"
	"github.com/zerowidth/gh-shorthand/pkg/snippets"
)

var rootCmd = &cobra.Command{
	Use: "gh-shorthand",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Usage()
	},
}

var includeRPC bool
var completeCommand = &cobra.Command{
	Use: "complete 'input string'",
	Run: func(cmd *cobra.Command, args []string) {
		input := strings.Join(args, " ")

		cfg, cfgErr := config.LoadFromDefault()
		env := completion.LoadAlfredEnvironment(input)
		if includeRPC {
			// override start time so it's in the past
			env.Start = time.Now().Add(-time.Minute)
		}

		result := completion.Complete(cfg, env)

		// only include config loading error result if there was any input
		if cfgErr != nil && len(env.Query) > 0 {
			result.AppendItems(completion.ErrorItem(fmt.Sprintf("Could not load config from %s", config.Filename), cfgErr.Error()))
		}

		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "could not generate JSON: %s\n", err)
		}
	},
}

var markdownDescription bool
var markdownCommand = &cobra.Command{
	Use:   "markdown-link",
	Short: "Generate a markdown link from the given input",
	Long: `Generates a markdown link for an issue or PR.

For example:

	github.com/zerowidth/gh-shorthand/issues/1

will generate a markdown link:

	[zerowidth/gh-shorthand#1](...)

If --description is set and RPC is configured, the markdown link will include a
description from the issue or PR's title.
`,
	Aliases: []string{"ml"},
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadFromDefault()
		input := strings.Join(args, " ")
		if err != nil {
			fmt.Fprintf(os.Stdout, "%s (error: %s)", input, err.Error())
		}
		rpcClient := rpc.NewClient(cfg.SocketPath)
		link := snippets.MarkdownLink(rpcClient, input, markdownDescription)
		fmt.Fprint(os.Stdout, link)
	},
}

var issueReferenceCommand = &cobra.Command{
	Use: "issue-reference",
	Run: func(cmd *cobra.Command, args []string) {
		ref := snippets.IssueReference(strings.Join(args, " "))
		fmt.Fprint(os.Stdout, ref)
	},
}

var serverCommand = &cobra.Command{
	Use:   "server",
	Short: "Run or manage a gh-shorthand server",
}

var serverRun = &cobra.Command{
	Use:   "run",
	Short: "Run a gh-shorthand server in the foreground",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := svc.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

var serverInstall = &cobra.Command{
	Use:   "install",
	Short: "Install the gh-shorthand server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := service.Control(svc, "install")
		if err != nil {
			log.Fatal(err)
		}
	},
}
var serverRemove = &cobra.Command{
	Use:   "remove",
	Short: "Remove the gh-shorthand server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := service.Control(svc, "uninstall")
		if err != nil {
			log.Fatal(err)
		}
	},
}
var serverStart = &cobra.Command{
	Use:   "start",
	Short: "Start the gh-shorthand server in the background",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := service.Control(svc, "start")
		if err != nil {
			log.Fatal(err)
		}
	},
}

var serverStop = &cobra.Command{
	Use:   "stop",
	Short: "Stop the gh-shorthand server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := service.Control(svc, "stop")
		if err != nil {
			log.Fatal(err)
		}
	},
}

var serverRestart = &cobra.Command{
	Use:   "restart",
	Short: "Restart the gh-shorthand server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		svc := server.Service(cfg)
		err := service.Control(svc, "restart")
		if err != nil {
			log.Fatal(err)
		}
	},
}

var editorScriptCommand = &cobra.Command{
	Use:   "editor",
	Short: "Emits an editor script for opening a $path",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.MustLoadFromDefault()
		script, err := cfg.OpenEditorScript()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(script)
	},
}

func init() {
	markdownCommand.PersistentFlags().BoolVarP(
		&markdownDescription,
		"description", "d", false,
		"include description of the issue or PR. Requires RPC.")

	completeCommand.PersistentFlags().BoolVarP(
		&includeRPC,
		"include-rpc", "r", false,
		"force an RPC request for the input (used for debugging)")

	rootCmd.AddCommand(completeCommand)
	rootCmd.AddCommand(serverCommand)
	rootCmd.AddCommand(markdownCommand)
	rootCmd.AddCommand(issueReferenceCommand)
	rootCmd.AddCommand(editorScriptCommand)

	serverCommand.AddCommand(serverRun)
	serverCommand.AddCommand(serverInstall)
	serverCommand.AddCommand(serverRemove)
	serverCommand.AddCommand(serverStart)
	serverCommand.AddCommand(serverStop)
	serverCommand.AddCommand(serverRestart)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
