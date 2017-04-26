package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Result is a Parse result, returning the matched repo, issue, etc. as applicable
type Result struct {
	Owner     string // the repository owner, if present
	Name      string // the repository name, if present
	RepoMatch string // the matched repo shorthand value, if shorthand was expanded
	UserMatch string // the matched repo shorthand value, if shorthand was expanded
	Query     string // the remainder of the input
}

// HasRepo checks if the result has a repo, either from a matched repo shorthand,
// or from an explicit owner/name.
func (r *Result) HasRepo() bool {
	return len(r.Name) > 0
}

// Repo returns the repo defined in the result, either from a matched repo
// shorthand or from an explicit owner/name.
func (r *Result) Repo() string {
	if r.HasRepo() {
		return r.Owner + "/" + r.Name
	}
	return ""
}

// SetRepo overrides owner and name on the result from an `owner/name` string.
func (r *Result) SetRepo(repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) > 1 {
		r.Owner = parts[0]
		r.Name = parts[1]
	} else {
		return fmt.Errorf("repo %q does not look like `owner/name`", repo)
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

// HasPath checks to see if the query looks like a path (leading `/` followed by non-whitespace)
func (r *Result) HasPath() bool {
	return pathRegexp.MatchString(r.Query)
}

// Path returns the defined path from the query, if applicable.
func (r *Result) Path() string {
	match := pathRegexp.FindStringSubmatch(r.Query)
	if len(match) > 0 {
		return match[1]
	}
	return ""
}

// Annotation is a helper for displaying details about a match. Returns a string
// with a leading space, noting the matched shorthand and issue if applicable.
func (r *Result) Annotation() (ann string) {
	if len(r.RepoMatch) > 0 {
		ann += " (" + r.RepoMatch
		if r.HasIssue() {
			ann += "#" + r.Issue()
		}
		ann += ")"
	} else if len(r.UserMatch) > 0 {
		ann += " (" + r.UserMatch + ")"
	}
	return
}

// RepoAnnotation is a helper for displaying details about a match, but only
// for user/repo matches, excluding issue.
func (r *Result) RepoAnnotation() (ann string) {
	if len(r.RepoMatch) > 0 {
		ann += " (" + r.RepoMatch + ")"
	} else if len(r.UserMatch) > 0 {
		ann += " (" + r.UserMatch + ")"
	}
	return
}

// EmptyQuery returns true if no query has been provided.
func (r *Result) EmptyQuery() bool {
	return len(r.Query) == 0
}

// Parse takes a repo mapping and input string and attempts to extract a repo,
// issue, etc. from the input using the repo map for shorthand expansion.
func Parse(repoMap, userMap map[string]string, input string) *Result {
	owner, name, repoMatch, query := extractRepo(repoMap, input)
	userMatch := ""
	if len(name) == 0 {
		owner, userMatch, query = extractUser(userMap, input)
	} else {
		if expanded, match, _ := extractUser(userMap, owner); len(expanded) > 0 {
			owner = expanded
			userMatch = match
		}
	}
	query = strings.Trim(query, " ")
	return &Result{
		Owner:     owner,
		Name:      name,
		RepoMatch: repoMatch,
		UserMatch: userMatch,
		Query:     query,
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
			next := ""
			if len(input) > len(k) {
				next = input[len(k) : len(k)+1]
			}
			if next == "" || next == "/" || next == "#" || next == " " {
				parts := strings.SplitN(repoMap[k], "/", 2)
				if len(parts) > 1 {
					return parts[0], parts[1], k, strings.TrimLeft(input[len(k):], " ")
				}
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
			if len(input) == len(k) || input[len(k):len(k)+1] == "/" {
				return userMap[k], k, strings.TrimLeft(input[len(k):], " ")
			}
		}
	}

	return "", "", input
}
