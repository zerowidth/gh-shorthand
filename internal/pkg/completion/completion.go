package completion

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
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
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
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

type projectMode int

const (
	modeEdit projectMode = iota
	modeTerm
)

// Used internally to collect the input and output for completion
type completion struct {
	cfg    config.Config
	env    Environment
	result alfred.FilterResult
	mode   string
	input  string
	parsed parser.Result
	retry  bool // for RPC calls on idle query input
}

// Complete runs the main completion code
func Complete(cfg config.Config, env Environment) alfred.FilterResult {
	mode, input, ok := extractMode(env.Query)
	if !ok {
		return alfred.NewFilterResult()
	}

	bareUser := mode == "p"
	ignoreNumeric := len(cfg.DefaultRepo) > 0
	parsed := parser.Parse(cfg.RepoMap, cfg.UserMap, input, bareUser, ignoreNumeric)

	c := completion{
		cfg:    cfg,
		env:    env,
		result: alfred.NewFilterResult(),
		mode:   mode,
		input:  input,
		parsed: parsed,
	}
	c.appendParsedItems()
	c.finalizeResult()

	return c.result
}

// Environment represents the runtime environment from Alfred's invocation of
// this binary.
type Environment struct {
	Query string
	Start time.Time
}

// LoadAlfredEnvironment extracts the runtime environment from the OS environment
//
// The result of a script filter can set environment variables along with a
// "rerun this" timer for another invocation, and this retrieves and stores that
// information.
func LoadAlfredEnvironment(input string) Environment {
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

// Duration since alfred saw the first query
func (e Environment) Duration() time.Duration {
	return time.Since(e.Start)
}

// given an input query, extract the mode and input string. returns false if
// mode+input is invalid.
//
// mode is an optional single character, followed by a space.
func extractMode(input string) (string, string, bool) {
	var mode string
	if len(input) == 1 {
		mode = input[0:1]
		input = ""
	} else if len(input) > 1 {
		mode = input[0:1]
		if mode == " " {
			input = input[1:]
		} else {
			// not a mode followed by space, it's invalid
			if input[1:2] != " " {
				return "", "", false
			}
			input = input[2:]
		}
	}
	return mode, input, true
}

func (c *completion) appendParsedItems() {
	fullInput := c.env.Query

	if !c.parsed.HasRepo() && len(c.cfg.DefaultRepo) > 0 && !c.parsed.HasOwner() && !c.parsed.HasPath() {
		c.parsed.SetRepo(c.cfg.DefaultRepo)
	}

	switch c.mode {
	case "": // no input, show default items
		c.result.AppendItems(
			repoDefaultItem,
			issueListDefaultItem,
			projectListDefaultItem,
			newIssueDefaultItem,
			issueSearchDefaultItem,
			openProjectDefaultItem,
		)

	case " ": // open repo, issue, and/or path
		// repo required, no query allowed
		if c.parsed.HasRepo() &&
			(c.parsed.HasIssue() || c.parsed.HasPath() || c.parsed.EmptyQuery()) {
			item := openRepoItem(c.parsed)
			if c.parsed.HasIssue() {
				c.retrieveIssueTitle(&item)
			} else {
				c.retrieveRepoDescription(&item)
			}
			c.result.AppendItems(item)
		}

		if !c.parsed.HasRepo() && !c.parsed.HasOwner() && c.parsed.HasPath() {
			c.result.AppendItems(openPathItem(c.parsed.Path()))
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, c.parsed,
				autocompleteOpenItem, autocompleteUserOpenItem, openEndedOpenItem)...)
	case "i":
		// repo required
		if c.parsed.HasRepo() {
			if c.parsed.EmptyQuery() {
				issuesItem := openIssuesItem(c.parsed)
				matches := c.retrieveIssueList(&issuesItem)
				c.result.AppendItems(issuesItem)
				c.result.AppendItems(searchIssuesItem(c.parsed, fullInput))
				c.result.AppendItems(matches...)
			} else {
				searchItem := searchIssuesItem(c.parsed, fullInput)
				matches := c.retrieveIssueSearchItems(&searchItem, c.parsed.Repo(), c.parsed.Query, false)
				c.result.AppendItems(searchItem)
				c.result.AppendItems(matches...)
			}
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, c.parsed,
				autocompleteIssueItem, autocompleteUserIssueItem, openEndedIssueItem)...)
	case "p":
		if c.parsed.HasOwner() && (c.parsed.HasIssue() || c.parsed.EmptyQuery()) {
			if c.parsed.HasRepo() {
				item := repoProjectsItem(c.parsed)
				if c.parsed.HasIssue() {
					c.retrieveRepoProjectName(&item)
					c.result.AppendItems(item)
				} else {
					projects := c.retrieveRepoProjects(&item)
					c.result.AppendItems(item)
					c.result.AppendItems(projects...)
				}
			} else {
				item := orgProjectsItem(c.parsed)
				if c.parsed.HasIssue() {
					c.retrieveOrgProjectName(&item)
					c.result.AppendItems(item)
				} else {
					projects := c.retrieveOrgProjects(&item)
					c.result.AppendItems(item)
					c.result.AppendItems(projects...)
				}
			}
		}

		if !strings.Contains(c.input, " ") {
			c.result.AppendItems(
				autocompleteRepoItems(c.cfg, c.input, autocompleteProjectItem)...)
			c.result.AppendItems(
				autocompleteUserItems(c.cfg, c.input, c.parsed, false, autocompleteOrgProjectItem)...)
			if len(c.input) == 0 || c.parsed.Repo() != c.input {
				c.result.AppendItems(openEndedProjectItem(c.input))
			}
		}
	case "n":
		// repo required
		if c.parsed.HasRepo() {
			c.result.AppendItems(newIssueItem(c.parsed))
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, c.parsed,
				autocompleteNewIssueItem, autocompleteUserNewIssueItem, openEndedNewIssueItem)...)
	case "e":
		c.result.AppendItems(
			projectItems(c.cfg.ProjectDirMap(), c.input, modeEdit)...)
	case "t":
		c.result.AppendItems(
			projectItems(c.cfg.ProjectDirMap(), c.input, modeTerm)...)
	case "s":
		searchItem := globalIssueSearchItem(c.input)
		matches := c.retrieveIssueSearchItems(&searchItem, "", c.input, true)
		c.result.AppendItems(searchItem)
		c.result.AppendItems(matches...)
	}

}

func projectItems(dirs map[string]string, search string, mode projectMode) (items alfred.Items) {
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
		var item = alfred.Item{
			Title: short,
			Valid: true,
			Text:  &alfred.Text{Copy: projects[short], LargeType: projects[short]},
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
		}

		if mode == modeEdit {
			item.UID = "ghe:" + short
			item.Subtitle = "Edit " + short
			item.Icon = editorIcon
			item.Mods.Cmd = &alfred.ModItem{
				Valid:    true,
				Arg:      "term " + projects[short],
				Subtitle: "Open terminal in " + short,
				Icon:     terminalIcon,
			}
		} else {
			item.UID = "ght:" + short
			item.Subtitle = "Open terminal in " + short
			item.Icon = terminalIcon
			item.Mods.Cmd = &alfred.ModItem{
				Valid:    true,
				Arg:      "edit " + projects[short],
				Subtitle: "Edit " + short,
				Icon:     editorIcon,
			}
		}

		items = append(items, item)
	}

	return
}

func openRepoItem(parsed parser.Result) alfred.Item {
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

	return alfred.Item{
		UID:   uid,
		Title: title,
		Arg:   arg,
		Valid: true,
		Icon:  icon,
		Mods:  mods,
	}
}

func openPathItem(path string) alfred.Item {
	return alfred.Item{
		UID:   "gh:" + path,
		Title: fmt.Sprintf("Open %s", path),
		Arg:   "open https://github.com" + path,
		Valid: true,
		Icon:  pathIcon,
	}
}

func openIssuesItem(parsed parser.Result) (item alfred.Item) {
	return alfred.Item{
		UID:   "ghi:" + parsed.Repo(),
		Title: "List issues for " + parsed.Repo() + parsed.Annotation(),
		Arg:   "open https://github.com/" + parsed.Repo() + "/issues",
		Valid: true,
		Icon:  issueListIcon,
	}
}

func searchIssuesItem(parsed parser.Result, fullInput string) alfred.Item {
	extra := parsed.Annotation()

	if len(parsed.Query) > 0 {
		escaped := url.PathEscape(parsed.Query)
		arg := "open https://github.com/" + parsed.Repo() + "/search?utf8=✓&type=Issues&q=" + escaped
		return alfred.Item{
			UID:   "ghis:" + parsed.Repo(),
			Title: "Search issues in " + parsed.Repo() + extra + " for " + parsed.Query,
			Arg:   arg,
			Valid: true,
			Icon:  searchIcon,
		}
	}

	return alfred.Item{
		Title:        "Search issues in " + parsed.Repo() + extra + " for...",
		Valid:        false,
		Icon:         searchIcon,
		Autocomplete: fullInput + " ",
	}
}

func repoProjectsItem(parsed parser.Result) alfred.Item {
	if parsed.HasIssue() {
		return alfred.Item{
			UID:   "ghp:" + parsed.Repo() + "/" + parsed.Issue(),
			Title: "Open project #" + parsed.Issue() + " in " + parsed.Repo() + parsed.Annotation(),
			Valid: true,
			Arg:   "open https://github.com/" + parsed.Repo() + "/projects/" + parsed.Issue(),
			Icon:  projectIcon,
		}
	}
	return alfred.Item{
		UID:   "ghp:" + parsed.Repo(),
		Title: "List projects in " + parsed.Repo() + parsed.Annotation(),
		Valid: true,
		Arg:   "open https://github.com/" + parsed.Repo() + "/projects",
		Icon:  projectIcon,
	}
}

func orgProjectsItem(parsed parser.Result) alfred.Item {
	if parsed.HasIssue() {
		return alfred.Item{
			UID:   "ghp:" + parsed.User + "/" + parsed.Issue(),
			Title: "Open project #" + parsed.Issue() + " for " + parsed.User + parsed.Annotation(),
			Valid: true,
			Arg:   "open https://github.com/orgs/" + parsed.User + "/projects/" + parsed.Issue(),
			Icon:  projectIcon,
		}
	}
	return alfred.Item{
		UID:   "ghp:" + parsed.User,
		Title: "List projects for " + parsed.User + parsed.Annotation(),
		Valid: true,
		Arg:   "open https://github.com/orgs/" + parsed.User + "/projects",
		Icon:  projectIcon,
	}
}

func newIssueItem(parsed parser.Result) alfred.Item {
	title := "New issue in " + parsed.Repo()
	title += parsed.Annotation()

	if parsed.EmptyQuery() {
		return alfred.Item{
			UID:   "ghn:" + parsed.Repo(),
			Title: title,
			Arg:   "open https://github.com/" + parsed.Repo() + "/issues/new",
			Valid: true,
			Icon:  newIssueIcon,
		}
	}

	escaped := url.PathEscape(parsed.Query)
	arg := "open https://github.com/" + parsed.Repo() + "/issues/new?title=" + escaped
	return alfred.Item{
		UID:   "ghn:" + parsed.Repo(),
		Title: title + ": " + parsed.Query,
		Arg:   arg,
		Valid: true,
		Icon:  newIssueIcon,
	}
}

func globalIssueSearchItem(input string) alfred.Item {
	if len(input) > 0 {
		escaped := url.PathEscape(input)
		arg := "open https://github.com/search?utf8=✓&type=Issues&q=" + escaped
		return alfred.Item{
			UID:   "ghs:",
			Title: "Search issues for " + input,
			Arg:   arg,
			Valid: true,
			Icon:  searchIcon,
		}
	}

	return alfred.Item{
		Title:        "Search issues for...",
		Valid:        false,
		Icon:         searchIcon,
		Autocomplete: "s ",
	}
}

func autocompleteOpenItem(key, repo string) alfred.Item {
	return alfred.Item{
		UID:          "gh:" + repo,
		Title:        fmt.Sprintf("Open %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo,
		Valid:        true,
		Autocomplete: " " + key,
		Icon:         repoIcon,
	}
}

func autocompleteUserOpenItem(key, user string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("Open %s/... (%s)", user, key),
		Autocomplete: " " + key + "/",
		Icon:         repoIcon,
	}
}

func autocompleteIssueItem(key, repo string) alfred.Item {
	return alfred.Item{
		UID:          "ghi:" + repo,
		Title:        fmt.Sprintf("List issues for %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/issues",
		Valid:        true,
		Autocomplete: "i " + key,
		Icon:         issueListIcon,
	}
}

func autocompleteUserIssueItem(key, repo string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("List issues for %s/... (%s)", repo, key),
		Autocomplete: "i " + key + "/",
		Icon:         issueListIcon,
	}
}

func autocompleteProjectItem(key, repo string) alfred.Item {
	return alfred.Item{
		UID:          "ghp:" + repo,
		Title:        fmt.Sprintf("List projects in %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/projects",
		Valid:        true,
		Autocomplete: "p " + key,
		Icon:         projectIcon,
	}
}

func autocompleteOrgProjectItem(key, user string) alfred.Item {
	return alfred.Item{
		UID:          "ghp:" + user,
		Title:        fmt.Sprintf("List projects for %s (%s)", user, key),
		Arg:          "open https://github.com/orgs/" + user + "/projects",
		Valid:        true,
		Autocomplete: "p " + key,
		Icon:         projectIcon,
	}
}

func autocompleteNewIssueItem(key, repo string) alfred.Item {
	return alfred.Item{
		UID:          "ghn:" + repo,
		Title:        fmt.Sprintf("New issue in %s (%s)", repo, key),
		Arg:          "open https://github.com/" + repo + "/issues/new",
		Valid:        true,
		Autocomplete: "n " + key,
		Icon:         newIssueIcon,
	}
}

func autocompleteUserNewIssueItem(key, user string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("New issue in %s/... (%s)", user, key),
		Autocomplete: "n " + key + "/",
		Icon:         newIssueIcon,
	}
}

func openEndedOpenItem(input string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("Open %s...", input),
		Autocomplete: " " + input,
		Valid:        false,
		Icon:         repoIcon,
	}
}

func openEndedIssueItem(input string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("List issues for %s...", input),
		Autocomplete: "i " + input,
		Valid:        false,
		Icon:         issueListIcon,
	}
}

func openEndedProjectItem(input string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("List projects for %s...", input),
		Autocomplete: "p " + input,
		Valid:        false,
		Icon:         projectIcon,
	}
}

func openEndedNewIssueItem(input string) alfred.Item {
	return alfred.Item{
		Title:        fmt.Sprintf("New issue in %s...", input),
		Autocomplete: "n " + input,
		Valid:        false,
		Icon:         newIssueIcon,
	}
}

func autocompleteItems(cfg config.Config, input string, parsed parser.Result,
	autocompleteRepoItem func(string, string) alfred.Item,
	autocompleteUserItem func(string, string) alfred.Item,
	openEndedItem func(string) alfred.Item) (items alfred.Items) {

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
	autocompleteRepoItem func(string, string) alfred.Item) (items alfred.Items) {
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
	parsed parser.Result, includeMatchedUser bool,
	autocompleteUserItem func(string, string) alfred.Item) (items alfred.Items) {
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

func (c *completion) rpcRequest(path, query string, delay float64) (rpc.Result, error) {
	var result rpc.Result

	if len(c.cfg.SocketPath) == 0 {
		return result, nil // RPC isn't enabled, don't worry about it
	}
	if c.env.Duration().Seconds() < delay {
		c.retry = true
		return result, nil
	}

	res, err := rpc.Query(c.cfg, path, query)

	if err == nil && !res.Complete {
		c.retry = true
	}

	// wrap result errors as real errors, for simpler handling by the caller
	if len(res.Error) > 0 {
		return res, fmt.Errorf("RPC service error: %s", res.Error)
	}

	return res, err
}

// Issue the given query string to the RPC backend.
//
// If RPC is not configured, the results will be empty.
//
// If the RPC request should be repeated (either not enough time has passed, or
// the remote request is still pending) c.retry is set to true.
func (c *completion) oldRPCRequest(query string, delay float64) (results []string, err error) {
	if len(c.cfg.SocketPath) == 0 {
		return results, nil // RPC isn't enabled, don't worry about it
	}
	if c.env.Duration().Seconds() < delay {
		c.retry = true
		return
	}
	sock, err := net.Dial("unix", c.cfg.SocketPath)
	if err != nil {
		return results, err
	}
	defer sock.Close()
	if err := sock.SetDeadline(time.Now().Add(socketTimeout)); err != nil {
		return results, err
	}

	// write query to socket:
	if _, err := sock.Write([]byte(query + "\n")); err != nil {
		return results, err
	}

	// read results:
	scanner := bufio.NewScanner(sock)
	if scanner.Scan() {
		status := scanner.Text()
		switch status {
		case "OK":
			for scanner.Scan() {
				results = append(results, scanner.Text())
			}
			return results, nil
		case "PENDING":
			c.retry = true
			return results, nil
		case "ERROR":
			for scanner.Scan() {
				results = append(results, scanner.Text())
			}
			if len(results) > 0 {
				err = errors.New(results[0])
			} else {
				err = errors.New("unknown RPC error")
			}
			return results, err
		default:
			if err := scanner.Err(); err != nil {
				return results, errors.Wrap(err, "Could not read RPC response")
			}
			return results, errors.Wrap(err, "Unexpected RPC response status")
		}
	} else {
		if err := scanner.Err(); err != nil {
			return results, errors.Wrap(err, "Could not read RPC response")
		}
		return results, errors.Wrap(err, "No response from RPC backend")
	}
}

func ellipsis(prefix string, duration time.Duration) string {
	return prefix + strings.Repeat(".", int((duration.Nanoseconds()/250000000)%4))
}

// retrieveRepoDescription adds the repo description to the "open repo" item
// using an RPC call.
func (c *completion) retrieveRepoDescription(item *alfred.Item) {
	res, err := c.rpcRequest("/repo", c.parsed.Repo(), delay)
	if err != nil {
		item.Subtitle = err.Error()
		return
	}
	if !res.Complete {
		item.Subtitle = ellipsis("Retrieving description", c.env.Duration())
		return
	}
	if len(res.Repos) == 0 {
		item.Subtitle = "rpc error: missing repo in result"
		return
	}

	item.Subtitle = res.Repos[0].Description
}

// retrieveIssueTitle adds the title and state to an "open issue" item
func (c *completion) retrieveIssueTitle(item *alfred.Item) {
	res, err := c.rpcRequest("/issue", c.parsed.Repo()+"#"+c.parsed.Issue(), delay)
	if err != nil {
		item.Subtitle = err.Error()
		return
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving issue title", c.env.Duration())
		return
	} else if len(res.Issues) == 0 {
		item.Subtitle = "rpc error: missing issue in result"
		return
	}

	issue := res.Issues[0]
	item.Subtitle = item.Title
	item.Title = issue.Title
	item.Icon = issueStateIcon(issue.Type, issue.State)
}

func (c *completion) retrieveRepoProjectName(item *alfred.Item) {
	res, err := c.rpcRequest("/project", c.parsed.Repo()+"/"+c.parsed.Issue(), delay)
	if err != nil {
		item.Subtitle = err.Error()
		return
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving project name", c.env.Duration())
		return
	} else if len(res.Projects) == 0 {
		item.Subtitle = "rpc error: missing project in result"
		return
	}

	project := res.Projects[0]
	item.Subtitle = item.Title
	item.Title = project.Name
	item.Icon = projectStateIcon(project.State)
	return
}

func (c *completion) retrieveOrgProjectName(item *alfred.Item) {
	res, err := c.rpcRequest("/project", c.parsed.User+"/"+c.parsed.Issue(), delay)
	if err != nil {
		item.Subtitle = err.Error()
		return
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving project name", c.env.Duration())
		return
	} else if len(res.Projects) == 0 {
		item.Subtitle = "rpc error: missing project in result"
		return
	}

	project := res.Projects[0]
	item.Subtitle = item.Title
	item.Title = project.Name
	item.Icon = projectStateIcon(project.State)
	return
}

func (c *completion) retrieveOrgProjects(item *alfred.Item) (projects alfred.Items) {
	results, err := c.oldRPCRequest("org_projects:"+c.parsed.User, delay)
	if err != nil {
		item.Subtitle = err.Error()
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving projects", c.env.Duration())
	} else if len(results) > 0 {
		projects = append(projects, projectItemsFromResults(results, "for "+c.parsed.User)...)
	}
	return
}

func (c *completion) retrieveRepoProjects(item *alfred.Item) (projects alfred.Items) {
	results, err := c.oldRPCRequest("repo_projects:"+c.parsed.Repo(), delay)
	if err != nil {
		item.Subtitle = err.Error()
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving projects", c.env.Duration())
	} else if len(results) > 0 {
		projects = append(projects, projectItemsFromResults(results, "in "+c.parsed.Repo())...)
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
		items = append(items, alfred.Item{
			Title:    name,
			Subtitle: fmt.Sprintf("Open project #%s %s", number, desc),
			Valid:    true,
			Arg:      "open " + url,
			Icon:     projectStateIcon(state),
		})
	}
	return
}

func (c *completion) retrieveIssueSearchItems(item *alfred.Item, repo, query string, includeRepo bool) alfred.Items {
	if len(repo) > 0 {
		query += "repo:" + repo + " "
	}
	return c.searchIssues(item, query, includeRepo)
}

func (c *completion) searchIssues(item *alfred.Item, query string, includeRepo bool) alfred.Items {
	var items alfred.Items

	if !item.Valid {
		return items
	}

	res, err := c.rpcRequest("/issues", query, searchDelay)
	if err != nil {
		item.Subtitle = err.Error()
		return items
	} else if c.retry {
		item.Subtitle = ellipsis("Searching issues", c.env.Duration())
		return items
	} else if len(res.Issues) == 0 {
		item.Subtitle = "No issues found"
		return items
	}

	items = append(items, issueItemsFromIssues(res.Issues, includeRepo)...)
	return items
}

func (c *completion) retrieveIssueList(item *alfred.Item) (matches alfred.Items) {
	results, err := c.oldRPCRequest("issuesearch:repo:"+c.parsed.Repo()+" sort:updated-desc", issueListDelay)
	if err != nil {
		item.Subtitle = err.Error()
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving recent issues", c.env.Duration())
	} else if len(results) > 0 {
		matches = append(matches, issueItemsFromResults(results, false)...)
	}
	return
}

func issueItemsFromIssues(issues []rpc.Issue, includeRepo bool) alfred.Items {
	var items alfred.Items

	for _, issue := range issues {
		itemTitle := fmt.Sprintf("#%s %s", issue.Number, issue.Title)
		if includeRepo {
			itemTitle = issue.Repo + itemTitle
		}
		arg := ""
		if issue.Type == "Issue" {
			arg = "open https://github.com/" + issue.Repo + "/issues/" + issue.Number
		} else {
			arg = "open https://github.com/" + issue.Repo + "/pull/" + issue.Number
		}

		// no UID so alfred doesn't remember these
		items = append(items, alfred.Item{
			Title:    itemTitle,
			Subtitle: fmt.Sprintf("Open %s#%s", issue.Repo, issue.Number),
			Valid:    true,
			Arg:      arg,
			Icon:     issueStateIcon(issue.Type, issue.State),
			Mods:     issueMods(issue.Repo, issue.Number),
		})
	}

	return items
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
		matches = append(matches, alfred.Item{
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
func ErrorItem(title, subtitle string) alfred.Item {
	return alfred.Item{
		Title:    title,
		Subtitle: subtitle,
		Icon:     octicon("alert"),
		Valid:    false,
	}
}

func (c *completion) finalizeResult() {
	// automatically set "open <url>" urls to copy/large text
	for i, item := range c.result.Items {
		if item.Text == nil && strings.HasPrefix(item.Arg, "open ") {
			url := item.Arg[5:]
			c.result.Items[i].Text = &alfred.Text{Copy: url, LargeType: url}
		}
	}

	// if any RPC-decorated items require a re-invocation of the script, save that
	// information in the environment for the next time
	if c.retry {
		c.result.SetVariable("query", c.env.Query)
		c.result.SetVariable("s", fmt.Sprintf("%d", c.env.Start.Unix()))
		c.result.SetVariable("ns", fmt.Sprintf("%d", c.env.Start.Nanosecond()))
		c.result.Rerun = rerunAfter
	}
}
