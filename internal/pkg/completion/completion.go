package completion

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sahilm/fuzzy"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/parser"
	"github.com/zerowidth/gh-shorthand/pkg/alfred"
)

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

// Complete runs the main completion code
func Complete(cfg config.Config, env Environment) *alfred.FilterResult {
	result := alfred.NewFilterResult()
	appendParsedItems(result, cfg, env)
	finalizeResult(result)
	return result
}

// Environment represents the runtime environment from Alfred's invocation of
// this binary.
type Environment struct {
	Query string
	Start time.Time
}

// AlfredEnvironment extracts the runtime environment from the OS environment
func AlfredEnvironment(input string) Environment {
	e := Environment{
		Query: input,
		Start: time.Now(),
	}

	if query, ok := os.LookupEnv("query"); ok && query == input {
		if sStr, ok := os.LookupEnv("s"); ok {
			if nsStr, ok := os.LookupEnv("ns"); ok {
				if s, err := strconv.ParseInt(sStr, 10, 64); err == nil {
					if ns, err := strconv.ParseInt(nsStr, 10, 64); err == nil {
						e.Start = time.Unix(s, ns)
					}
				}
			}
		}
	}

	return e
}

func appendParsedItems(result *alfred.FilterResult, cfg config.Config, env Environment) {
	fullInput := env.Query
	input := env.Query

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

	bareUser := mode == "p"
	ignoreNumeric := len(cfg.DefaultRepo) > 0
	parsed := parser.Parse(cfg.RepoMap, cfg.UserMap, input, bareUser, ignoreNumeric)

	// for RPC calls on idle query input:
	shouldRetry := false
	duration := time.Since(env.Start)

	if !parsed.HasRepo() && len(cfg.DefaultRepo) > 0 && !parsed.HasOwner() && !parsed.HasPath() {
		if err := parsed.SetRepo(cfg.DefaultRepo); err != nil {
			result.AppendItems(ErrorItem("Could not set default repo", err.Error()))
		}
	}

	switch mode {
	case "x": // test mode for new RPC
		item := &alfred.Item{
			Title: fmt.Sprintf("x query test: %#v", input),
			Valid: false,
		}

		shouldRetry = annotateQuery(input, item, duration, cfg)
		result.AppendItems(item)

	case "": // no input, show default items
		result.AppendItems(
			repoDefaultItem,
			issueListDefaultItem,
			projectListDefaultItem,
			newIssueDefaultItem,
			issueSearchDefaultItem,
			openProjectDefaultItem,
		)

	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if parsed.HasRepo() &&
			(parsed.HasIssue() || parsed.HasPath() || parsed.EmptyQuery()) {
			item := openRepoItem(parsed)
			if parsed.HasIssue() {
				shouldRetry = retrieveIssueTitle(item, duration, parsed, cfg)
			} else {
				shouldRetry = retrieveRepoDescription(item, duration, parsed, cfg)
			}
			result.AppendItems(item)
		}

		if !parsed.HasRepo() && !parsed.HasOwner() && parsed.HasPath() {
			result.AppendItems(openPathItem(parsed.Path()))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteOpenItem, autocompleteUserOpenItem, openEndedOpenItem)...)
	case "i":
		// repo required
		if parsed.HasRepo() {
			if parsed.EmptyQuery() {
				issuesItem := openIssuesItem(parsed)
				retry, matches := retrieveIssueList(issuesItem, duration, parsed, cfg)
				shouldRetry = retry
				result.AppendItems(issuesItem)
				result.AppendItems(searchIssuesItem(parsed, fullInput))
				result.AppendItems(matches...)
			} else {
				searchItem := searchIssuesItem(parsed, fullInput)
				retry, matches := retrieveIssueSearchItems(searchItem, duration, parsed.Repo(), parsed.Query, cfg, false)
				shouldRetry = retry
				result.AppendItems(searchItem)
				result.AppendItems(matches...)
			}
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteIssueItem, autocompleteUserIssueItem, openEndedIssueItem)...)
	case "p":
		if parsed.HasOwner() && (parsed.HasIssue() || parsed.EmptyQuery()) {
			if parsed.HasRepo() {
				item := repoProjectsItem(parsed)
				if parsed.HasIssue() {
					shouldRetry = retrieveRepoProjectName(item, duration, parsed, cfg)
					result.AppendItems(item)
				} else {
					retry, projects := retrieveRepoProjects(item, duration, parsed, cfg)
					shouldRetry = retry
					result.AppendItems(item)
					result.AppendItems(projects...)
				}
			} else {
				item := orgProjectsItem(parsed)
				if parsed.HasIssue() {
					shouldRetry = retrieveOrgProjectName(item, duration, parsed, cfg)
					result.AppendItems(item)
				} else {
					retry, projects := retrieveOrgProjects(item, duration, parsed, cfg)
					shouldRetry = retry
					result.AppendItems(item)
					result.AppendItems(projects...)
				}
			}
		}

		if !strings.Contains(input, " ") {
			result.AppendItems(
				autocompleteRepoItems(cfg, input, autocompleteProjectItem)...)
			result.AppendItems(
				autocompleteUserItems(cfg, input, parsed, false, autocompleteOrgProjectItem)...)
			if len(input) == 0 || parsed.Repo() != input {
				result.AppendItems(openEndedProjectItem(input))
			}
		}
	case "n":
		// repo required
		if parsed.HasRepo() {
			result.AppendItems(newIssueItem(parsed))
		}

		result.AppendItems(
			autocompleteItems(cfg, input, parsed,
				autocompleteNewIssueItem, autocompleteUserNewIssueItem, openEndedNewIssueItem)...)
	case "e":
		result.AppendItems(
			projectItems(cfg.ProjectDirMap(), input, editorIcon)...)
	case "s":
		searchItem := globalIssueSearchItem(input)
		retry, matches := retrieveIssueSearchItems(searchItem, duration, "", input, cfg, true)
		shouldRetry = retry
		result.AppendItems(searchItem)
		result.AppendItems(matches...)
	}

	// if any RPC-decorated items require a re-invocation of the script, save that
	// information in the environment for the next time
	if shouldRetry {
		result.SetVariable("query", fullInput)
		result.SetVariable("s", fmt.Sprintf("%d", env.Start.Unix()))
		result.SetVariable("ns", fmt.Sprintf("%d", env.Start.Nanosecond()))
	}

	// automatically copy "open <url>" urls to copy/large text
	for _, item := range result.Items {
		if item.Text == nil && len(item.Arg) > 5 && strings.HasPrefix(item.Arg, "open ") {
			url := item.Arg[5:]
			item.Text = &alfred.Text{Copy: url, LargeType: url}
		}
	}
}

func projectItems(dirs map[string]string, search string, icon *alfred.Icon) (items alfred.Items) {
	projects := map[string]string{}
	projectNames := []string{}

	for base, expanded := range dirs {
		if dirs, err := findProjectDirs(expanded); err == nil {
			for _, dirname := range dirs {
				short := filepath.Join(base, dirname)
				full := filepath.Join(expanded, dirname)
				projects[short] = full
				projectNames = append(projectNames, short)
			}
		} else {
			items = append(items, ErrorItem("Invalid project directory: "+base, err.Error()))
		}
	}

	if len(search) > 0 {
		sorted := fuzzy.Find(search, projectNames)
		projectNames = []string{}
		for _, result := range sorted {
			projectNames = append(projectNames, result.Str)
		}
	}

	for _, short := range projectNames {
		items = append(items, &alfred.Item{
			UID:      "ghe:" + short,
			Title:    short,
			Subtitle: "Edit " + short,
			Arg:      "edit " + projects[short],
			Text:     &alfred.Text{Copy: projects[short], LargeType: projects[short]},
			Valid:    true,
			Icon:     icon,
			Mods: &alfred.Mods{
				Cmd: &alfred.ModItem{
					Valid:    true,
					Arg:      "term " + projects[short],
					Subtitle: "Open terminal in " + short,
					Icon:     terminalIcon,
				},
				Alt: &alfred.ModItem{
					Valid:    true,
					Arg:      "finder " + projects[short],
					Subtitle: "Open finder in " + short,
					Icon:     finderIcon,
				},
			},
		})
	}

	return
}

func openRepoItem(parsed *parser.Result) *alfred.Item {
	uid := "gh:" + parsed.Repo()
	title := "Open " + parsed.Repo()
	arg := "open https://github.com/" + parsed.Repo()
	icon := repoIcon
	var mods *alfred.Mods

	if parsed.HasIssue() {
		uid += "#" + parsed.Issue()
		title += "#" + parsed.Issue()
		arg += "/issues/" + parsed.Issue()
		icon = issueIcon
		mods = issueMods(parsed.Repo(), parsed.Issue())
	}

	if parsed.HasPath() {
		uid += parsed.Path()
		title += parsed.Path()
		arg += parsed.Path()
		icon = pathIcon
	}

	if !parsed.HasIssue() && !parsed.HasPath() {
		mods = repoMods(parsed.Repo())
	}

	title += parsed.Annotation()

	return &alfred.Item{
		UID:   uid,
		Title: title,
		Arg:   arg,
		Valid: true,
		Icon:  icon,
		Mods:  mods,
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
		Title: "List issues for " + parsed.Repo() + parsed.Annotation(),
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

func repoProjectsItem(parsed *parser.Result) *alfred.Item {
	if parsed.HasIssue() {
		return &alfred.Item{
			UID:   "ghp:" + parsed.Repo() + "/" + parsed.Issue(),
			Title: "Open project #" + parsed.Issue() + " in " + parsed.Repo() + parsed.Annotation(),
			Valid: true,
			Arg:   "open https://github.com/" + parsed.Repo() + "/projects/" + parsed.Issue(),
			Icon:  projectIcon,
		}
	}
	return &alfred.Item{
		UID:   "ghp:" + parsed.Repo(),
		Title: "List projects in " + parsed.Repo() + parsed.Annotation(),
		Valid: true,
		Arg:   "open https://github.com/" + parsed.Repo() + "/projects",
		Icon:  projectIcon,
	}
}

func orgProjectsItem(parsed *parser.Result) *alfred.Item {
	if parsed.HasIssue() {
		return &alfred.Item{
			UID:   "ghp:" + parsed.Owner + "/" + parsed.Issue(),
			Title: "Open project #" + parsed.Issue() + " for " + parsed.Owner + parsed.Annotation(),
			Valid: true,
			Arg:   "open https://github.com/orgs/" + parsed.Owner + "/projects/" + parsed.Issue(),
			Icon:  projectIcon,
		}
	}
	return &alfred.Item{
		UID:   "ghp:" + parsed.Owner,
		Title: "List projects for " + parsed.Owner + parsed.Annotation(),
		Valid: true,
		Arg:   "open https://github.com/orgs/" + parsed.Owner + "/projects",
		Icon:  projectIcon,
	}
}

func newIssueItem(parsed *parser.Result) *alfred.Item {
	title := "New issue in " + parsed.Repo()
	title += parsed.Annotation()

	if parsed.EmptyQuery() {
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

func globalIssueSearchItem(input string) *alfred.Item {
	if len(input) > 0 {
		escaped := url.PathEscape(input)
		arg := "open https://github.com/search?utf8=✓&type=Issues&q=" + escaped
		return &alfred.Item{
			UID:   "ghs:",
			Title: "Search issues for " + input,
			Arg:   arg,
			Valid: true,
			Icon:  searchIcon,
		}
	}

	return &alfred.Item{
		Title:        "Search issues for...",
		Valid:        false,
		Icon:         searchIcon,
		Autocomplete: "s ",
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

func autocompleteUserOpenItem(key, user string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("Open %s/... (%s)", user, key),
		Autocomplete: " " + key + "/",
		Icon:         repoIcon,
	}
}

func autocompleteIssueItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghi:" + repo,
		Title:        fmt.Sprintf("List issues for %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/issues",
		Valid:        true,
		Autocomplete: "i " + key,
		Icon:         issueListIcon,
	}
}

func autocompleteUserIssueItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("List issues for %s/... (%s)", repo, key),
		Autocomplete: "i " + key + "/",
		Icon:         issueListIcon,
	}
}

func autocompleteProjectItem(key, repo string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghp:" + repo,
		Title:        fmt.Sprintf("List projects in %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/projects",
		Valid:        true,
		Autocomplete: "p " + key,
		Icon:         projectIcon,
	}
}

func autocompleteOrgProjectItem(key, user string) *alfred.Item {
	return &alfred.Item{
		UID:          "ghp:" + user,
		Title:        fmt.Sprintf("List projects for %s (%s)", user, key),
		Arg:          "open https://github.com/orgs/" + user + "/projects",
		Valid:        true,
		Autocomplete: "p " + key,
		Icon:         projectIcon,
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

func autocompleteUserNewIssueItem(key, user string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("New issue in %s/... (%s)", user, key),
		Autocomplete: "n " + key + "/",
		Icon:         newIssueIcon,
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
		Title:        fmt.Sprintf("List issues for %s...", input),
		Autocomplete: "i " + input,
		Valid:        false,
		Icon:         issueListIcon,
	}
}

func openEndedProjectItem(input string) *alfred.Item {
	return &alfred.Item{
		Title:        fmt.Sprintf("List projects for %s...", input),
		Autocomplete: "p " + input,
		Valid:        false,
		Icon:         projectIcon,
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

func autocompleteItems(cfg config.Config, input string, parsed *parser.Result,
	autocompleteRepoItem func(string, string) *alfred.Item,
	autocompleteUserItem func(string, string) *alfred.Item,
	openEndedItem func(string) *alfred.Item) (items alfred.Items) {

	if strings.Contains(input, " ") {
		return
	}

	items = append(items,
		autocompleteRepoItems(cfg, input, autocompleteRepoItem)...)
	items = append(items,
		autocompleteUserItems(cfg, input, parsed, true, autocompleteUserItem)...)

	if len(input) == 0 || parsed.Repo() != input {
		items = append(items, openEndedItem(input))
	}
	return
}

func autocompleteRepoItems(cfg config.Config, input string,
	autocompleteRepoItem func(string, string) *alfred.Item) (items alfred.Items) {
	if len(input) > 0 {
		for key, repo := range cfg.RepoMap {
			if strings.HasPrefix(key, input) && len(key) > len(input) {
				items = append(items, autocompleteRepoItem(key, repo))
			}
		}
	}
	return
}

func autocompleteUserItems(cfg config.Config, input string,
	parsed *parser.Result, includeMatchedUser bool,
	autocompleteUserItem func(string, string) *alfred.Item) (items alfred.Items) {
	if len(input) > 0 {
		for key, user := range cfg.UserMap {
			prefixed := strings.HasPrefix(key, input) && len(key) > len(input)
			matched := includeMatchedUser && key == parsed.UserMatch && !parsed.HasRepo()
			if prefixed || matched {
				items = append(items, autocompleteUserItem(key, user))
			}
		}
	}
	return
}

func findProjectDirs(root string) (dirs []string, err error) {
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
	} else {
		return dirs, err
	}
	return dirs, nil
}

// Issue the given query string to the RPC backend.
// If RPC is not configured, the results will be empty.
func rpcRequest(query string, cfg config.Config) (shouldRetry bool, results []string, err error) {
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
func retrieveRepoDescription(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("repo:"+parsed.Repo(), cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving description", duration)
	} else if len(results) > 0 {
		item.Subtitle = results[0]
	}

	return
}

func annotateQuery(query string, item *alfred.Item, duration time.Duration, cfg config.Config) bool {
	if len(query) == 0 {
		return false
	}

	if duration.Seconds() < delay {
		return true
	}

	if len(cfg.SocketPath) == 0 {
		return false // RPC isn't enabled, don't worry about it
	}

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "unix", cfg.SocketPath)
			},
		},
		Timeout: socketTimeout,
	}

	u, err := url.Parse("http://gh-shorthand/")
	if err != nil {
		item.Subtitle = err.Error()
		return false
	}
	v := url.Values{}
	v.Set("q", query)
	u.RawQuery = v.Encode()

	resp, err := c.Get(u.String())
	if err != nil {
		item.Subtitle = err.Error()
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	if resp.StatusCode == 204 {
		item.Subtitle = ellipsis("RPC query", duration)
		return true
	} else if resp.StatusCode == 200 {
		item.Subtitle = fmt.Sprintf("rpc response: %s", body)
	} else {
		item.Subtitle = fmt.Sprintf("rpc error: %s", body)
	}

	return false
}

// retrieveIssueTitle adds the title to the "open issue" item using an RPC call
func retrieveIssueTitle(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool) {
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

func retrieveRepoProjectName(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("repo_project:"+parsed.Repo()+"/"+parsed.Issue(), cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving project name", duration)
	} else if len(results) > 0 {
		parts := strings.SplitN(results[0], ":", 2)
		if len(parts) != 2 {
			return
		}
		state, name := parts[0], parts[1]
		item.Subtitle = item.Title
		item.Title = name
		item.Icon = projectStateIcon(state)
	}

	return
}

func retrieveOrgProjectName(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("org_project:"+parsed.Owner+"/"+parsed.Issue(), cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving project name", duration)
	} else if len(results) > 0 {
		parts := strings.SplitN(results[0], ":", 2)
		if len(parts) != 2 {
			return
		}
		state, name := parts[0], parts[1]
		item.Subtitle = item.Title
		item.Title = name
		item.Icon = projectStateIcon(state)
	}

	return
}

func retrieveOrgProjects(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool, projects alfred.Items) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("org_projects:"+parsed.Owner, cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving projects", duration)
	} else if len(results) > 0 {
		projects = append(projects, projectItemsFromResults(results, "for "+parsed.Owner)...)
	}
	return
}

func retrieveRepoProjects(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool, projects alfred.Items) {
	if duration.Seconds() < delay {
		shouldRetry = true
		return
	}

	retry, results, err := rpcRequest("repo_projects:"+parsed.Repo(), cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Retrieving projects", duration)
	} else if len(results) > 0 {
		projects = append(projects, projectItemsFromResults(results, "in "+parsed.Repo())...)
	}
	return
}

func projectItemsFromResults(results []string, desc string) (items alfred.Items) {
	for _, result := range results {
		parts := strings.SplitN(result, "#", 4)
		if len(parts) != 4 {
			continue
		}
		number, state, url, name := parts[0], parts[1], parts[2], parts[3]

		// no UID so alfred doesn't remember these
		items = append(items, &alfred.Item{
			Title:    name,
			Subtitle: fmt.Sprintf("Open project #%s %s", number, desc),
			Valid:    true,
			Arg:      "open " + url,
			Icon:     projectStateIcon(state),
		})
	}
	return
}

func retrieveIssueSearchItems(item *alfred.Item, duration time.Duration, repo, query string, cfg config.Config, includeRepo bool) (shouldRetry bool, matches alfred.Items) {
	if !item.Valid {
		return
	}
	if duration.Seconds() < searchDelay {
		shouldRetry = true
		return
	}

	rpcQuery := "issuesearch:"
	if len(repo) > 0 {
		rpcQuery += "repo:" + repo + " "
	}
	rpcQuery += query
	retry, results, err := rpcRequest(rpcQuery, cfg)
	shouldRetry = retry
	if err != nil {
		item.Subtitle = err.Error()
	} else if shouldRetry {
		item.Subtitle = ellipsis("Searching issues", duration)
	} else if len(results) > 0 {
		matches = append(matches, issueItemsFromResults(results, includeRepo)...)
	}

	return
}

func retrieveIssueList(item *alfred.Item, duration time.Duration, parsed *parser.Result, cfg config.Config) (shouldRetry bool, matches alfred.Items) {
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
		matches = append(matches, issueItemsFromResults(results, false)...)
	}

	return
}

func issueItemsFromResults(results []string, includeRepo bool) (matches alfred.Items) {
	for _, result := range results {
		parts := strings.SplitN(result, ":", 5)
		if len(parts) != 5 {
			continue
		}
		repo, number, kind, state, title := parts[0], parts[1], parts[2], parts[3], parts[4]
		itemTitle := fmt.Sprintf("#%s %s", number, title)
		if includeRepo {
			itemTitle = repo + itemTitle
		}
		arg := ""
		if kind == "Issue" {
			arg = "open https://github.com/" + repo + "/issues/" + number
		} else {
			arg = "open https://github.com/" + repo + "/pull/" + number
		}

		// no UID so alfred doesn't remember these
		matches = append(matches, &alfred.Item{
			Title:    itemTitle,
			Subtitle: fmt.Sprintf("Open %s#%s", repo, number),
			Valid:    true,
			Arg:      arg,
			Icon:     issueStateIcon(kind, state),
			Mods:     issueMods(repo, number),
		})
	}
	return
}

func repoMods(repo string) *alfred.Mods {
	return &alfred.Mods{
		Cmd: &alfred.ModItem{
			Valid:    true,
			Arg:      fmt.Sprintf("paste [%s](https://github.com/%s)", repo, repo),
			Subtitle: fmt.Sprintf("Insert Markdown link to %s", repo),
			Icon:     markdownIcon,
		},
	}
}

func issueMods(repo, number string) *alfred.Mods {
	return &alfred.Mods{
		Cmd: &alfred.ModItem{
			Valid:    true,
			Arg:      fmt.Sprintf("paste [%s#%s](https://github.com/%s/issues/%s)", repo, number, repo, number),
			Subtitle: fmt.Sprintf("Insert Markdown link to %s#%s", repo, number),
			Icon:     markdownIcon,
		},
		Alt: &alfred.ModItem{
			Valid:    true,
			Arg:      fmt.Sprintf("paste %s#%s", repo, number),
			Subtitle: fmt.Sprintf("Insert issue reference to %s#%s", repo, number),
			Icon:     issueIcon,
		},
	}
}

// ErrorItem returns an error message entry to display in alfred
func ErrorItem(title, subtitle string) *alfred.Item {
	return &alfred.Item{
		Title:    title,
		Subtitle: subtitle,
		Icon:     octicon("alert"),
		Valid:    false,
	}
}

func finalizeResult(result *alfred.FilterResult) {
	if result.Variables != nil && len(*result.Variables) > 0 {
		result.Rerun = rerunAfter
	}
}
