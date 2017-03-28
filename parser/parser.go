package parser

import (
	"regexp"
	"sort"
	"strings"
)

// Result is a Parse result, returning the matched repo, issue, etc. as applicable
type Result struct {
	Repo  string // the matched/expanded repo, if applicable
	Match string // the matched shorthand value, if applicable
	Issue string // the matched issue number, if applicable
	Path  string // the matched path fragment, if applicable
	Query string // the remainder of the input, if not otherwise parsed
}

// Parse takes a repo mapping and input string and attempts to extract a repo,
// issue, etc. from the input using the repo map for shorthand expansion.
func Parse(repoMap map[string]string, input string) *Result {
	path := ""
	repo, match, query := extractRepo(repoMap, input)
	issue, query := extractIssue(query)
	if issue == "" {
		path, query = extractPath(query)
	}
	return &Result{repo, match, issue, path, query}
}

var userRepoRegexp = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*/[\w\.\-]+\b`) // user/repo
var issueRegexp = regexp.MustCompile(`^#?([1-9]\d*)$`)
var pathRegexp = regexp.MustCompile(`^(/\S*)$`)

func extractRepo(repoMap map[string]string, input string) (repo, match, query string) {
	var keys []string
	for k := range repoMap {
		keys = append(keys, k)
	}

	// sort the keys in reverse so the longest is matched first
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, k := range keys {
		if strings.HasPrefix(input, k) {
			return repoMap[k], k, strings.TrimLeft(input[len(k):], " ")
		}
	}

	result := userRepoRegexp.FindStringSubmatch(input)
	if len(result) > 0 {
		repo = result[0]
		return repo, "", strings.TrimLeft(input[len(repo):], " ")
	}
	return "", "", input
}

func extractIssue(query string) (issue, remainder string) {
	match := issueRegexp.FindStringSubmatch(query)
	if len(match) > 0 {
		issue = match[1]
		remainder = ""
	} else {
		issue = ""
		remainder = query
	}
	return
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
