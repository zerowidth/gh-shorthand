package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
)

type envVars map[string]string

// rerunAfter defines how soon the alfred filter is invoked again
const rerunAfter = 0.1

var (
	repoIcon        = octicon("repo")
	issueIcon       = octicon("git-pull-request")
	issueListIcon   = octicon("list-ordered")
	pathIcon        = octicon("browser")
	issueSearchIcon = octicon("issue-opened")
	newIssueIcon    = octicon("bug")
	editorIcon      = octicon("file-code")
	finderIcon      = octicon("file-directory")
	terminalIcon    = octicon("terminal")
	markdownIcon    = octicon("markdown")
)

func main() {
	var input string
	var result = alfred.NewFilterResult()

	if len(os.Args) == 1 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	path, _ := homedir.Expand("~/.gh-shorthand.yml")
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		result.AppendItems(errorItem("when loading ~/.gh-shorthand.yml", err.Error()))
	} else {
		vars := getEnvironment()
		appendParsedItems(result, cfg, vars, input)
	}

	finalizeResult(result)
	printResult(result)
}

func appendParsedItems(result *alfred.FilterResult, cfg *config.Config, env map[string]string, input string) {
	fullInput := input

	if len(input) == 0 {
		return
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

	parsed := parser.Parse(cfg.RepoMap, input)
	usedDefault := false

	if len(cfg.DefaultRepo) > 0 && len(parsed.Repo) == 0 && len(parsed.Path) == 0 &&
		((mode == "i" || mode == "n") || len(parsed.Query) == 0) {
		parsed.Repo = cfg.DefaultRepo
		usedDefault = true
	}

	switch mode {
	case "b":
		var count int64 = 1

		if countStr, ok := env["count"]; ok {
			count, _ = strconv.ParseInt(countStr, 10, 64)
		}

		if query, ok := env["query"]; ok {
			if query == input {
				count++
			} else {
				count = 1
			}
		}
		result.SetVariable("count", fmt.Sprintf("%d", count))
		result.SetVariable("query", input)

		result.AppendItems(
			&alfred.Item{
				Title:    fmt.Sprintf("Processing %q", input),
				Subtitle: fmt.Sprintf("count: %d", count),
			})

	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if len(parsed.Repo) > 0 && len(parsed.Query) == 0 {
			result.AppendItems(openRepoItem(parsed, usedDefault))
		}

		if len(parsed.Repo) == 0 && len(parsed.Path) > 0 {
			result.AppendItems(openPathItem(parsed.Path))
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteItems(cfg, input, parsed,
					autocompleteOpenItem, openEndedOpenItem)...)
		}
	case "i":
		// repo required, no issue or path, query allowed
		if len(parsed.Repo) > 0 && len(parsed.Issue) == 0 && len(parsed.Path) == 0 {
			result.AppendItems(openIssueItems(parsed, usedDefault, fullInput)...)
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteItems(cfg, input, parsed,
					autocompleteIssueItem, openEndedIssueItem)...)
		}
	case "n":
		// repo required, no issue or path, query allowed
		if len(parsed.Repo) > 0 && len(parsed.Issue) == 0 && len(parsed.Path) == 0 {
			result.AppendItems(newIssueItem(parsed, usedDefault))
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteItems(cfg, input, parsed,
					autocompleteNewIssueItem, openEndedNewIssueItem)...)
		}
	case "m":
		// repo required, issue optional
		if len(parsed.Repo) > 0 && len(parsed.Path) == 0 && len(parsed.Query) == 0 {
			result.AppendItems(markdownLinkItem(parsed, usedDefault))
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteItems(cfg, input, parsed,
					autocompleteMarkdownLinkItem, openEndedMarkdownLinkItem)...)
		}
	case "r":
		// repo required, issue required (issue handled in issueReferenceItem)
		if len(parsed.Repo) > 0 && len(parsed.Path) == 0 && len(parsed.Query) == 0 {
			result.AppendItems(issueReferenceItem(parsed, usedDefault))
		}

		if len(input) > 0 && !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteItems(cfg, input, parsed,
					autocompleteIssueReferenceItem, openEndedIssueReferenceItem)...)
		}
	case "e":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "ghe", "edit", "Edit", editorIcon)...)
	case "o":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "gho", "finder", "Open Finder in", editorIcon)...)
	case "t":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "ght", "term", "Open terminal in", editorIcon)...)
	}

}

func actionItems(dirs map[string]string, search, uidPrefix, action, desc string, icon *alfred.Icon) (items alfred.Items) {
	projects := map[string]string{}
	projectNames := []string{}

	for base, expanded := range dirs {
		for _, dirname := range findProjectDirs(expanded) {
			short := filepath.Join(base, dirname)
			full := filepath.Join(expanded, dirname)
			projects[short] = full
			projectNames = append(projectNames, short)
		}
	}

	if len(search) > 0 {
		projectNames = fuzzy.Find(search, projectNames)
	}

	for _, short := range projectNames {
		items = append(items, &alfred.Item{
			UID:   uidPrefix + ":" + short,
			Title: desc + " " + short,
			Arg:   action + " " + projects[short],
			Valid: true,
			Icon:  icon,
		})
	}

	return
}

func openRepoItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	uid := "gh:" + parsed.Repo
	title := "Open " + parsed.Repo
	arg := "open https://github.com/" + parsed.Repo
	icon := repoIcon

	if len(parsed.Issue) > 0 {
		uid += "#" + parsed.Issue
		title += "#" + parsed.Issue
		arg += "/issues/" + parsed.Issue
		icon = issueIcon
	}

	if len(parsed.Path) > 0 {
		uid += parsed.Path
		title += parsed.Path
		arg += parsed.Path
		icon = pathIcon
	}

	if len(parsed.Match) > 0 {
		title += " (" + parsed.Match
		if len(parsed.Issue) > 0 {
			title += "#" + parsed.Issue
		}
		title += ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	return &alfred.Item{
		UID:   uid,
		Title: title,
		Arg:   arg,
		Valid: true,
		Icon:  icon,
	}
}

func openPathItem(path string) *alfred.Item {
	return &alfred.Item{
		UID:   "gh:" + path,
		Title: fmt.Sprintf("Open %s", path),
		Arg:   "open https://github.com" + path,
		Valid: true,
		Icon:  pathIcon,
	}
}

func openIssueItems(parsed *parser.Result, usedDefault bool, fullInput string) (items alfred.Items) {
	extra := ""
	if len(parsed.Match) > 0 {
		extra += " (" + parsed.Match + ")"
	} else if usedDefault {
		extra += " (default repo)"
	}

	if len(parsed.Query) == 0 {
		items = append(items, &alfred.Item{
			UID:   "ghi:" + parsed.Repo,
			Title: "Open issues for " + parsed.Repo + extra,
			Arg:   "open https://github.com/" + parsed.Repo + "/issues",
			Valid: true,
			Icon:  issueListIcon,
		})
		items = append(items, &alfred.Item{
			Title:        "Search issues in " + parsed.Repo + extra + " for...",
			Valid:        false,
			Icon:         issueSearchIcon,
			Autocomplete: fullInput + " ",
		})
	} else {
		escaped := url.PathEscape(parsed.Query)
		arg := "open https://github.com/" + parsed.Repo + "/search?utf8=✓&type=Issues&q=" + escaped
		items = append(items, &alfred.Item{
			UID:   "ghis:" + parsed.Repo,
			Title: "Search issues in " + parsed.Repo + extra + " for " + parsed.Query,
			Arg:   arg,
			Valid: true,
			Icon:  issueSearchIcon,
		})
	}
	return
}

func newIssueItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	title := "New issue in " + parsed.Repo
	if len(parsed.Match) > 0 {
		title += " (" + parsed.Match + ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	if len(parsed.Query) == 0 {
		return &alfred.Item{
			UID:   "ghn:" + parsed.Repo,
			Title: title,
			Arg:   "open https://github.com/" + parsed.Repo + "/issues/new",
			Valid: true,
			Icon:  newIssueIcon,
		}
	}

	escaped := url.PathEscape(parsed.Query)
	arg := "open https://github.com/" + parsed.Repo + "/issues/new?title=" + escaped
	return &alfred.Item{
		UID:   "ghn:" + parsed.Repo,
		Title: title + ": " + parsed.Query,
		Arg:   arg,
		Valid: true,
		Icon:  newIssueIcon,
	}
}

func markdownLinkItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	uid := "ghm:" + parsed.Repo
	title := "Insert Markdown link to " + parsed.Repo
	desc := parsed.Repo
	link := "https://github.com/" + parsed.Repo
	icon := markdownIcon

	if len(parsed.Issue) > 0 {
		uid += "#" + parsed.Issue
		title += "#" + parsed.Issue
		desc += "#" + parsed.Issue
		link += "/issues/" + parsed.Issue
		icon = issueIcon
	}

	if len(parsed.Match) > 0 {
		title += " (" + parsed.Match
		if len(parsed.Issue) > 0 {
			title += "#" + parsed.Issue
		}
		title += ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	return &alfred.Item{
		UID:   uid,
		Title: title,
		Arg:   fmt.Sprintf("paste [%s](%s)", desc, link),
		Valid: true,
		Icon:  icon,
	}
}

func issueReferenceItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	title := "Insert issue reference to " + parsed.Repo
	ref := parsed.Repo

	if len(parsed.Issue) > 0 {
		title += "#" + parsed.Issue
		ref += "#" + parsed.Issue
	} else {
		title += "#..."
	}

	if len(parsed.Match) > 0 {
		title += " (" + parsed.Match
		if len(parsed.Issue) > 0 {
			title += "#" + parsed.Issue
		} else {
			title += "#..."
		}
		title += ")"
	} else if usedDefault {
		title += " (default repo)"
	}

	if len(parsed.Issue) > 0 {

		return &alfred.Item{
			UID:   "ghr:" + ref,
			Title: title,
			Arg:   "paste " + ref,
			Valid: true,
			Icon:  issueIcon,
		}

	}

	auto := "r " + parsed.Repo
	if len(parsed.Match) > 0 {
		auto = "r " + parsed.Match
	}
	return &alfred.Item{
		Title:        title,
		Autocomplete: auto + " ",
		Valid:        false,
		Icon:         issueIcon,
	}
}

func autocompleteOpenItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "gh:" + repo,
		Title:        fmt.Sprintf("Open %s (%s)", repo, key),
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

func autocompleteMarkdownLinkItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghm:" + repo,
		Title:        fmt.Sprintf("Insert Markdown link to %s (%s)", repo, key),
		Valid:        true,
		Arg:          fmt.Sprintf("paste [%s](https://github.com/%s)", repo, repo),
		Autocomplete: "m " + key,
		Icon:         markdownIcon,
	}
}

func autocompleteIssueReferenceItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghr:" + repo,
		Title:        fmt.Sprintf("Insert issue reference to %s#... (%s#...)", repo, key),
		Valid:        false,
		Autocomplete: "r " + key + " ",
		Icon:         issueSearchIcon,
	}
}

func openEndedOpenItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Open %s...", input),
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

func openEndedMarkdownLinkItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Insert Markdown link to %s...", input),
		Autocomplete: "m " + input,
		Valid:        false,
		Icon:         markdownIcon,
	}
}

func openEndedIssueReferenceItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Insert issue reference to %s...", input),
		Autocomplete: "r " + input,
		Valid:        false,
		Icon:         issueSearchIcon,
	}
}

func autocompleteItems(cfg *config.Config, input string, parsed *parser.Result,
	autocompleteItem func(string, string) *alfred.Item,
	openEndedItem func(string) *alfred.Item) (items alfred.Items) {
	for key, repo := range cfg.RepoMap {
		if strings.HasPrefix(key, input) && key != parsed.Match && repo != parsed.Repo {
			items = append(items, autocompleteItem(key, repo))
		}
	}

	if len(input) > 0 && parsed.Repo != input {
		items = append(items, openEndedItem(input))
	}
	return
}

func findProjectDirs(root string) (dirs []string) {
	if entries, err := ioutil.ReadDir(root); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, entry.Name())
			}
		}
	}
	return
}

// octicon is relative to the alfred workflow, so this tells alfred to retrieve
// icons from there rather than relative to this go binary.
func octicon(name string) *alfred.Icon {
	return &alfred.Icon{
		Path: fmt.Sprintf("octicons-%s.png", name),
	}
}

func errorItem(context, msg string) *alfred.Item {
	return &alfred.Item{
		Title:    fmt.Sprintf("Error %s", context),
		Subtitle: msg,
		Icon:     octicon("alert"),
		Valid:    false,
	}
}

// getEnvironment parses the environment and returns a map.
func getEnvironment() envVars {
	env := envVars{}
	for _, entry := range os.Environ() {
		pair := strings.SplitN(entry, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}

func finalizeResult(result *alfred.FilterResult) {
	sort.Sort(alfred.ByValidAndTitle(result.Items))
	if result.Variables != nil && len(*result.Variables) > 0 {
		result.Rerun = rerunAfter
	}
}

func printResult(result *alfred.FilterResult) {
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err.Error())
	}
}
