package main

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
	"os"
	"sort"
	"strings"
)

func main() {
	var input string
	var items = []alfred.Item{}

	if len(os.Args) < 2 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	path, _ := homedir.Expand("~/.gh-shorthand.yml")
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		items = []alfred.Item{errorItem("when loading ~/.gh-shorthand.yml", err.Error())}
		printItems(items)
		return
	}

	printItems(generateItems(cfg, input))
}

var repoIcon = octicon("repo")
var issueIcon = octicon("git-pull-request")
var issueListIcon = octicon("list-ordered")
var pathIcon = octicon("browser")

func generateItems(cfg *config.Config, input string) []alfred.Item {
	items := []alfred.Item{}

	if input == "" {
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
	icon := repoIcon
	usedDefault := false

	// fixme assign default if query given for issue mode
	if cfg.DefaultRepo != "" && result.Repo == "" && result.Query == "" && result.Path == "" {
		result.Repo = cfg.DefaultRepo
		usedDefault = true
	}

	switch mode {
	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if result.Repo != "" && result.Query == "" {
			uid := "gh:" + result.Repo
			title := "Open " + result.Repo
			arg := "open https://github.com/" + result.Repo

			if result.Issue != "" {
				uid += "#" + result.Issue
				title += "#" + result.Issue
				arg += "/issues/" + result.Issue
				icon = issueIcon
			}

			if result.Path != "" {
				uid += result.Path
				title += result.Path
				arg += result.Path
				icon = pathIcon
			}

			if result.Match != "" {
				title += " (" + result.Match
				if result.Issue != "" {
					title += "#" + result.Issue
				}
				title += ")"
			}

			if usedDefault {
				title += " (default repo)"
			}

			items = append(items, alfred.Item{
				UID:   uid,
				Title: title + " on GitHub",
				Arg:   arg,
				Valid: true,
				Icon:  icon,
			})
		}

		if result.Repo == "" && result.Path != "" {
			items = append(items, alfred.Item{
				UID:   "gh:" + result.Path,
				Title: fmt.Sprintf("Open %s on GitHub", result.Path),
				Arg:   "open https://github.com" + result.Path,
				Valid: true,
				Icon:  pathIcon,
			})
		}

		if !strings.Contains(input, " ") {
			for key, repo := range cfg.RepoMap {
				if strings.HasPrefix(key, input) && key != result.Match && repo != result.Repo {
					items = append(items, alfred.Item{
						UID:          "gh:" + repo,
						Title:        fmt.Sprintf("Open %s (%s) on GitHub", repo, key),
						Arg:          "open https://github.com/" + repo,
						Valid:        true,
						Autocomplete: " " + key,
						Icon:         repoIcon,
					})
				}
			}

			if input != "" && result.Repo != input {
				items = append(items, alfred.Item{
					Title:        fmt.Sprintf("Open %s... on GitHub", input),
					Autocomplete: " " + input,
					Valid:        false,
				})
			}
		}
	case "i":
		// repo required, no issue or path, query allowed
		if result.Repo != "" && result.Issue == "" && result.Path == "" {
			uid := "ghi:" + result.Repo
			title := "Open issues for " + result.Repo
			arg := "open https://github.com/" + result.Repo + "/issues"

			if result.Match != "" {
				title += " (" + result.Match + ")"
			}

			if usedDefault {
				title += " (default repo)"
			}

			items = append(items, alfred.Item{
				UID:   uid,
				Title: title,
				Arg:   arg,
				Valid: true,
				Icon:  issueListIcon,
			})
		}

		if !strings.Contains(input, " ") {
			for key, repo := range cfg.RepoMap {
				if strings.HasPrefix(key, input) && key != result.Match && repo != result.Repo {
					items = append(items, alfred.Item{
						UID:          "ghi:" + repo,
						Title:        fmt.Sprintf("Open issues for %s (%s)", repo, key),
						Arg:          "open https://github.com/" + repo + "/issues",
						Valid:        true,
						Autocomplete: "i " + key,
						Icon:         issueListIcon,
					})
				}
			}

			if input != "" && result.Repo != input {
				items = append(items, alfred.Item{
					Title:        fmt.Sprintf("Open issues for %s...", input),
					Autocomplete: "i " + input,
					Valid:        false,
					Icon:         issueListIcon,
				})
			}
		}
	}

	return items
}

func errorItem(context, msg string) alfred.Item {
	return alfred.Item{
		Title:    fmt.Sprintf("Error %s", context),
		Subtitle: msg,
		Icon:     octicon("alert"),
		Valid:    false,
	}
}

func printItems(items []alfred.Item) {
	sort.Sort(alfred.ByTitle(items))
	doc := alfred.Items{Items: items}
	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		panic(err.Error())
	}
}

// octicon is relative to the alfred workflow, so this tells alfred to retrieve
// icons from there rather than relative to this go binary.
func octicon(name string) *alfred.Icon {
	return &alfred.Icon{
		Path: fmt.Sprintf("octicons-%s.png", name),
	}
}
