package completion

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/zerowidth/gh-shorthand/pkg/alfred"
	"github.com/zerowidth/gh-shorthand/pkg/config"
	"github.com/zerowidth/gh-shorthand/pkg/parser"
	"github.com/zerowidth/gh-shorthand/pkg/rpc"
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
)

// Used internally to collect the input and output for completion
type completion struct {
	// input
	cfg       config.Config // the gh-shorthand config
	env       Environment   // the runtime environment from alfred
	input     string        // the input string from the user (minus mode)
	rpcClient rpc.Client

	// output
	result alfred.FilterResult // the final assembled result
	retry  bool                // should this script be re-invoked? (for RPC)
}

// Complete runs the main completion code
func Complete(cfg config.Config, env Environment) alfred.FilterResult {
	mode, input, ok := extractMode(env.Query)
	if !ok {
		// this didn't have a valid mode, just skip it.
		return alfred.NewFilterResult()
	}

	c := completion{
		cfg:       cfg,
		env:       env,
		result:    alfred.NewFilterResult(),
		input:     input,
		rpcClient: rpc.NewClient(cfg.SocketPath),
	}
	c.appendParsedItems(mode)
	c.finalizeResult()

	return c.result
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

	// default is "no mode", with empty input
	return mode, input, true
}

func (c *completion) appendParsedItems(mode string) {
	fullInput := c.env.Query

	switch mode {
	case "": // no mode, no input, show default items
		c.result.AppendItems(
			defaultItems...,
		)

	case " ": // open repo, issue, and/or path
		parser := parser.NewRepoParser(c.cfg.RepoMap, c.cfg.UserMap, c.cfg.DefaultRepo)
		result := parser.Parse(c.input)

		if result.HasRepo() {
			item := openRepoItem(result)
			if result.HasIssue() {
				c.retrieveIssue(result.Repo(), result.Issue, &item)
			} else {
				c.retrieveRepo(result.Repo(), &item)
			}
			c.result.AppendItems(item)
		}

		if !result.HasRepo() && result.HasPath() {
			c.result.AppendItems(openPathItem(result.Path))
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, result,
				autocompleteOpenItem, autocompleteUserOpenItem, openEndedOpenItem)...)

	case "i":
		parser := parser.NewIssueParser(c.cfg.RepoMap, c.cfg.UserMap, c.cfg.DefaultRepo)
		result := parser.Parse(c.input)

		// repo required
		if result.HasRepo() {

			if result.HasQuery() {
				searchItem := searchIssuesItem(result, fullInput)
				matches := c.retrieveIssueSearchItems(&searchItem, result.Repo(), result.Query, false)
				c.result.AppendItems(searchItem)
				c.result.AppendItems(matches...)
			} else {
				issuesItem := openIssuesItem(result)
				matches := c.retrieveRecentIssues(result.Repo(), &issuesItem)
				c.result.AppendItems(issuesItem)
				c.result.AppendItems(searchIssuesItem(result, fullInput))
				c.result.AppendItems(matches...)
			}
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, result,
				autocompleteIssueItem, autocompleteUserIssueItem, openEndedIssueItem)...)

	case "p":
		parser := parser.NewProjectParser(c.cfg.RepoMap, c.cfg.UserMap, c.cfg.DefaultRepo)
		result := parser.Parse(c.input)

		if result.HasRepo() {
			item := repoProjectsItem(result)
			if result.HasIssue() {
				c.retrieveRepoProject(result.Repo(), result.Issue, &item)
				c.result.AppendItems(item)
			} else {
				projects := c.retrieveRepoProjects(result.Repo(), &item)
				c.result.AppendItems(item)
				c.result.AppendItems(projects...)
			}
		} else if result.HasUser() {
			item := orgProjectsItem(result)
			if result.HasIssue() {
				c.retrieveOrgProject(result.User, result.Issue, &item)
				c.result.AppendItems(item)
			} else {
				projects := c.retrieveOrgProjects(result.User, &item)
				c.result.AppendItems(item)
				c.result.AppendItems(projects...)
			}
		}

		if !strings.Contains(c.input, " ") {
			c.result.AppendItems(
				autocompleteRepoItems(c.cfg, c.input, autocompleteProjectItem)...)
			c.result.AppendItems(
				autocompleteUserItems(c.cfg, c.input, result, false, autocompleteOrgProjectItem)...)
			if len(c.input) == 0 || result.Repo() != c.input {
				c.result.AppendItems(openEndedProjectItem(c.input))
			}
		}

	case "n":
		parser := parser.NewParser(c.cfg.RepoMap, c.cfg.UserMap, c.cfg.DefaultRepo, parser.RequireRepo, parser.WithQuery)
		result := parser.Parse(c.input)

		// repo required
		if result.HasRepo() {
			c.result.AppendItems(newIssueItem(result))
		}

		c.result.AppendItems(
			autocompleteItems(c.cfg, c.input, result,
				autocompleteNewIssueItem, autocompleteUserNewIssueItem, openEndedNewIssueItem)...)

	case "e":
		c.result.AppendItems(
			projectDirItems(c.cfg.ProjectDirMap(), c.input, modeEdit)...)

	case "t":
		c.result.AppendItems(
			projectDirItems(c.cfg.ProjectDirMap(), c.input, modeTerm)...)

	case "s":
		searchItem := globalIssueSearchItem(c.input)
		matches := c.retrieveIssueSearchItems(&searchItem, "", c.input, true)
		c.result.AppendItems(searchItem)
		c.result.AppendItems(matches...)
	}
}

func openRepoItem(parsed *parser.Result) alfred.Item {
	uid := "gh:" + parsed.Repo()
	title := "Open " + parsed.Repo()
	arg := "https://github.com/" + parsed.Repo()
	icon := repoIcon
	var mods *alfred.Mods

	if parsed.HasIssue() {
		uid += "#" + parsed.Issue
		title += "#" + parsed.Issue
		arg += "/issues/" + parsed.Issue
		icon = issueIcon
		mods = issueMods(parsed.Repo(), parsed.Issue, "")
	}

	if parsed.HasPath() {
		uid += parsed.Path
		title += parsed.Path
		arg += parsed.Path
		icon = pathIcon
	}

	if !parsed.HasIssue() && !parsed.HasPath() {
		mods = repoMods(parsed.Repo())
	}

	title += parsed.Annotation()

	return alfred.Item{
		UID:       uid,
		Title:     title,
		Arg:       arg,
		Valid:     true,
		Icon:      icon,
		Variables: alfred.Variables{"action": "open"},
		Mods:      mods,
	}
}

func openPathItem(path string) alfred.Item {
	return alfred.Item{
		UID:       "gh:" + path,
		Title:     fmt.Sprintf("Open %s", path),
		Arg:       "https://github.com" + path,
		Valid:     true,
		Variables: alfred.Variables{"action": "open"},
		Icon:      pathIcon,
	}
}

func openIssuesItem(parsed *parser.Result) (item alfred.Item) {
	return alfred.Item{
		UID:       "ghi:" + parsed.Repo(),
		Title:     "List issues for " + parsed.Repo() + parsed.Annotation(),
		Arg:       "https://github.com/" + parsed.Repo() + "/issues",
		Valid:     true,
		Variables: alfred.Variables{"action": "open"},
		Icon:      issueListIcon,
	}
}

func searchIssuesItem(parsed *parser.Result, fullInput string) alfred.Item {
	extra := parsed.Annotation()

	if len(parsed.Query) > 0 {
		escaped := url.PathEscape(parsed.Query)
		arg := "https://github.com/" + parsed.Repo() + "/search?utf8=✓&type=Issues&q=" + escaped
		return alfred.Item{
			UID:       "ghis:" + parsed.Repo(),
			Title:     "Search issues in " + parsed.Repo() + extra + " for " + parsed.Query,
			Arg:       arg,
			Valid:     true,
			Variables: alfred.Variables{"action": "open"},
			Icon:      searchIcon,
		}
	}

	return alfred.Item{
		Title:        "Search issues in " + parsed.Repo() + extra + " for...",
		Valid:        false,
		Icon:         searchIcon,
		Autocomplete: fullInput + " ",
	}
}

func repoProjectsItem(parsed *parser.Result) alfred.Item {
	if parsed.HasIssue() {
		return alfred.Item{
			UID:       "ghp:" + parsed.Repo() + "/" + parsed.Issue,
			Title:     "Open project #" + parsed.Issue + " in " + parsed.Repo() + parsed.Annotation(),
			Valid:     true,
			Arg:       "https://github.com/" + parsed.Repo() + "/projects/" + parsed.Issue,
			Variables: alfred.Variables{"action": "open"},
			Icon:      projectIcon,
		}
	}
	return alfred.Item{
		UID:       "ghp:" + parsed.Repo(),
		Title:     "List projects in " + parsed.Repo() + parsed.Annotation(),
		Valid:     true,
		Arg:       "https://github.com/" + parsed.Repo() + "/projects",
		Variables: alfred.Variables{"action": "open"},
		Icon:      projectIcon,
	}
}

func orgProjectsItem(parsed *parser.Result) alfred.Item {
	if parsed.HasIssue() {
		return alfred.Item{
			UID:       "ghp:" + parsed.User + "/" + parsed.Issue,
			Title:     "Open project #" + parsed.Issue + " for " + parsed.User + parsed.Annotation(),
			Valid:     true,
			Arg:       "https://github.com/orgs/" + parsed.User + "/projects/" + parsed.Issue,
			Variables: alfred.Variables{"action": "open"},
			Icon:      projectIcon,
		}
	}
	return alfred.Item{
		UID:       "ghp:" + parsed.User,
		Title:     "List projects for " + parsed.User + parsed.Annotation(),
		Valid:     true,
		Arg:       "https://github.com/orgs/" + parsed.User + "/projects",
		Variables: alfred.Variables{"action": "open"},
		Icon:      projectIcon,
	}
}

func newIssueItem(parsed *parser.Result) alfred.Item {
	title := "New issue in " + parsed.Repo()
	title += parsed.Annotation()

	if !parsed.HasQuery() {
		return alfred.Item{
			UID:       "ghn:" + parsed.Repo(),
			Title:     title,
			Arg:       "https://github.com/" + parsed.Repo() + "/issues/new",
			Variables: alfred.Variables{"action": "open"},
			Valid:     true,
			Icon:      newIssueIcon,
		}
	}

	escaped := url.PathEscape(parsed.Query)
	arg := "https://github.com/" + parsed.Repo() + "/issues/new?title=" + escaped
	return alfred.Item{
		UID:       "ghn:" + parsed.Repo(),
		Title:     title + ": " + parsed.Query,
		Arg:       arg,
		Variables: alfred.Variables{"action": "open"},
		Valid:     true,
		Icon:      newIssueIcon,
	}
}

func globalIssueSearchItem(input string) alfred.Item {
	if len(input) > 0 {
		escaped := url.PathEscape(input)
		arg := "https://github.com/search?utf8=✓&type=Issues&q=" + escaped
		return alfred.Item{
			UID:       "ghs:",
			Title:     "Search issues for " + input,
			Arg:       arg,
			Valid:     true,
			Variables: alfred.Variables{"action": "open"},
			Icon:      searchIcon,
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
		Arg:          "https://github.com/" + repo,
		Valid:        true,
		Variables:    alfred.Variables{"action": "open"},
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
		Arg:          "https://github.com/" + repo + "/issues",
		Valid:        true,
		Variables:    alfred.Variables{"action": "open"},
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
		Arg:          "https://github.com/" + repo + "/projects",
		Valid:        true,
		Variables:    alfred.Variables{"action": "open"},
		Autocomplete: "p " + key,
		Icon:         projectIcon,
	}
}

func autocompleteOrgProjectItem(key, user string) alfred.Item {
	return alfred.Item{
		UID:          "ghp:" + user,
		Title:        fmt.Sprintf("List projects for %s (%s)", user, key),
		Arg:          "https://github.com/orgs/" + user + "/projects",
		Valid:        true,
		Variables:    alfred.Variables{"action": "open"},
		Autocomplete: "p " + key,
		Icon:         projectIcon,
	}
}

func autocompleteNewIssueItem(key, repo string) alfred.Item {
	return alfred.Item{
		UID:          "ghn:" + repo,
		Title:        fmt.Sprintf("New issue in %s (%s)", repo, key),
		Arg:          "https://github.com/" + repo + "/issues/new",
		Valid:        true,
		Variables:    alfred.Variables{"action": "open"},
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

func autocompleteItems(cfg config.Config, input string, parsed *parser.Result,
	repoItem func(string, string) alfred.Item,
	userItem func(string, string) alfred.Item,
	openEndedItem func(string) alfred.Item) (items alfred.Items) {

	parser := parser.NewUserCompletionParser(cfg.RepoMap, cfg.UserMap)
	result := parser.Parse(input)

	if strings.Contains(input, " ") {
		return
	}

	items = append(items,
		autocompleteRepoItems(cfg, input, repoItem)...)
	items = append(items,
		autocompleteUserItems(cfg, input, result, true, userItem)...)

	if len(input) == 0 || result.Repo() != input {
		items = append(items, openEndedItem(input))
	}
	return
}

func autocompleteRepoItems(cfg config.Config, input string,
	repoItem func(string, string) alfred.Item) (items alfred.Items) {
	if len(input) > 0 {
		for key, repo := range cfg.RepoMap {
			if strings.HasPrefix(key, input) && len(key) > len(input) {
				items = append(items, repoItem(key, repo))
			}
		}
	}
	return
}

func autocompleteUserItems(cfg config.Config, input string,
	parsed *parser.Result, includeMatchedUser bool,
	userItem func(string, string) alfred.Item) (items alfred.Items) {
	if len(input) > 0 {
		for key, user := range cfg.UserMap {
			prefixed := strings.HasPrefix(key, input) && len(key) > len(input)
			matched := includeMatchedUser && key == parsed.UserShorthand && !parsed.HasRepo()
			if prefixed || matched {
				items = append(items, userItem(key, user))
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

func (c *completion) rpcRequest(path, query string, delay float64) rpc.Result {
	if len(c.cfg.SocketPath) == 0 {
		panic("rpc not enabled") // should be exercised by tests only, FIXME remove
	}
	if c.env.Duration().Seconds() < delay {
		c.retry = true
		return rpc.Result{Complete: false}
	}

	res := c.rpcClient.Query(path, query)

	if !res.Complete && len(res.Error) == 0 {
		c.retry = true
	}

	return res
}

func ellipsis(prefix string, duration time.Duration) string {
	return prefix + strings.Repeat(".", int((duration.Nanoseconds()/250000000)%4))
}

// retrieveRepo adds a repo's description to an "open repo" item
// using an RPC call.
func (c *completion) retrieveRepo(repo string, item *alfred.Item) {
	if len(c.cfg.SocketPath) == 0 {
		return
	}
	res := c.rpcRequest("/repo", repo, delay)
	if len(res.Error) > 0 {
		item.Subtitle = res.Error
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

	if item.Mods != nil {
		item.Mods.Ctrl = &alfred.ModItem{
			Valid: true,
			Arg: fmt.Sprintf("[%s: %s](https://github.com/%s)",
				repo, res.Repos[0].Description, repo),
			Subtitle: fmt.Sprintf("Insert Markdown link with description to %s",
				repo),
			Variables: alfred.Variables{"action": "paste"},
			Icon:      markdownIcon,
		}
	}
}

// retrieveIssue adds the title and state to an "open issue" item
func (c *completion) retrieveIssue(repo, issuenum string, item *alfred.Item) {
	if len(c.cfg.SocketPath) == 0 {
		return
	}
	res := c.rpcRequest("/issue", repo+"#"+issuenum, delay)
	if len(res.Error) > 0 {
		item.Subtitle = res.Error
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
	if item.Mods != nil {
		item.Mods.Ctrl = &alfred.ModItem{
			Valid: true,
			Arg: fmt.Sprintf("[%s#%s: %s](https://github.com/%s/issues/%s)",
				repo, issuenum, issue.Title, repo, issuenum),
			Subtitle: fmt.Sprintf("Insert Markdown link with description to %s#%s",
				repo, issuenum),
			Variables: alfred.Variables{"action": "paste"},
			Icon:      markdownIcon,
		}
	}
}

func (c *completion) retrieveRepoProject(repo, issuenum string, item *alfred.Item) {
	c.retrieveProject(item, repo+"/"+issuenum)
}

func (c *completion) retrieveOrgProject(user, issuenum string, item *alfred.Item) {
	c.retrieveProject(item, user+"/"+issuenum)
}

func (c *completion) retrieveProject(item *alfred.Item, query string) {
	if len(c.cfg.SocketPath) == 0 {
		return
	}
	res := c.rpcRequest("/project", query, delay)
	if len(res.Error) > 0 {
		item.Subtitle = res.Error
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
}

func (c *completion) retrieveOrgProjects(user string, item *alfred.Item) alfred.Items {
	return c.retrieveProjects(item, user)
}

func (c *completion) retrieveRepoProjects(repo string, item *alfred.Item) alfred.Items {
	return c.retrieveProjects(item, repo)
}

func (c *completion) retrieveProjects(item *alfred.Item, query string) (projects alfred.Items) {
	if len(c.cfg.SocketPath) == 0 {
		return
	}
	res := c.rpcRequest("/projects", query, delay)
	if len(res.Error) > 0 {
		item.Subtitle = res.Error
		return
	} else if c.retry {
		item.Subtitle = ellipsis("Retrieving projects", c.env.Duration())
		return
	} else if len(res.Projects) == 0 {
		item.Subtitle = "No projects found"
		return
	}
	projects = append(projects, projectItemsFromProjects(res.Projects, "in "+query)...)
	return
}

func projectItemsFromProjects(projects []rpc.Project, desc string) alfred.Items {
	var items alfred.Items
	for _, project := range projects {
		// no UID so alfred doesn't remember these
		items = append(items, alfred.Item{
			Title:     project.Name,
			Subtitle:  fmt.Sprintf("Open project #%d %s", project.Number, desc),
			Valid:     true,
			Arg:       project.URL,
			Variables: alfred.Variables{"action": "open"},
			Icon:      projectStateIcon(project.State),
		})
	}
	return items
}

func (c *completion) retrieveIssueSearchItems(item *alfred.Item, repo, query string, includeRepo bool) alfred.Items {
	if len(repo) > 0 {
		query += " repo:" + repo + " "
	}
	return c.searchIssues(item, query, includeRepo, searchDelay)
}

func (c *completion) retrieveRecentIssues(repo string, item *alfred.Item) alfred.Items {
	return c.searchIssues(item, "repo:"+repo+" sort:updated-desc", false, issueListDelay)
}

func (c *completion) searchIssues(item *alfred.Item, query string, includeRepo bool, delay float64) alfred.Items {

	var items alfred.Items

	if !item.Valid || len(c.cfg.SocketPath) == 0 {
		return items
	}

	res := c.rpcRequest("/issues", query, delay)
	if len(res.Error) > 0 {
		item.Subtitle = res.Error
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

func issueItemsFromIssues(issues []rpc.Issue, includeRepo bool) alfred.Items {
	var items alfred.Items

	for _, issue := range issues {
		itemTitle := fmt.Sprintf("#%s %s", issue.Number, issue.Title)
		if includeRepo {
			itemTitle = issue.Repo + itemTitle
		}
		arg := ""
		if issue.Type == "Issue" {
			arg = "https://github.com/" + issue.Repo + "/issues/" + issue.Number
		} else {
			arg = "https://github.com/" + issue.Repo + "/pull/" + issue.Number
		}

		// no UID so alfred doesn't remember these
		items = append(items, alfred.Item{
			Title:     itemTitle,
			Subtitle:  fmt.Sprintf("Open %s#%s", issue.Repo, issue.Number),
			Valid:     true,
			Arg:       arg,
			Icon:      issueStateIcon(issue.Type, issue.State),
			Variables: alfred.Variables{"action": "open"},
			Mods:      issueMods(issue.Repo, issue.Number, issue.Title),
		})
	}

	return items
}

func repoMods(repo string) *alfred.Mods {
	return &alfred.Mods{
		Cmd: &alfred.ModItem{
			Valid:     true,
			Arg:       fmt.Sprintf("[%s](https://github.com/%s)", repo, repo),
			Subtitle:  fmt.Sprintf("Insert Markdown link to %s", repo),
			Icon:      markdownIcon,
			Variables: alfred.Variables{"action": "paste"},
		},
	}
}

func issueMods(repo, number, title string) *alfred.Mods {
	mods := &alfred.Mods{
		Cmd: &alfred.ModItem{
			Valid:     true,
			Arg:       fmt.Sprintf("[%s#%s](https://github.com/%s/issues/%s)", repo, number, repo, number),
			Subtitle:  fmt.Sprintf("Insert Markdown link to %s#%s", repo, number),
			Variables: alfred.Variables{"action": "paste"},
			Icon:      markdownIcon,
		},
		Alt: &alfred.ModItem{
			Valid:     true,
			Arg:       fmt.Sprintf("%s#%s", repo, number),
			Subtitle:  fmt.Sprintf("Insert issue reference to %s#%s", repo, number),
			Variables: alfred.Variables{"action": "paste"},
			Icon:      issueIcon,
		},
	}
	if len(title) > 0 {
		mods.Ctrl = &alfred.ModItem{
			Valid:     true,
			Arg:       fmt.Sprintf("[%s#%s: %s](https://github.com/%s/issues/%s)", repo, number, title, repo, number),
			Subtitle:  fmt.Sprintf("Insert Markdown link with description to %s#%s", repo, number),
			Variables: alfred.Variables{"action": "paste"},
			Icon:      markdownIcon,
		}
	}
	return mods
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
		if item.Text == nil && item.Variables != nil {
			if action, ok := item.Variables["action"]; ok && action == "open" {
				url := item.Arg
				c.result.Items[i].Text = &alfred.Text{Copy: url, LargeType: url}
			}
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
