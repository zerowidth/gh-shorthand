package parser

import (
	"regexp"
	"strings"
)

// Parse takes a user and repo mapping along with an input string and attempts
// to extract a repo, issue, path, or query, using the user and repo mappings
// for shorthand expansion.
//
// bareUser determines whether or not a bare username is allowed as input.
// ignoreNumeric determines whether or not to ignore a bare user if it's
// entirely numeric. if true, numeric-only will be parsed as an issue, not user.
func Parse(repoMap, userMap map[string]string, input string, bareUser, ignoreNumeric bool) Result {
	var res Result

	if r := userRepoRegexp.FindString(input); len(r) > 0 {
		res.SetRepo(r)
		if su, ok := userMap[res.User]; ok {
			res.UserMatch = res.User
			res.User = su
		}
		input = input[len(r):]
	} else if u := userRegexp.FindString(input); len(u) > 0 {
		if sr, ok := repoMap[u]; ok {
			res.SetRepo(sr)
			res.RepoMatch = u
			input = input[len(u):]
		} else if su, ok := userMap[u]; ok {
			res.UserMatch = u
			res.User = su
			input = input[len(u):]
		} else if bareUser && (!ignoreNumeric || !issueRegexp.MatchString(input)) {
			res.User = u
			input = input[len(u):]
		}
	}

	// only remove the first leading space
	res.Query = strings.TrimPrefix(strings.TrimRight(input, " "), " ")

	return res
}

var (
	// using (\A|\z|\W) since \b requires a \w on the left
	userRepoRegexp = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)/([\w\.\-]*)(\A|\z|\w)`) // user/repo
	userRegexp     = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)\b`)                     // user
	issueRegexp    = regexp.MustCompile(`^#?([1-9]\d*)$`)
	pathRegexp     = regexp.MustCompile(`^(/\S*)$`)
)
