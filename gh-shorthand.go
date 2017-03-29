package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
)

var (
	repoIcon        = octicon("repo")
	issueIcon       = octicon("git-pull-request")
	issueListIcon   = octicon("list-ordered")
	pathIcon        = octicon("browser")
	issueSearchIcon = octicon("issue-opened")
)

func main() {
	var input string
	var items = []*alfred.Item{}

	if len(os.Args) < 2 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	path, _ := homedir.Expand("~/.gh-shorthand.yml")
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		items = []*alfred.Item{errorItem("when loading ~/.gh-shorthand.yml", err.Error())}
	} else {
		items = generateItems(cfg, input)
	}

	printItems(items)
}

func generateItems(cfg *config.Config, input string) []*alfred.Item {
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
	icon := repoIcon
	usedDefault := false

	// fixme assign default if query given for issue mode
	if len(cfg.DefaultRepo) > 0 && len(result.Repo) == 0 && len(result.Query) == 0 && len(result.Path) == 0 {
		result.Repo = cfg.DefaultRepo
		usedDefault = true
	}

	switch mode {
	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if len(result.Repo) > 0 && len(result.Query) == 0 {
			uid := "gh:" + result.Repo
			title := "Open " + result.Repo
			arg := "open https://github.com/" + result.Repo

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
		}

		if len(result.Repo) == 0 && len(result.Path) > 0 {
			items = append(items, &alfred.Item{
				UID:   "gh:" + result.Path,
				Title: fmt.Sprintf("Open %s on GitHub", result.Path),
				Arg:   "open https://github.com" + result.Path,
				Valid: true,
				Icon:  pathIcon,
			})
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			for key, repo := range cfg.RepoMap {
				if strings.HasPrefix(key, input) && key != result.Match && repo != result.Repo {
					items = append(items, &alfred.Item{
						UID:          "gh:" + repo,
						Title:        fmt.Sprintf("Open %s (%s) on GitHub", repo, key),
						Arg:          "open https://github.com/" + repo,
						Valid:        true,
						Autocomplete: " " + key,
						Icon:         repoIcon,
					})
				}
			}

			if len(input) > 0 && result.Repo != input {
				items = append(items, &alfred.Item{
					Title:        fmt.Sprintf("Open %s... on GitHub", input),
					Autocomplete: " " + input,
					Valid:        false,
				})
			}
		}
	case "i":
		// repo required, no issue or path, query allowed
		if len(result.Repo) > 0 && len(result.Issue) == 0 && len(result.Path) == 0 {
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
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			for key, repo := range cfg.RepoMap {
				if strings.HasPrefix(key, input) && key != result.Match && repo != result.Repo {
					items = append(items, &alfred.Item{
						UID:          "ghi:" + repo,
						Title:        fmt.Sprintf("Open issues for %s (%s)", repo, key),
						Arg:          "open https://github.com/" + repo + "/issues",
						Valid:        true,
						Autocomplete: "i " + key,
						Icon:         issueListIcon,
					})
				}
			}

			if len(input) > 0 && result.Repo != input {
				items = append(items, &alfred.Item{
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

// octicon is relative to the alfred workflow, so this tells alfred to retrieve
// icons from there rather than relative to this go binary.
func octicon(name string) *alfred.Icon {
	return &alfred.Icon{
		Path: fmt.Sprintf("octicons-%s.png", name),
	}
}
