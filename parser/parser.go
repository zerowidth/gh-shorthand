package parser

import (
	"regexp"
	"sort"
	"strings"
)

// Result is a Parse result, returning the matched repo, issue, etc. as applicable
type Result struct {
	Repo  string // the matched/expanded repo, if applicable
	Issue string // the matched issue number, if applicable
	Match string // the matched  value, if applicable
	Query string // the remainder of the input, if not otherwise parsed
}

// Parse takes a repo mapping and input string and attempts to extract a repo,
// issue, etc. from the input using the repo map for shorthand expansion.
func Parse(repoMap map[string]string, input string) *Result {
	repo, match, query := extractRepo(repoMap, input)
	issue, query := extractIssue(query)
	return &Result{repo, issue, match, query}
}

var userRepoRegexp = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*/[\w\.\-]+\b`) // user/repo

func extractRepo(repoMap map[string]string, input string) (repo, match, query string) {
	var keys []string
	for k := range repoMap {
		keys = append(keys, k)
	}

	// sort the keys in reverse so the longest is matched first
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, k := range keys {
		if strings.HasPrefix(input, k) {
			return repoMap[k], k, input[len(k):]
		}
	}

	result := userRepoRegexp.FindStringSubmatch(input)
	if len(result) > 0 {
		repo = result[0]
		return repo, "", input[len(repo):]
	}
	return "", "", input
}

func extractIssue(query string) (issue, remainder string) {
	re := regexp.MustCompile(`^[\s#]*([1-9]\d*)$`)
	match := re.FindStringSubmatch(query)
	if len(match) > 0 {
		issue = match[1]
		remainder = ""
	} else {
		issue = ""
		remainder = query
	}
	return
}
