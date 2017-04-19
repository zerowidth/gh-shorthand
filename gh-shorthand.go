package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
)

type envVars map[string]string

const (
	// rerunAfter defines how soon the alfred filter is invoked again.
	// This number is an ideal, so true delay must be measured externally.
	rerunAfter = 0.1
	// delay is how long in seconds to wait before showing "processing"
	delay = 0.1
	// searchDelay is how long to wait before issuing a search query
	searchDelay = 0.5
	// socketTimeout is how long to wait before giving up on the backend
	socketTimeout = 100 * time.Millisecond
)

var (
	githubIcon      = octicon("mark-github")
	repoIcon        = octicon("repo")
	pullRequestIcon = octicon("git-pull-request")
	issueListIcon   = octicon("list-ordered")
	pathIcon        = octicon("browser")
	issueIcon       = octicon("issue-opened")
	newIssueIcon    = octicon("bug")
	editorIcon      = octicon("file-code")
	finderIcon      = octicon("file-directory")
	terminalIcon    = octicon("terminal")
	markdownIcon    = octicon("markdown")
	searchIcon      = octicon("search")
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

	// for RPC calls on idle query input:
	shouldRetry := false
	start := queryStart(input, env)
	duration := time.Since(start)

	if len(cfg.DefaultRepo) > 0 && len(parsed.Repo) == 0 && len(parsed.Path) == 0 &&
		((mode == "i" || mode == "n") || len(parsed.Query) == 0) {
		parsed.Repo = cfg.DefaultRepo
		usedDefault = true
	}

	switch mode {
	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if len(parsed.Repo) > 0 && len(parsed.Query) == 0 {
			item := openRepoItem(parsed, usedDefault)
			if len(parsed.Issue) == 0 {
				shouldRetry = retrieveRepoDescription(item, duration, parsed, cfg)
			} else {
				shouldRetry = retrieveIssueTitle(item, duration, parsed, cfg)
			}
			result.AppendItems(item)
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
			if len(parsed.Query) == 0 {
				result.AppendItems(openIssuesAndSearchItems(parsed, usedDefault, fullInput)...)
			} else {
				searchItem := searchIssuesItem(parsed, usedDefault)
				retry, matches := retrieveIssueSearchItems(searchItem, duration, parsed, cfg)
				shouldRetry = retry
				result.AppendItems(searchItem)
				result.AppendItems(matches...)
			}
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

	// if any RPC-decorated items require a re-invocation of the script, save that
	// information in the environment for the next time
	if shouldRetry {
		result.SetVariable("query", input)
		result.SetVariable("s", fmt.Sprintf("%d", start.Unix()))
		result.SetVariable("ns", fmt.Sprintf("%d", start.Nanosecond()))
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

	title += parsed.Annotation(usedDefault)

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

func openIssuesAndSearchItems(parsed *parser.Result, usedDefault bool, fullInput string) (items alfred.Items) {
	extra := parsed.Annotation(usedDefault)

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
		Icon:         searchIcon,
		Autocomplete: fullInput + " ",
	})
	return
}

func searchIssuesItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	extra := parsed.Annotation(usedDefault)
	escaped := url.PathEscape(parsed.Query)
	arg := "open https://github.com/" + parsed.Repo + "/search?utf8=✓&type=Issues&q=" + escaped
	return &alfred.Item{
		UID:   "ghis:" + parsed.Repo,
		Title: "Search issues in " + parsed.Repo + extra + " for " + parsed.Query,
		Arg:   arg,
		Valid: true,
		Icon:  searchIcon,
	}
}

func newIssueItem(parsed *parser.Result, usedDefault bool) *alfred.Item {
	title := "New issue in " + parsed.Repo
	title += parsed.Annotation(usedDefault)

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

	title += parsed.Annotation(usedDefault)

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

	title += parsed.Annotation(usedDefault)

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
		Icon:         issueIcon,
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
		Icon:         issueIcon,
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

func queryStart(input string, env envVars) time.Time {
	if query, ok := env["query"]; ok && query == input {
		if sStr, ok := env["s"]; ok {
			if nsStr, ok := env["ns"]; ok {
				if s, err := strconv.ParseInt(sStr, 10, 64); err == nil {
					if ns, err := strconv.ParseInt(nsStr, 10, 64); err == nil {
						return time.Unix(s, ns)
					}
				}
			}
		}
	}

	return time.Now()
}

func rpcRequest(query string, cfg *config.Config) (shouldRetry bool, results []string, err error) {
	if len(cfg.SocketPath) == 0 {
		return false, results, nil // RPC isn't enabled, don't worry about it
	}
	sock, err := net.Dial("unix", cfg.SocketPath)
	if err != nil {
		return false, results, fmt.Errorf("Could not connect to %s: %s", cfg.SocketPath, err)
	}
	defer sock.Close()
	if err := sock.SetDeadline(time.Now().Add(socketTimeout)); err != nil {
		return false, results, fmt.Errorf("Could not set socket timeout: %s: %s", cfg.SocketPath, err)
	}
	// write query to socket:
	if _, err := sock.Write([]byte(query + "\n")); err != nil {
		return false, results, fmt.Errorf("Could not send query: %s", err)
	}
	// now, read results:
	scanner := bufio.NewScanner(sock)
	if scanner.Scan() {
		status := scanner.Text()
		switch status {
		case "OK":
			for scanner.Scan() {
				results = append(results, scanner.Text())
			}
			return false, results, nil
		case "PENDING":
			return true, results, nil
		case "ERROR":
			for scanner.Scan() {
				results = append(results, scanner.Text())
			}
			if len(results) > 0 {
				err = errors.New(results[0])
			} else {
				err = errors.New("unknown RPC error")
			}
			return false, results, err
		default:
			if err := scanner.Err(); err != nil {
				return false, results, fmt.Errorf("Error when reading RPC response: %s", err)
			}
			return false, results, fmt.Errorf("Unexpected RPC response status: %s", status)
		}
	} else {
		if err := scanner.Err(); err != nil {
			return false, results, fmt.Errorf("Error when reading RPC response: %s", err)
		}
		return false, results, fmt.Errorf("Error: no response from RPC backend: %s", err)
	}
}

func ellipsis(prefix string, duration time.Duration) string {
	return prefix + strings.Repeat(".", int((duration.Nanoseconds()/250000000)%4))
}

// retrieveRepoDescription adds the repo description to the "open repo" item
// using an RPC call.
func retrieveRepoDescription(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg *config.Config) (shouldRetry bool) {
	if duration.Seconds() < delay {
		shouldRetry = true
	} else {
		retry, results, err := rpcRequest("repo:"+parsed.Repo, cfg)
		shouldRetry = retry
		if err != nil {
			item.Subtitle = err.Error()
		} else if shouldRetry {
			item.Subtitle = ellipsis("Retrieving description", duration)
		} else if len(results) > 0 {
			item.Subtitle = results[0]
		} else {
			item.Subtitle = "No description found."
		}
	}

	return shouldRetry
}

// retrieveIssueTitle adds the title to the "open issue" item using an RPC call
func retrieveIssueTitle(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg *config.Config) (shouldRetry bool) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("issue:"+parsed.Repo+"#"+parsed.Issue, cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving issue title", duration)
	} else if len(results) > 0 {
		parts := strings.SplitN(results[0], ":", 2)
		if len(parts) != 2 {
			return
		}
		kind, title := parts[0], parts[1]
		item.Subtitle = item.Title
		item.Title = title
		if kind == "PullRequest" {
			item.Icon = pullRequestIcon
		}
	}

	return
}

func retrieveIssueSearchItems(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg *config.Config) (shouldRetry bool, matches alfred.Items) {
	if duration.Seconds() < searchDelay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("issuesearch:repo:"+parsed.Repo+" "+parsed.Query, cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Searching issues", duration)
	} else if len(results) > 0 {
		for _, result := range results {
			parts := strings.SplitN(result, ":", 4)
			if len(parts) != 4 {
				continue
			}
			repo, number, kind, title := parts[0], parts[1], parts[2], parts[3]
			arg := ""
			icon := issueIcon
			if kind == "Issue" {
				arg = "open https://github.com/" + repo + "/issues/" + number
			} else {
				arg = "open https://github.com/" + repo + "/pull/" + number
				icon = pullRequestIcon
			}

			// no UID so alfred doesn't remember these
			matches = append(matches, &alfred.Item{
				Title:    title,
				Subtitle: fmt.Sprintf("Open %s#%s", repo, number),
				Valid:    true,
				Arg:      arg,
				Icon:     icon,
			})
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
	if result.Variables != nil && len(*result.Variables) > 0 {
		result.Rerun = rerunAfter
	}
}

func printResult(result *alfred.FilterResult) {
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err.Error())
	}
}
