package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var repoMap = map[string]string{
	"df":   "zerowidth/dotfiles",
	"df2":  "zerowidth/dotfiles2", // prefix collision
	"dupe": "dupe-repo/stuff",     // duplicate shorthand in userMap
}

var userMap = map[string]string{
	"zw":   "zerowidth",
	"dupe": "dupe-user",
}

var repoTests = []struct {
	// input:
	test        string // test name
	input       string // input
	defaultRepo string // default repo, if set

	// assertions:
	user          string
	name          string
	userShorthand string
	repoShorthand string
	issue         string
	path          string
}{

	// basic shorthand tests
	{
		test:  "empty input matches nothing",
		input: "",
	},
	{
		test:  "blank input matches nothing",
		input: "     ",
	},
	{
		test:  "matches a fully qualified repo",
		input: "zerowidth/dotfiles",
		user:  "zerowidth",
		name:  "dotfiles",
	},
	{
		test:  "ignores trailing whitespace",
		input: "foo/bar",
		user:  "foo",
		name:  "bar",
	},
	{
		test:  "does not match leading space",
		input: " foo/bar",
	},
	{
		test:          "expands repo shorthand",
		input:         "df",
		user:          "zerowidth",
		name:          "dotfiles",
		repoShorthand: "df",
	},
	{
		test:          "expands user shorthand when part of a repo",
		input:         "zw/foo",
		user:          "zerowidth",
		name:          "foo",
		userShorthand: "zw",
	},
	{
		test:          "expands duplicate as repo",
		input:         "dupe",
		user:          "dupe-repo",
		name:          "stuff",
		repoShorthand: "dupe",
	},
	{
		test:          "expands duplicate user with repo as user",
		input:         "dupe/foo",
		user:          "dupe-user",
		name:          "foo",
		userShorthand: "dupe",
	},
	{
		test:        "sets default repo if no repo match",
		input:       "",
		defaultRepo: "default/repo",
		user:        "default",
		name:        "repo",
	},
	{
		test:        "does not use default repo when repo matches",
		input:       "foo/bar",
		defaultRepo: "baz/mumble",
		user:        "foo",
		name:        "bar",
	},
	{
		test:  "does not match trailing text with valid repo",
		input: "valid/repo x",
	},
	{
		test:        "does not match invalid query with default repo",
		input:       "..invalid",
		defaultRepo: "default/repo",
	},
	{
		test:  "does not match a bare user",
		input: "foo",
	},
	{
		test:        "does not match a bare user with a default repo",
		input:       "baz",
		defaultRepo: "foo/bar",
	},
	{
		test:  "does not match trailing slash on user",
		input: "foo/",
	},

	// issue parsing
	{
		test:  "matches a numeric-only issue",
		input: "foo/bar 123",
		user:  "foo",
		name:  "bar",
		issue: "123",
	},
	{
		test:  "matches a hash-delimited issue with no space",
		input: "foo/bar#123",
		user:  "foo",
		name:  "bar",
		issue: "123",
	},
	{
		test:  "matches a hash-delimited issue with space",
		input: "foo/bar #123",
		user:  "foo",
		name:  "bar",
		issue: "123",
	},
	{
		test:        "matches an issue with a default repo",
		input:       "123",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		issue:       "123",
	},
	{
		test:        "matches a hash-delimited issue with a default repo",
		input:       "#123",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		issue:       "123",
	},
	{
		test:          "matches an issue with expanded repo shorthand",
		input:         "df 123",
		user:          "zerowidth",
		name:          "dotfiles",
		issue:         "123",
		repoShorthand: "df",
	},
	{
		test:          "matches an issue with expanded user shorthand",
		input:         "zw/dotfiles 123",
		user:          "zerowidth",
		name:          "dotfiles",
		issue:         "123",
		userShorthand: "zw",
	},
	{
		test:  "does not match invalid issue",
		input: "foo/bar 0123",
	},

	// path parsing
	{
		test:  "matches a path",
		input: "foo/bar /pulls",
		user:  "foo",
		name:  "bar",
		path:  "/pulls",
	},
	{
		test:  "matches a path with no space after repo",
		input: "foo/bar/pulls",
		user:  "foo",
		name:  "bar",
		path:  "/pulls",
	},
	{
		test:  "does not match a path with spaces",
		input: "foo/bar /pull another word",
	},
	{
		test:  "does not match a bare path",
		input: "/pulls",
	},
	{
		test:        "matches a path with a default repo",
		input:       "/pulls",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		path:        "/pulls",
	},

	// issue and path together
	{
		test:  "does not match an issue followed by a path",
		input: "foo/bar 123/foo",
	},
}

// TestRepoParser for testing the default "repo" mode parsing
func TestRepoParser(t *testing.T) {
	for _, tc := range repoTests {
		t.Run(tc.test, func(t *testing.T) {
			parser := NewRepoParser(repoMap, userMap, tc.defaultRepo)
			result := parser.Parse(tc.input)

			assert.Equal(t, tc.user, result.User, "result.User")
			assert.Equal(t, tc.name, result.Name, "result.Name")
			assert.Equal(t, tc.repoShorthand, result.RepoShorthand, "result.RepoShorthand")
			assert.Equal(t, tc.userShorthand, result.UserShorthand, "result.UserShorthand")
			assert.Equal(t, tc.issue, result.Issue, "result.Issue")
			assert.Equal(t, tc.path, result.Path, "result.Path")
		})
	}
}

var issueTests = []struct {
	// input:
	test        string // test name
	input       string // input
	defaultRepo string // default repo, if set

	// assertions:
	user          string
	name          string
	userShorthand string
	repoShorthand string
	query         string
}{
	// basic shorthand tests
	{
		test:  "empty input matches nothing",
		input: "",
	},
	{
		test:          "expands repo shorthand",
		input:         "df",
		user:          "zerowidth",
		name:          "dotfiles",
		repoShorthand: "df",
	},
	{
		test:  "matches a repo and a query",
		input: "foo/bar a query",
		user:  "foo",
		name:  "bar",
		query: "a query",
	},
	{
		test:  "does not match a query if no repo given",
		input: "a query",
	},
	{
		test:  "removes trailing space",
		input: "foo/bar q     ",
		user:  "foo",
		name:  "bar",
		query: "q",
	},
	{
		test:        "uses the default repo when no repo matches",
		input:       "q",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		query:       "q",
	},
}

// TestIssueParser for testing the "issue search" mode parsing
func TestIssueParser(t *testing.T) {
	for _, tc := range issueTests {
		t.Run(tc.test, func(t *testing.T) {
			parser := NewIssueParser(repoMap, userMap, tc.defaultRepo)
			result := parser.Parse(tc.input)

			assert.Equal(t, tc.user, result.User, "result.User")
			assert.Equal(t, tc.name, result.Name, "result.Name")
			assert.Equal(t, tc.repoShorthand, result.RepoShorthand, "result.RepoShorthand")
			assert.Equal(t, tc.userShorthand, result.UserShorthand, "result.UserShorthand")
			assert.Equal(t, tc.query, result.Query, "result.Query")
		})
	}
}

var projectTests = []struct {
	// input:
	test        string // test name
	input       string // input
	defaultRepo string // default repo, if set

	// assertions:
	user          string
	name          string
	userShorthand string
	repoShorthand string
	issue         string
	query         string
}{
	{
		test:  "parses a repo and project",
		input: "foo/bar 123",
		user:  "foo",
		name:  "bar",
		issue: "123",
	},
	{
		test:          "expands a repo for a project",
		input:         "df 123",
		user:          "zerowidth",
		name:          "dotfiles",
		repoShorthand: "df",
		issue:         "123",
	},
	{
		test:  "parses a user and a project",
		input: "foo 123",
		user:  "foo",
		issue: "123",
	},
	{
		test:          "expands a user from shorthand",
		input:         "zw 123",
		user:          "zerowidth",
		userShorthand: "zw",
		issue:         "123",
	},
	{
		test:          "expands a user in a repo from shorthand",
		input:         "zw/dotfiles",
		user:          "zerowidth",
		name:          "dotfiles",
		userShorthand: "zw",
	},
	{
		test:        "uses the default repo if set",
		input:       "123",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		issue:       "123",
	},
	{
		test:        "ignores default repo when parsing just a user",
		input:       "baz 123",
		defaultRepo: "foo/bar",
		user:        "baz",
		issue:       "123",
	},
	{
		test:  "allows a numeric user when no default repo set",
		input: "123",
		user:  "123",
	},
	{
		test:        "parses a bare numeric as an issue when default repo is set",
		input:       "123",
		defaultRepo: "foo/bar",
		user:        "foo",
		name:        "bar",
		issue:       "123",
	},

	{
		test:  "parses a numeric user followed by a project id",
		input: "123 456",
		user:  "123",
		issue: "456",
	},
	{
		test:        "does not parse a numeric user and project id when default repo set",
		input:       "123 456",
		defaultRepo: "foo/bar",
	},
}

// TestProjectParser for testing the "project" mode parsing
func TestProjectParser(t *testing.T) {
	for _, tc := range projectTests {
		t.Run(tc.test, func(t *testing.T) {
			parser := NewProjectParser(repoMap, userMap, tc.defaultRepo)
			result := parser.Parse(tc.input)

			assert.Equal(t, tc.user, result.User, "result.User")
			assert.Equal(t, tc.name, result.Name, "result.Name")
			assert.Equal(t, tc.repoShorthand, result.RepoShorthand, "result.RepoShorthand")
			assert.Equal(t, tc.userShorthand, result.UserShorthand, "result.UserShorthand")
			assert.Equal(t, tc.issue, result.Issue, "result.Issue")
			assert.Equal(t, tc.query, result.Query, "result.Query")
		})
	}
}

var userCompletionTests = []struct {
	// input:
	test  string // test name
	input string // input

	// assertions:
	user          string
	name          string
	userShorthand string
	repoShorthand string
}{
	{
		test:  "parses a repo",
		input: "foo/bar",
		user:  "foo",
		name:  "bar",
	},
	{
		test:          "expands repo shorthand",
		input:         "df",
		user:          "zerowidth",
		name:          "dotfiles",
		repoShorthand: "df",
	},
	{
		test:          "expands user shorthand with a repo",
		input:         "zw/foo",
		user:          "zerowidth",
		name:          "foo",
		userShorthand: "zw",
	},
	{
		test:  "matches just a user",
		input: "foo",
		user:  "foo",
	},
	{
		test:          "expands user shorthand",
		input:         "zw",
		user:          "zerowidth",
		userShorthand: "zw",
	},
	{
		test:  "matches a user with a trailing slash",
		input: "foo/",
		user:  "foo",
	},
	{
		test:          "expands user shorthand with a trailing slash",
		input:         "zw/",
		user:          "zerowidth",
		userShorthand: "zw",
	},
	{
		test:          "expands duplicate user with trailing slash as user",
		input:         "dupe/",
		user:          "dupe-user",
		userShorthand: "dupe",
	},
	{
		test:  "does not match invalid input",
		input: "foo bar",
	},
}

// TestUserCompletionParserfor testing the user completion parser
func TestUserCompletionParser(t *testing.T) {
	for _, tc := range userCompletionTests {
		t.Run(tc.test, func(t *testing.T) {
			parser := NewUserCompletionParser(repoMap, userMap)
			result := parser.Parse(tc.input)

			assert.Equal(t, tc.user, result.User, "result.User")
			assert.Equal(t, tc.name, result.Name, "result.Name")
			assert.Equal(t, tc.repoShorthand, result.RepoShorthand, "result.RepoShorthand")
			assert.Equal(t, tc.userShorthand, result.UserShorthand, "result.UserShorthand")
		})
	}
}
