package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/internal/pkg/completion"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
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

		cfg, cfgErr := config.LoadFromDefault()
		env := completion.LoadAlfredEnvironment(input)
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

func init() {
	rootCmd.AddCommand(completeCommand)
	rootCmd.AddCommand(serverCommand)
	rootCmd.AddCommand(markdownCommand)
	rootCmd.AddCommand(issueReferenceCommand)

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
