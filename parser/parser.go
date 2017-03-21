// shorthand parser
package parser

import (
	"regexp"
	"strings"
)

type Result struct {
	Repo  string
	Issue string
}

func Parse(repoMap map[string]string, input string) *Result {
	repo, query := extractRepo(repoMap, input)
	issue := extractIssue(query)
	return &Result{repo, issue}
}

func extractRepo(repoMap map[string]string, input string) (repo string, query string) {
	var keys []string
	for k := range repoMap {
		keys = append(keys, k)
	}
	// sort.Ints(keys)
	for _, k := range keys {
		if strings.HasPrefix(input, k) {
			return repoMap[k], input[len(k):]
		}
	}
	re := regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*/[\w\.\-]+\b`) // user/repo
	match := re.FindStringSubmatch(input)
	if len(match) > 0 {
		repo = match[0]
		return repo, input[len(repo):]
	}
	return "", input
}

func extractIssue(query string) (issue string) {
	re := regexp.MustCompile(`^[\s#]*([1-9]\d+)$`)
	match := re.FindStringSubmatch(query)
	if len(match) > 0 {
		issue = match[1]
	}
	return
}
