package parser

import (
	"regexp"
	"strings"
)

// Result is a Parse result, returning the matched repo, issue, etc. as applicable
type Result struct {
	User      string // the repository owner, if present
	Name      string // the repository name, if present
	RepoMatch string // the matched repo shorthand value, if shorthand was expanded
	UserMatch string // the matched repo shorthand value, if shorthand was expanded
	Query     string // the remainder of the input
}

// HasOwner checks if the result has an owner.
func (r *Result) HasOwner() bool {
	return len(r.User) > 0
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
		return r.User + "/" + r.Name
	}
	return ""
}

// SetRepo overrides owner and name on the result from an `owner/name` string.
func (r *Result) SetRepo(repo string) {
	parts := strings.SplitN(repo, "/", 2)
	r.User = parts[0]
	r.Name = parts[1]
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
func (r *Result) Annotation() string {
	var a string
	if len(r.RepoMatch) > 0 {
		a += " (" + r.RepoMatch
		if r.HasIssue() {
			a += "#" + r.Issue()
		}
		a += ")"
	} else if len(r.UserMatch) > 0 {
		a += " (" + r.UserMatch + ")"
	}
	return a
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

	res.Query = strings.Trim(input, " ")

	return res
}

var (
	// using (\A|\z|\W) since \b requires a \w on the left
	userRepoRegexp = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)/([\w\.\-]*)(\A|\z|\w)`) // user/repo
	userRegexp     = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)\b`)                     // user
	issueRegexp    = regexp.MustCompile(`^#?([1-9]\d*)$`)
	pathRegexp     = regexp.MustCompile(`^(/\S*)$`)
)
