package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
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
		var input string
		var items = []*alfred.Item{}

		if len(args) == 0 {
			input = ""
		} else {
			input = strings.Join(args, " ")
		}

		path, _ := homedir.Expand("~/.gh-shorthand.yml")
		cfg, err := config.LoadFromFile(path)
		if err != nil {
			items = []*alfred.Item{errorItem("when loading ~/.gh-shorthand.yml", err.Error())}
		} else {
			items = completeItems(cfg, input)
		}

		printItems(items)
	},
}

func completeItems(cfg *config.Config, input string) []*alfred.Item {
	items := []*alfred.Item{}
	fullInput := input

	if len(input) == 0 {
		return items
	}

	// input includes leading space or leading mode char followed by a space
	var mode string
	if len(input) > 1 && input[0:1] != " " {
		mode = input[0:1]
		input = input[2:]
	} else if len(input) > 0 && input[0:1] == " " {
		mode = " "
		input = input[1:]
	}

	result := parser.Parse(cfg.RepoMap, input)
	usedDefault := false

	if len(cfg.DefaultRepo) > 0 && len(result.Repo) == 0 && len(result.Path) == 0 &&
		((mode == "i" || mode == "n") || len(result.Query) == 0) {
		result.Repo = cfg.DefaultRepo
		usedDefault = true
	}

	switch mode {
	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if len(result.Repo) > 0 && len(result.Query) == 0 {
			items = append(items, openRepoItems(result, usedDefault)...)
		}

		if len(result.Repo) == 0 && len(result.Path) > 0 {
			items = append(items, openPathItem(result.Path))
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			items = append(items,
				autocompleteItems(cfg, input, result,
					autocompleteOpenItem, openEndedOpenItem)...)
		}
	case "i":
		// repo required, no issue or path, query allowed
		if len(result.Repo) > 0 && len(result.Issue) == 0 && len(result.Path) == 0 {
			items = append(items, openIssueItems(result, usedDefault, fullInput)...)
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			items = append(items,
				autocompleteItems(cfg, input, result,
					autocompleteIssueItem, openEndedIssueItem)...)
		}
	case "n":
		// repo required, no issue or path, query allowed
		if len(result.Repo) > 0 && len(result.Issue) == 0 && len(result.Path) == 0 {
			items = append(items, newIssueItems(result, usedDefault)...)
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			items = append(items,
				autocompleteItems(cfg, input, result,
					autocompleteNewIssueItem, openEndedNewIssueItem)...)
		}
	}

	return items
}

func openRepoItems(result *parser.Result, usedDefault bool) (items []*alfred.Item) {
	uid := "gh:" + result.Repo
	title := "Open " + result.Repo
	arg := "open https://github.com/" + result.Repo
	icon := repoIcon

	if len(result.Issue) > 0 {
		uid += "#" + result.Issue
		title += "#" + result.Issue
		arg += "/issues/" + result.Issue
		icon = issueIcon
	}

	if len(result.Path) > 0 {
		uid += result.Path
		title += result.Path
		arg += result.Path
		icon = pathIcon
	}

	if len(result.Match) > 0 {
		title += " (" + result.Match
		if len(result.Issue) > 0 {
			title += "#" + result.Issue
		}
		title += ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	items = append(items, &alfred.Item{
		UID:   uid,
		Title: title + " on GitHub",
		Arg:   arg,
		Valid: true,
		Icon:  icon,
	})
	return items
}

func openPathItem(path string) *alfred.Item {
	return &alfred.Item{
		UID:   "gh:" + path,
		Title: fmt.Sprintf("Open %s on GitHub", path),
		Arg:   "open https://github.com" + path,
		Valid: true,
		Icon:  pathIcon,
	}
}

func openIssueItems(result *parser.Result, usedDefault bool, fullInput string) (items []*alfred.Item) {
	extra := ""
	if len(result.Match) > 0 {
		extra += " (" + result.Match + ")"
	} else if usedDefault {
		extra += " (default repo)"
	}

	if len(result.Query) == 0 {
		items = append(items, &alfred.Item{
			UID:   "ghi:" + result.Repo,
			Title: "Open issues for " + result.Repo + extra,
			Arg:   "open https://github.com/" + result.Repo + "/issues",
			Valid: true,
			Icon:  issueListIcon,
		})
		items = append(items, &alfred.Item{
			Title:        "Search issues in " + result.Repo + extra + " for...",
			Valid:        false,
			Icon:         issueSearchIcon,
			Autocomplete: fullInput + " ",
		})
	} else {
		escaped := url.PathEscape(result.Query)
		arg := "open https://github.com/" + result.Repo + "/search?utf8=✓&type=Issues&q=" + escaped
		items = append(items, &alfred.Item{
			UID:   "ghis:" + result.Repo,
			Title: "Search issues in " + result.Repo + extra + " for " + result.Query,
			Arg:   arg,
			Valid: true,
			Icon:  issueSearchIcon,
		})
	}
	return
}

func newIssueItems(result *parser.Result, usedDefault bool) (items []*alfred.Item) {
	title := "New issue in " + result.Repo
	if len(result.Match) > 0 {
		title += " (" + result.Match + ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	if len(result.Query) == 0 {
		items = append(items, &alfred.Item{
			UID:   "ghn:" + result.Repo,
			Title: title,
			Arg:   "open https://github.com/" + result.Repo + "/issues/new",
			Valid: true,
			Icon:  newIssueIcon,
		})
	} else {
		escaped := url.PathEscape(result.Query)
		arg := "open https://github.com/" + result.Repo + "/issues/new?title=" + escaped
		items = append(items, &alfred.Item{
			UID:   "ghn:" + result.Repo,
			Title: title + ": " + result.Query,
			Arg:   arg,
			Valid: true,
			Icon:  newIssueIcon,
		})
	}
	return
}

func autocompleteOpenItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "gh:" + repo,
		Title:        fmt.Sprintf("Open %s (%s) on GitHub", repo, key),
		Arg:          "open https://github.com/" + repo,
		Valid:        true,
		Autocomplete: " " + key,
		Icon:         repoIcon,
	}
}

func autocompleteIssueItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghi:" + repo,
		Title:        fmt.Sprintf("Open issues for %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/issues",
		Valid:        true,
		Autocomplete: "i " + key,
		Icon:         issueListIcon,
	}
}

func autocompleteNewIssueItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghn:" + repo,
		Title:        fmt.Sprintf("New issue in %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/issues/new",
		Valid:        true,
		Autocomplete: "n " + key,
		Icon:         newIssueIcon,
	}
}

func openEndedOpenItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Open %s... on GitHub", input),
		Autocomplete: " " + input,
		Valid:        false,
	}
}

func openEndedIssueItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Open issues for %s...", input),
		Autocomplete: "i " + input,
		Valid:        false,
		Icon:         issueListIcon,
	}
}

func openEndedNewIssueItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("New issue in %s...", input),
		Autocomplete: "n " + input,
		Valid:        false,
		Icon:         newIssueIcon,
	}
}

func autocompleteItems(cfg *config.Config, input string, result *parser.Result,
	autocompleteItem func(string, string) *alfred.Item,
	openEndedItem func(string) *alfred.Item) (items []*alfred.Item) {
	for key, repo := range cfg.RepoMap {
		if strings.HasPrefix(key, input) && key != result.Match && repo != result.Repo {
			items = append(items, autocompleteItem(key, repo))
		}
	}

	if len(input) > 0 && result.Repo != input {
		items = append(items, openEndedItem(input))
	}
	return
}

func errorItem(context, msg string) *alfred.Item {
	return &alfred.Item{
		Title:    fmt.Sprintf("Error %s", context),
		Subtitle: msg,
		Icon:     octicon("alert"),
		Valid:    false,
	}
}

func printItems(items []*alfred.Item) {
	sort.Sort(alfred.ByTitle(items))
	doc := alfred.Items{Items: items}
	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		panic(err.Error())
	}
}
