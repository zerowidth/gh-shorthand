package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/alfred"
)

var (
	repoIcon        = octicon("repo")
	issueIcon       = octicon("git-pull-request")
	issueListIcon   = octicon("list-ordered")
	pathIcon        = octicon("browser")
	issueSearchIcon = octicon("issue-opened")
	newIssueIcon    = octicon("bug")
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

// octicon is relative to the alfred workflow, so this tells alfred to retrieve
// icons from there rather than relative to this go binary.
func octicon(name string) *alfred.Icon {
	return &alfred.Icon{
		Path: fmt.Sprintf("octicons-%s.png", name),
	}
}
