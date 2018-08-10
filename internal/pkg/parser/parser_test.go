package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var repoMap = map[string]string{
	"df":  "zerowidth/dotfiles",
	"df2": "zerowidth/dotfiles2", // prefix collision
}

var userMap = map[string]string{
	"zw": "zerowidth",
}

type testCase struct {
	input         string // the input
	bare          bool   // allow bare user
	ignoreNumeric bool   // whether or not to ignore numeric bare usernames
	repo          string // the expected repo match or expansion
	user          string // the expected user match or expansion
	hasRepo       bool   // should the parser have matched a repo?
	hasUser       bool   // should the parser have matched as a user?
	repoMatch     string // the matched repo shorthand
	userMatch     string // the matched user shorthand
	issue         string // the expected issue match
	path          string // the expected path match
	query         string // the remaining query text after parsing/expansion
}

func (tc *testCase) assert(t *testing.T) {
	t.Parallel()
	result := Parse(repoMap, userMap, tc.input, tc.bare, tc.ignoreNumeric)

	if result.Repo() != tc.repo {
		t.Errorf("expected Repo %#v, got %#v", tc.repo, result.Repo())
	}
	if len(tc.user) > 0 && result.User != tc.user {
		t.Errorf("expected Owner %#v, got %#v", tc.user, result.User)
	}
	if result.RepoMatch != tc.repoMatch {
		t.Errorf("expected RepoMatch %#v, got %#v", tc.repoMatch, result.RepoMatch)
	}
	if result.UserMatch != tc.userMatch {
		t.Errorf("expected UserMatch %#v, got %#v", tc.userMatch, result.UserMatch)
	}
	if result.Issue() != tc.issue {
		t.Errorf("expected Issue %#v, got %#v", tc.issue, result.Issue())
	}
	if result.Path() != tc.path {
		t.Errorf("expected Path %#v, got %#v", tc.path, result.Path())
	}
	if result.Query != tc.query {
		t.Errorf("expected Query %#v, got %#v", tc.query, result.Query)
	}
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
	t.Parallel()
	result := &Result{}
	result.SetRepo("foo/bar")
	assert.Equal(t, "foo/bar", result.Repo())
}
