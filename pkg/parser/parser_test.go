package parser

import (
	"fmt"
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

type testCase struct {
	input         string // the input
	bare          bool   // allow bare user
	ignoreNumeric bool   // whether or not to ignore numeric bare usernames
	repo          string // the expected repo match or expansion
	user          string // the expected user match or expansion
	repoMatch     string // the matched repo shorthand
	userMatch     string // the matched user shorthand
	issue         string // the expected issue match
	path          string // the expected path match
	query         string // the remaining query text after parsing/expansion
}

func (tc *testCase) assert(t *testing.T) {
	result := Parse(repoMap, userMap, tc.input, tc.bare, tc.ignoreNumeric)

	assert.Equal(t, tc.repo, result.Repo(), "result.Repo() with input %#v", tc.input)
	if len(tc.user) > 0 {
		assert.Equal(t, tc.user, result.User, "result.User with input %#v", tc.input)
	}
	assert.Equal(t, tc.repoMatch, result.RepoMatch, "result.RepoMatch with input %#v", tc.input)
	assert.Equal(t, tc.userMatch, result.UserMatch, "result.UserMatch with input %#v", tc.input)
	assert.Equal(t, tc.issue, result.Issue(), "result.Issue() with input %#v", tc.input)
	assert.Equal(t, tc.path, result.Path(), "result.Path() with input %#v", tc.input)
	assert.Equal(t, tc.query, result.Query, "result.Query with input %#v", tc.input)
}

func TestParse(t *testing.T) {
	for desc, tc := range map[string]*testCase{
		"no issue, no repo": {
			input: "",
		},
		"shorthand match": {
			input:     "df",
			repo:      "zerowidth/dotfiles",
			repoMatch: "df",
		},
		"no match, leading space": {
			input:     " df",
			repo:      "",
			repoMatch: "",
			query:     "df",
		},
		"fully qualified repo name": {
			input:     "foo/bar",
			repo:      "foo/bar",
			repoMatch: "",
		},
		"normal expansion": {
			input:     "df 123",
			repo:      "zerowidth/dotfiles",
			issue:     "123",
			repoMatch: "df",
			query:     "123",
		},
		"expansion with #": {
			input:     "df#123",
			repo:      "zerowidth/dotfiles",
			issue:     "123",
			repoMatch: "df",
			query:     "#123",
		},
		"space and # both": {
			input:     "df #123",
			repo:      "zerowidth/dotfiles",
			issue:     "123",
			repoMatch: "df",
			query:     "#123",
		},
		"single digit issue": {
			input:     "df 1",
			repo:      "zerowidth/dotfiles",
			issue:     "1",
			repoMatch: "df",
			query:     "1",
		},
		"numeric suffix on match": {
			input:     "df2 34",
			repo:      "zerowidth/dotfiles2",
			issue:     "34",
			repoMatch: "df2",
			query:     "34",
		},
		"fully qualified repo and issue": {
			input:     "foo/bar 123",
			repo:      "foo/bar",
			issue:     "123",
			repoMatch: "",
			query:     "123",
		},
		"invalid issue": {
			input:     "df 0123",
			repo:      "zerowidth/dotfiles",
			issue:     "",
			repoMatch: "df",
			query:     "0123",
		},
		"matches query after repo expansion": {
			input:     "df foo",
			repo:      "zerowidth/dotfiles",
			repoMatch: "df",
			query:     "foo",
		},
		"treats unparsed issue as query": {
			input:     "df 123 foo",
			repo:      "zerowidth/dotfiles",
			issue:     "",
			repoMatch: "df",
			query:     "123 foo",
		},
		"treats issue with any other text as a query": {
			input: "123 foo",
			issue: "",
			query: "123 foo",
		},
		"retrieve query": {
			input: "foo bar",
			query: "foo bar",
		},
		"ignores whitespace after shorthand": {
			input:     "df ",
			repo:      "zerowidth/dotfiles",
			repoMatch: "df",
			query:     "",
		},
		"ignores whitespace after repo": {
			input: "foo/bar ",
			repo:  "foo/bar",
			query: "",
		},
		"extracts path component after shorthand": {
			input:     "df /foo",
			repo:      "zerowidth/dotfiles",
			repoMatch: "df",
			path:      "/foo",
			query:     "/foo",
		},
		"extracts path component after repo": {
			input: "foo/bar /baz",
			repo:  "foo/bar",
			path:  "/baz",
			query: "/baz",
		},
		"extracts path component after repo without space": {
			input: "foo/bar/baz",
			repo:  "foo/bar",
			path:  "/baz",
			query: "/baz",
		},
		"ignores path after issue number": {
			input: "123 /foo",
			query: "123 /foo",
		},
		"parses as repo, not path": {
			input: "foo/bar",
			repo:  "foo/bar",
		},
		"expands user with empty repo name": {
			input:     "zw/",
			user:      "zerowidth",
			userMatch: "zw",
		},
		"expands user in repo declaration": {
			input:     "zw/foo",
			user:      "zerowidth",
			repo:      "zerowidth/foo",
			userMatch: "zw",
		},
		"matches non-shorthand user with empty repo": {
			input: "foo/",
			user:  "foo",
			query: "",
		},
		"strips whitespace from query": {
			input: " x    ",
			query: "x",
		},
		"requires exact match for repo shorthand expansion": {
			input: "dfx/foo",
			user:  "dfx",
			repo:  "dfx/foo",
		},
		"requires exact match for user shorthand expansion": {
			input: "zwx/foo",
			user:  "zwx",
			repo:  "zwx/foo",
		},
		"expands duplicate repo with shorthand by itself": {
			input:     "dupe",
			user:      "dupe-repo",
			repo:      "dupe-repo/stuff",
			repoMatch: "dupe",
		},
		"expands duplicate user with trailing slash": {
			input:     "dupe/",
			user:      "dupe-user",
			userMatch: "dupe",
		},
		"expands duplicate user with repo appended to shorthand": {
			input:     "dupe/foo",
			user:      "dupe-user",
			userMatch: "dupe",
			repo:      "dupe-user/foo",
		},
		"matches bare user when allowed": {
			input: "foo",
			bare:  true,
			user:  "foo",
		},
		"matches bare user and leaves the remainder as a query": {
			input: "foo bar",
			bare:  true,
			user:  "foo",
			query: "bar",
		},
		"allows trailing separator on bare user": {
			input: "foo/",
			bare:  true,
			user:  "foo",
		},
		"expands bare user shorthand": {
			input:     "zw",
			bare:      true,
			user:      "zerowidth",
			userMatch: "zw",
		},
		"expands bare user with query": {
			input:     "zw foo",
			bare:      true,
			user:      "zerowidth",
			userMatch: "zw",
			query:     "foo",
		},
		"can ignore numeric-only username for bare user": {
			input:         "1234",
			bare:          true,
			ignoreNumeric: true,
			user:          "",
			issue:         "1234",
			query:         "1234",
		},
	} {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, desc), tc.assert)
	}

}

func TestSetRepo(t *testing.T) {
	result := &Result{}
	result.SetRepo("foo/bar")
	assert.Equal(t, "foo/bar", result.Repo())
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
		test:  "matches a fully qualified repo",
		input: "zerowidth/dotfiles",
		user:  "zerowidth",
		name:  "dotfiles",
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
			parser := NewParser(repoMap, userMap, tc.defaultRepo,
				RequireRepo, WithIssue, WithPath)
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
			parser := NewParser(repoMap, userMap, tc.defaultRepo,
				RequireRepo, WithQuery)
			result := parser.Parse(tc.input)

			assert.Equal(t, tc.user, result.User, "result.User")
			assert.Equal(t, tc.name, result.Name, "result.Name")
			assert.Equal(t, tc.repoShorthand, result.RepoShorthand, "result.RepoShorthand")
			assert.Equal(t, tc.userShorthand, result.UserShorthand, "result.UserShorthand")
			assert.Equal(t, tc.query, result.Query, "result.Query")
		})
	}
}
