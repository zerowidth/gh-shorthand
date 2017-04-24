package parser

import (
	"fmt"
	"testing"
)

var repoMap = map[string]string{
	"df":  "zerowidth/dotfiles",
	"df2": "zerowidth/dotfiles2", // prefix collision
}

var userMap = map[string]string{
	"zw": "zerowidth",
}

type testCase struct {
	input string // the input
	repo  string // the expected repo match or expansion
	owner string // the expected user match or expansion
	match string // the matched repo shorthand
	issue string // the expected issue match
	path  string // the expected path match
	query string // the remaining query text after parsing/expansion
}

func (tc *testCase) assert(t *testing.T) {
	result := Parse(repoMap, userMap, tc.input)
	if result.Repo() != tc.repo {
		t.Errorf("expected Repo %#v, got %#v", tc.repo, result.Repo())
	}
	if len(tc.owner) > 0 && result.Owner != tc.owner {
		t.Errorf("expected Owner %#v, got %#v", tc.owner, result.Owner)
	}
	if result.Match != tc.match {
		t.Errorf("expected Match %#v, got %#v", tc.match, result.Match)
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
			input: "df",
			repo:  "zerowidth/dotfiles",
			match: "df",
		},
		"no match, leading space": {
			input: " df",
			repo:  "",
			match: "",
			query: "df",
		},
		"fully qualified repo name": {
			input: "foo/bar",
			repo:  "foo/bar",
			match: "",
		},
		"normal expansion": {
			input: "df 123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			query: "123",
		},
		"expansion with #": {
			input: "df#123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			query: "#123",
		},
		"space and # both": {
			input: "df #123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			query: "#123",
		},
		"prefix match": {
			input: "df123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			query: "123",
		},
		"single digit issue": {
			input: "df 1",
			repo:  "zerowidth/dotfiles",
			issue: "1",
			match: "df",
			query: "1",
		},
		"numeric suffix on match": {
			input: "df2 34",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
			query: "34",
		},
		"numerix suffix with no space": {
			input: "df234",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
			query: "34",
		},
		"fully qualified repo": {
			input: "foo/bar 123",
			repo:  "foo/bar",
			issue: "123",
			match: "",
			query: "123",
		},
		"invalid issue": {
			input: "df 0123",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			query: "0123",
		},
		"retrieve query after expansion": {
			input: "df foo",
			repo:  "zerowidth/dotfiles",
			match: "df",
			query: "foo",
		},
		"treats unparsed issue as query": {
			input: "df 123 foo",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			query: "123 foo",
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
			input: "df ",
			repo:  "zerowidth/dotfiles",
			match: "df",
			query: "",
		},
		"ignores whitespace after repo": {
			input: "foo/bar ",
			repo:  "foo/bar",
			query: "",
		},
		"extracts path component after shorthand": {
			input: "df /foo",
			repo:  "zerowidth/dotfiles",
			match: "df",
			path:  "/foo",
			query: "/foo",
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
		"parses repo, not path": {
			input: "foo/bar",
			repo:  "foo/bar",
		},
		"expands user": {
			input: "zw/",
			owner: "zerowidth",
			match: "zw",
			path:  "/",
			query: "/",
		},
		"expands user in repo declaration": {
			input: "zw/foo",
			owner: "zerowidth",
			repo:  "zerowidth/foo",
			match: "zw",
		},
		"does not match non-shorthand user": {
			input: "foo/",
			owner: "",
			query: "foo/",
		},
		"strips whitespace from query": {
			input: " x    ",
			query: "x",
		},
	} {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, desc), tc.assert)
	}

}
