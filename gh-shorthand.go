package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
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

	// how long in seconds to wait before requesting repo title or issue details
	delay = 0.1
	// how long to wait before issuing a search query
	searchDelay = 0.5
	// how long to wait before listing recent issues in a repo
	issueListDelay = 1.0

	// how long to wait before giving up on the backend
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
	commitIcon      = octicon("git-commit")

	issueIconOpen         = octicon("issue-opened_open")
	issueIconClosed       = octicon("issue-closed_closed")
	pullRequestIconOpen   = octicon("git-pull-request_open")
	pullRequestIconClosed = octicon("git-pull-request_closed")
	pullRequestIconMerged = octicon("git-merge_merged")

	// the minimum length of 7 is enforced elsewhere
	sha1Regexp = regexp.MustCompile(`[0-9a-f]{1,40}$`)

	repoDefaultItem = &alfred.Item{
		Title:        "Open repositories and issues on GitHub",
		Autocomplete: " ",
		Icon:         repoIcon,
	}
	issueListDefaultItem = &alfred.Item{
		Title:        "List and search issues on GitHub",
		Autocomplete: "i ",
		Icon:         issueListIcon,
	}
	newIssueDefaultItem = &alfred.Item{
		Title:        "New issue on GitHub",
		Autocomplete: "n ",
		Icon:         newIssueIcon,
	}
	commitDefaultItem = &alfred.Item{
		Title:        "Find a commit in a GitHub repository",
		Autocomplete: "c ",
		Icon:         commitIcon,
	}
	markdownLinkDefaultItem = &alfred.Item{
		Title:        "Insert Markdown link to a GitHub repository or issue",
		Autocomplete: "m ",
		Icon:         markdownIcon,
	}
	issueReferenceDefaultItem = &alfred.Item{
		Title:        "Insert issue reference shorthand for a GitHub repository or issue",
		Autocomplete: "r ",
		Icon:         issueIcon,
	}
	editProjectDefaultItem = &alfred.Item{
		Title:        "Edit a project",
		Autocomplete: "e ",
		Icon:         editorIcon,
	}
	openFinderDefaultItem = &alfred.Item{
		Title:        "Open a project directory in Finder",
		Autocomplete: "o ",
		Icon:         finderIcon,
	}
	openTerminalDefaultItem = &alfred.Item{
		Title:        "Open terminal in a project",
		Autocomplete: "t ",
		Icon:         terminalIcon,
	}
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
	cfg, configErr := config.LoadFromFile(path)

	vars := getEnvironment()
	appendParsedItems(result, cfg, vars, input)

	// only show the config error when needed (i.e. there's input)
	if configErr != nil && len(input) > 0 {
		result.AppendItems(errorItem("Could not load config from ~/.gh-shorthand.yml", configErr.Error()))
	}

	finalizeResult(result)
	printResult(result)
}

func appendParsedItems(result *alfred.FilterResult, cfg *config.Config, env map[string]string, input string) {
	fullInput := input

	// input includes leading space or leading mode char followed by a space
	var mode string
	if len(input) > 1 {
		mode = input[0:1]
		if mode == " " {
			input = input[1:]
		} else {
			if input[1:2] != " " {
				return
			}
			input = input[2:]
		}
	} else if len(input) > 0 {
		mode = input[0:1]
		input = ""
	}

	parsed := parser.Parse(cfg.RepoMap, cfg.UserMap, input)

	// for RPC calls on idle query input:
	shouldRetry := false
	start := queryStart(input, env)
	duration := time.Since(start)

	if !parsed.HasRepo() && len(cfg.DefaultRepo) > 0 && !parsed.HasPath() &&
		((mode == "i" || mode == "n" || mode == "c") ||
			(parsed.HasIssue() || len(parsed.Query) == 0)) {
		parsed.SetRepo(cfg.DefaultRepo)
	}

	switch mode {
	case "": // no input, show default items
		result.AppendItems(
			repoDefaultItem,
			issueListDefaultItem,
			newIssueDefaultItem,
			commitDefaultItem,
			markdownLinkDefaultItem,
			issueReferenceDefaultItem,
			editProjectDefaultItem,
			openFinderDefaultItem,
			openTerminalDefaultItem,
		)

	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if parsed.HasRepo() &&
			(parsed.HasIssue() || parsed.HasPath() || len(parsed.Query) == 0) {
			item := openRepoItem(parsed)
			if parsed.HasIssue() {
				shouldRetry = retrieveIssueTitle(item, duration, parsed, cfg)
			} else {
				shouldRetry = retrieveRepoDescription(item, duration, parsed, cfg)
			}
			result.AppendItems(item)
		}

		if !parsed.HasRepo() && parsed.HasPath() {
			result.AppendItems(openPathItem(parsed.Path()))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteOpenItem, openEndedOpenItem)...)
	case "i":
		// repo required, no issue or path, query allowed
		if parsed.HasRepo() && !parsed.HasPath() {
			if len(parsed.Query) == 0 {
				issuesItem := openIssuesItem(parsed)
				retry, matches := retrieveIssueList(issuesItem, duration, parsed, cfg)
				shouldRetry = retry
				result.AppendItems(issuesItem)
				result.AppendItems(searchIssuesItem(parsed, fullInput))
				result.AppendItems(matches...)
			} else {
				searchItem := searchIssuesItem(parsed, fullInput)
				retry, matches := retrieveIssueSearchItems(searchItem, duration, parsed, cfg)
				shouldRetry = retry
				result.AppendItems(searchItem)
				result.AppendItems(matches...)
			}
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteIssueItem, openEndedIssueItem)...)
	case "n":
		// repo required, no issue or path, query allowed
		if parsed.HasRepo() && !parsed.HasIssue() && !parsed.HasPath() {
			result.AppendItems(newIssueItem(parsed))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteNewIssueItem, openEndedNewIssueItem)...)
	case "c":
		// repo required, query must look like a SHA of at least 7 hex digits.
		if parsed.HasRepo() && !parsed.HasPath() {
			isSHA1 := sha1Regexp.MatchString(parsed.Query)
			if len(parsed.Query) >= 7 && isSHA1 {
				searchItem := commitSearchItem(parsed, true)
				retry, matches := retrieveIssueSearchItems(searchItem, duration, parsed, cfg)
				shouldRetry = retry
				result.AppendItems(searchItem)
				result.AppendItems(matches...)
			} else if len(parsed.Query) == 0 || isSHA1 {
				result.AppendItems(commitSearchItem(parsed, false))
			}
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteCommitSearchItem, openEndedCommitSearchItem)...)
	case "m":
		// repo required, issue optional
		if parsed.HasRepo() && !parsed.HasPath() && (parsed.HasIssue() || len(parsed.Query) == 0) {
			result.AppendItems(markdownLinkItem(parsed))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteMarkdownLinkItem, openEndedMarkdownLinkItem)...)
	case "r":
		// repo required, issue required (issue handled in issueReferenceItem)
		if parsed.HasRepo() && !parsed.HasPath() && (parsed.HasIssue() || len(parsed.Query) == 0) {
			result.AppendItems(issueReferenceItem(parsed))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteIssueReferenceItem, openEndedIssueReferenceItem)...)
	case "e":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "ghe", "edit", "Edit", editorIcon)...)
	case "o":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "gho", "finder", "Open Finder in", finderIcon)...)
	case "t":
		result.AppendItems(
			actionItems(cfg.ProjectDirMap(), input, "ght", "term", "Open terminal in", terminalIcon)...)
	}

	// if any RPC-decorated items require a re-invocation of the script, save that
	// information in the environment for the next time
	if shouldRetry {
		result.SetVariable("query", input)
		result.SetVariable("s", fmt.Sprintf("%d", start.Unix()))
		result.SetVariable("ns", fmt.Sprintf("%d", start.Nanosecond()))
	}

	// automatically copy "open <url>" urls to copy/large text
	for _, item := range result.Items {
		if item.Text == nil && len(item.Arg) > 5 && strings.HasPrefix(item.Arg, "open ") {
			url := item.Arg[5:]
			item.Text = &alfred.Text{Copy: url, LargeType: url}
		}
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
			Text:  &alfred.Text{Copy: projects[short], LargeType: projects[short]},
			Valid: true,
			Icon:  icon,
		})
	}

	return
}

func openRepoItem(parsed *parser.Result) *alfred.Item {
	uid := "gh:" + parsed.Repo()
	title := "Open " + parsed.Repo()
	arg := "open https://github.com/" + parsed.Repo()
	icon := repoIcon

	if parsed.HasIssue() {
		uid += "#" + parsed.Issue()
		title += "#" + parsed.Issue()
		arg += "/issues/" + parsed.Issue()
		icon = issueIcon
	}

	if parsed.HasPath() {
		uid += parsed.Path()
		title += parsed.Path()
		arg += parsed.Path()
		icon = pathIcon
	}

	title += parsed.Annotation()

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

func openIssuesItem(parsed *parser.Result) (item *alfred.Item) {
	return &alfred.Item{
		UID:   "ghi:" + parsed.Repo(),
		Title: "Open issues for " + parsed.Repo() + parsed.Annotation(),
		Arg:   "open https://github.com/" + parsed.Repo() + "/issues",
		Valid: true,
		Icon:  issueListIcon,
	}
}

func searchIssuesItem(parsed *parser.Result, fullInput string) *alfred.Item {
	extra := parsed.Annotation()

	if len(parsed.Query) > 0 {
		escaped := url.PathEscape(parsed.Query)
		arg := "open https://github.com/" + parsed.Repo() + "/search?utf8=✓&type=Issues&q=" + escaped
		return &alfred.Item{
			UID:   "ghis:" + parsed.Repo(),
			Title: "Search issues in " + parsed.Repo() + extra + " for " + parsed.Query,
			Arg:   arg,
			Valid: true,
			Icon:  searchIcon,
		}
	}

	return &alfred.Item{
		Title:        "Search issues in " + parsed.Repo() + extra + " for...",
		Valid:        false,
		Icon:         searchIcon,
		Autocomplete: fullInput + " ",
	}
}

func newIssueItem(parsed *parser.Result) *alfred.Item {
	title := "New issue in " + parsed.Repo()
	title += parsed.Annotation()

	if len(parsed.Query) == 0 {
		return &alfred.Item{
			UID:   "ghn:" + parsed.Repo(),
			Title: title,
			Arg:   "open https://github.com/" + parsed.Repo() + "/issues/new",
			Valid: true,
			Icon:  newIssueIcon,
		}
	}

	escaped := url.PathEscape(parsed.Query)
	arg := "open https://github.com/" + parsed.Repo() + "/issues/new?title=" + escaped
	return &alfred.Item{
		UID:   "ghn:" + parsed.Repo(),
		Title: title + ": " + parsed.Query,
		Arg:   arg,
		Valid: true,
		Icon:  newIssueIcon,
	}
}

func commitSearchItem(parsed *parser.Result, validQuery bool) *alfred.Item {
	title := "Find commit in " + parsed.Repo()
	title += parsed.RepoAnnotation()

	if validQuery {
		escaped := url.PathEscape(parsed.Query)
		arg := "open https://github.com/" + parsed.Repo() + "/search?utf8=✓&type=Issues&q=" + escaped
		return &alfred.Item{
			UID:   "ghc:" + parsed.Repo(),
			Title: title + " with SHA1 " + parsed.Query,
			Arg:   arg,
			Valid: true,
			Icon:  commitIcon,
		}
	}

	space := ""
	if len(parsed.Query) > 0 {
		space = " "
	}

	return &alfred.Item{
		Title: title + " with SHA1" + space + parsed.Query + "...",
		Valid: false,
		Icon:  commitIcon,
	}

}

func markdownLinkItem(parsed *parser.Result) *alfred.Item {
	uid := "ghm:" + parsed.Repo()
	title := "Insert Markdown link to " + parsed.Repo()
	desc := parsed.Repo()
	link := "https://github.com/" + parsed.Repo()
	icon := markdownIcon

	if parsed.HasIssue() {
		uid += "#" + parsed.Issue()
		title += "#" + parsed.Issue()
		desc += "#" + parsed.Issue()
		link += "/issues/" + parsed.Issue()
		icon = issueIcon
	}

	title += parsed.Annotation()
	markdown := fmt.Sprintf("[%s](%s)", desc, link)

	return &alfred.Item{
		UID:   uid,
		Title: title,
		Arg:   "paste " + markdown,
		Text:  &alfred.Text{Copy: markdown, LargeType: markdown},
		Valid: true,
		Icon:  icon,
	}
}

func issueReferenceItem(parsed *parser.Result) *alfred.Item {
	title := "Insert issue reference to " + parsed.Repo()
	ref := parsed.Repo()

	if parsed.HasIssue() {
		title += "#" + parsed.Issue()
		ref += "#" + parsed.Issue()
	} else {
		title += "#..."
	}

	title += parsed.Annotation()

	if parsed.HasIssue() {

		return &alfred.Item{
			UID:   "ghr:" + ref,
			Title: title,
			Arg:   "paste " + ref,
			Valid: true,
			Icon:  issueIcon,
			Text:  &alfred.Text{Copy: ref, LargeType: ref},
		}

	}

	auto := "r " + parsed.Repo()
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

func autocompleteCommitSearchItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Find commit in %s (%s) with SHA1...", repo, key),
		Valid:        false,
		Autocomplete: "c " + key + " ",
		Icon:         commitIcon,
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
		Icon:         repoIcon,
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

func openEndedCommitSearchItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Find commit in %s...", input),
		Autocomplete: "c " + input,
		Valid:        false,
		Icon:         commitIcon,
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
	if strings.Contains(input, " ") {
		return
	}

	if len(input) > 0 {
		for key, repo := range cfg.RepoMap {
			if strings.HasPrefix(key, input) && key != parsed.Match && repo != parsed.Repo() {
				items = append(items, autocompleteItem(key, repo))
			}
		}
	}

	if len(input) == 0 || parsed.Repo() != input {
		items = append(items, openEndedItem(input))
	}
	return
}

func findProjectDirs(root string) (dirs []string) {
	if entries, err := ioutil.ReadDir(root); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, entry.Name())
			} else if entry.Mode()&os.ModeSymlink != 0 {
				full := path.Join(root, entry.Name())
				if link, err := os.Readlink(full); err != nil {
					continue
				} else {
					if !path.IsAbs(link) {
						if link, err = filepath.Abs(path.Join(root, link)); err != nil {
							continue
						}
					}
					if linkInfo, err := os.Stat(link); err != nil {
						continue
					} else {
						if linkInfo.IsDir() {
							dirs = append(dirs, entry.Name())
						}
					}
				}
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

// Issue the given query string to the RPC backend.
// If RPC is not configured, the results will be empty.
func rpcRequest(query string, cfg *config.Config) (shouldRetry bool, results []string, err error) {
	if len(cfg.SocketPath) == 0 {
		return false, results, nil // RPC isn't enabled, don't worry about it
	}
	sock, err := net.Dial("unix", cfg.SocketPath)
	if err != nil {
		return false, results, err
	}
	defer sock.Close()
	if err := sock.SetDeadline(time.Now().Add(socketTimeout)); err != nil {
		return false, results, err
	}
	// write query to socket:
	if _, err := sock.Write([]byte(query + "\n")); err != nil {
		return false, results, err
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
				return false, results, errors.Wrap(err, "Could not read RPC response")
			}
			return false, results, errors.Wrap(err, "Unexpected RPC response status")
		}
	} else {
		if err := scanner.Err(); err != nil {
			return false, results, errors.Wrap(err, "Could not read RPC response")
		}
		return false, results, errors.Wrap(err, "No response from RPC backend")
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
		retry, results, err := rpcRequest("repo:"+parsed.Repo(), cfg)
		shouldRetry = retry
		if err != nil {
			item.Subtitle = err.Error()
		} else if shouldRetry {
			item.Subtitle = ellipsis("Retrieving description", duration)
		} else if len(results) > 0 {
			item.Subtitle = results[0]
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

	retry, results, err := rpcRequest("issue:"+parsed.Repo()+"#"+parsed.Issue(), cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving issue title", duration)
	} else if len(results) > 0 {
		parts := strings.SplitN(results[0], ":", 3)
		if len(parts) != 3 {
			return
		}
		kind, state, title := parts[0], parts[1], parts[2]
		item.Subtitle = item.Title
		item.Title = title
		item.Icon = issueStateIcon(kind, state)
	}

	return
}

func retrieveIssueSearchItems(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg *config.Config) (shouldRetry bool, matches alfred.Items) {
	if duration.Seconds() < searchDelay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("issuesearch:repo:"+parsed.Repo()+" "+parsed.Query, cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Searching issues", duration)
	} else if len(results) > 0 {
		matches = append(matches, issueItemsFromResults(results)...)
	}

	return
}

func retrieveIssueList(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg *config.Config) (shouldRetry bool, matches alfred.Items) {
	if duration.Seconds() < issueListDelay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("issuesearch:repo:"+parsed.Repo()+" sort:updated-desc", cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving recent issues", duration)
	} else if len(results) > 0 {
		matches = append(matches, issueItemsFromResults(results)...)
	}

	return
}

func issueItemsFromResults(results []string) (matches alfred.Items) {
	for _, result := range results {
		parts := strings.SplitN(result, ":", 5)
		if len(parts) != 5 {
			continue
		}
		repo, number, kind, state, title := parts[0], parts[1], parts[2], parts[3], parts[4]
		arg := ""
		if kind == "Issue" {
			arg = "open https://github.com/" + repo + "/issues/" + number
		} else {
			arg = "open https://github.com/" + repo + "/pull/" + number
		}

		// no UID so alfred doesn't remember these
		matches = append(matches, &alfred.Item{
			Title:    fmt.Sprintf("#%s %s", number, title),
			Subtitle: fmt.Sprintf("Open %s#%s", repo, number),
			Valid:    true,
			Arg:      arg,
			Icon:     issueStateIcon(kind, state),
		})
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

func issueStateIcon(kind, state string) *alfred.Icon {
	switch kind {
	case "Issue":
		if state == "OPEN" {
			return issueIconOpen
		}
		return issueIconClosed
	case "PullRequest":
		switch state {
		case "OPEN":
			return pullRequestIconOpen
		case "CLOSED":
			return pullRequestIconClosed
		case "MERGED":
			return pullRequestIconMerged
		}
	}
	return issueIcon // sane default
}

func errorItem(context, msg string) *alfred.Item {
	return &alfred.Item{
		Title:    context,
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
