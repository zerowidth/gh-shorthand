package parser

import (
	"regexp"
	"strings"
)

// Parser is a shorthand parser
type Parser struct {
	repoMap     map[string]string
	userMap     map[string]string
	defaultRepo string
	requireRepo bool // require a repository match
	parseRepo   bool // look for a repository match
	parseUser   bool // look for users
	parseIssue  bool // look for issues (#123, 123)
	parsePath   bool // look for /path
	parseQuery  bool // any extra text
}

// Option is a functional option to configure a Parser
type Option func(*Parser)

// NewParser returns a configured Parser
func NewParser(repoMap, userMap map[string]string, defaultRepo string, options ...Option) *Parser {
	parser := &Parser{
		repoMap:     repoMap,
		userMap:     userMap,
		defaultRepo: defaultRepo,
	}
	for _, option := range options {
		option(parser)
	}
	return parser
}

// NewRepoParser returns a parser for repo/issue/path queries
func NewRepoParser(repoMap, userMap map[string]string, defaultRepo string) *Parser {
	return NewParser(repoMap, userMap, defaultRepo, RequireRepo, WithIssue, WithPath)
}

// NewIssueParser returns a parser for issue searches
func NewIssueParser(repoMap, userMap map[string]string, defaultRepo string) *Parser {
	return NewParser(repoMap, userMap, defaultRepo, RequireRepo, WithQuery)
}

// NewProjectParser returns a parser for projects
func NewProjectParser(repoMap, userMap map[string]string, defaultRepo string) *Parser {
	return NewParser(repoMap, userMap, defaultRepo, WithRepo, WithUser, WithIssue, WithQuery)
}

// NewUserCompletionParser returns a parser for matching user/repo completion
// for autocomplete. Does not require a default repo.
func NewUserCompletionParser(repoMap, userMap map[string]string) *Parser {
	return NewParser(repoMap, userMap, "", WithRepo, WithUser)
}

// RequireRepo instructs the parser to require a repository
func RequireRepo(p *Parser) {
	p.parseRepo = true
	p.requireRepo = true
}

// WithRepo instructs the parser to look for a repo match
func WithRepo(p *Parser) { p.parseRepo = true }

// WithUser instructs the parser to look for a user
//
// When this is set alongside WithRepo, a repo will take precedence
func WithUser(p *Parser) { p.parseUser = true }

// WithIssue instructs the parser to look for issue (or project) numbers
func WithIssue(p *Parser) { p.parseIssue = true }

// WithPath instructs the parser to look for a path
func WithPath(p *Parser) { p.parsePath = true }

// WithQuery instructs the parser to match any remaining text as a query
func WithQuery(p *Parser) { p.parseQuery = true }

// Parse parses the given input and returns a result
func (p *Parser) Parse(input string) *Result {
	res := &Result{}

	if p.parseRepo {
		if repo := userRepoRegexp.FindString(input); len(repo) > 0 {
			// found a repository directly, check for expansion:
			res.SetRepo(repo)
			if shortUser, ok := p.userMap[res.User]; ok {
				res.UserShorthand = res.User
				res.User = shortUser
			}
			input = input[len(repo):]
		} else if user := userRegexp.FindString(input); len(user) > 0 {
			// found a user, see if it's repo shorthand:
			if shortRepo, ok := p.repoMap[user]; ok {
				res.SetRepo(shortRepo)
				res.RepoShorthand = user
				input = input[len(user):]
			} else if p.parseUser {
				// not repo shorthand, but we're allowed to match a user:
				res.User = user
				if shortUser, ok := p.userMap[user]; ok {
					res.UserShorthand = user
					res.User = shortUser
				}
				input = input[len(user):]
			}
		}

		// assign default repository if needed:
		if p.parseRepo && !res.HasRepo() && len(p.defaultRepo) > 0 {
			if p.parseUser && res.HasUser() {
				// if the matched user looks like an issue and there's no further input,
				// use the default repo and use the numeric user as an issue:
				if issue := issueRegexp.FindString(res.User); len(issue) > 0 {

					// if there's still input even after a numeric-looking user, this is
					// invalid. NB this _could_ be valid if parsing a query, but that use
					// case isn't needed/supported
					if len(input) > 0 {
						return &Result{}
					}

					res.Issue = res.User
					res.SetRepo(p.defaultRepo)
				}
			} else {
				res.SetRepo(p.defaultRepo)
			}
		}
	}

	// if we don't have a repo assigned by now, there's no match
	if p.requireRepo && !res.HasRepo() {
		return &Result{}
	}

	if p.parseIssue {
		if matches := issueRegexp.FindStringSubmatch(input); matches != nil {
			res.Issue = matches[1]
			input = input[len(matches[0]):]
		}
	}

	if p.parsePath {
		if matches := pathRegexp.FindStringSubmatch(input); matches != nil {
			res.Path = matches[1]
			input = input[len(matches[0]):]
		}
	}

	remainder := strings.TrimPrefix(strings.TrimRight(input, " "), " ")
	if p.parseQuery {
		// only remove the first leading space, and all trailing spaces
		res.Query = remainder
	} else if len(remainder) > 0 {
		res = &Result{} // invalid match, there's leftover characters
	}

	return res
}

var (
	// using (\A|\z|\W) since \b requires a \w on the left
	userRepoRegexp = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)/([\w\.\-]*)(\A|\z|\w)`) // user/repo
	userRegexp     = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9]*)\b`)                     // user
	issueRegexp    = regexp.MustCompile(`^ ?#?([1-9]\d*)$`)
	pathRegexp     = regexp.MustCompile(`^ ?(/\S*)$`)
)
