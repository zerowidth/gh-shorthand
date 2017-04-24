package parser

import (
	"regexp"
	"sort"
	"strings"
)

// Result is a Parse result, returning the matched repo, issue, etc. as applicable
type Result struct {
	owner string
	name  string
	Match string // the matched shorthand value, if applicable
	Query string // the remainder of the input
	User  string // the matched user, if applicable
}

// HasRepo checks if the result has a repo, either from a matched repo shorthand,
// or from an explicit owner/name.
func (r *Result) HasRepo() bool {
	return len(r.name) > 0
}

// Repo returns the repo defined in the result, either from a matched repo
// shorthand or from an explicit owner/name.
func (r *Result) Repo() string {
	if r.HasRepo() {
		return r.owner + "/" + r.name
	}
	return ""
}

// SetRepo overrides owner and name on the result from an `owner/name` string.
func (r *Result) SetRepo(repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) > 1 {
		r.owner = parts[0]
		r.name = parts[1]
	}
	return nil
}

// HasIssue checks to see if the result's query looks like an issue reference.
func (r *Result) HasIssue() bool {
	return issueRegexp.MatchString(r.Query)
}

// Issue returns the issue reference from the query, if applicable.
func (r *Result) Issue() string {
	match := issueRegexp.FindStringSubmatch(r.Query)
	if len(match) > 0 {
		return match[1]
	}
	return ""
}

// Annotation is a helper for displaying details about a match. Returns a string
// with a leading space, noting the matched shorthand and issue if applicable.
func (r *Result) Annotation() (ann string) {
	if len(r.Match) > 0 {
		ann += " (" + r.Match
		if r.HasIssue() {
			ann += "#" + r.Issue()
		}
		ann += ")"
	}
	return
}

// RepoAnnotation is a helper for displaying details about a match, but only
// for user/repo matches, excluding issue.
func (r *Result) RepoAnnotation() (ann string) {
	if len(r.Match) > 0 {
		ann += " (" + r.Match + ")"
	}
	return
}

// Parse takes a repo mapping and input string and attempts to extract a repo,
// issue, etc. from the input using the repo map for shorthand expansion.
func Parse(repoMap, userMap map[string]string, input string) *Result {
	path := ""
	user := ""
	owner, name, match, query := extractRepo(repoMap, input)
	if len(name) == 0 {
		owner, match, query = extractUser(userMap, input)
		user = owner
	}
	path, query = extractPath(query)
	return &Result{
		owner: owner,
		name:  name,
		User:  user,
		Match: match,
		Path:  path,
		Query: query,
	}
}

var (
	userRepoRegexp = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)/([\w\.\-]+)\b`) // user/repo
	issueRegexp    = regexp.MustCompile(`^#?([1-9]\d*)$`)
	pathRegexp     = regexp.MustCompile(`^(/\S*)$`)
)

func extractRepo(repoMap map[string]string, input string) (owner, name, match, query string) {
	var keys []string
	for k := range repoMap {
		keys = append(keys, k)
	}

	// sort the keys in reverse so the longest is matched first
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, k := range keys {
		if strings.HasPrefix(input, k) {
			parts := strings.SplitN(repoMap[k], "/", 2)
			if len(parts) > 1 {
				return parts[0], parts[1], k, strings.TrimLeft(input[len(k):], " ")
			}
		}
	}

	result := userRepoRegexp.FindStringSubmatch(input)
	if len(result) > 0 {
		repo, owner, name := result[0], result[1], result[2]
		return owner, name, "", strings.TrimLeft(input[len(repo):], " ")
	}
	return "", "", "", input
}

func extractUser(userMap map[string]string, input string) (user, match, query string) {
	var keys []string
	for k := range userMap {
		keys = append(keys, k)
	}

	// sort the keys in reverse so the longest is matched first
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, k := range keys {
		if strings.HasPrefix(input, k) {
			return userMap[k], k, strings.TrimLeft(input[len(k):], " ")
		}
	}

	return "", "", input
}

func extractPath(query string) (path, remainder string) {
	match := pathRegexp.FindStringSubmatch(query)
	if len(match) > 0 {
		path = match[1]
		remainder = ""
	} else {
		path = ""
		remainder = query
	}
	return
}
