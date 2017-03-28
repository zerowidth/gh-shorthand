package parser

import (
	"fmt"
	"testing"
)

var repoMap = map[string]string{
	"df":  "zerowidth/dotfiles",
	"df2": "zerowidth/dotfiles2", // prefix collision
}

func TestParse(t *testing.T) {
	repoTests := []struct {
		desc  string // description of the test case
		input string // the input
		repo  string // the expected repo match or expansion
		match string // the matched repo shorthand
		issue string // the expected issue match
		path  string // the expected path match
		query string // the remaining query text after parsing/expansion
	}{
		{
			desc:  "no issue, no repo",
			input: "",
		},
		{
			desc:  "shorthand match",
			input: "df",
			repo:  "zerowidth/dotfiles",
			match: "df",
		},
		{
			desc:  "no match, leading space",
			input: " df",
			repo:  "",
			match: "",
			query: " df",
		},
		{
			desc:  "fully qualified repo name",
			input: "foo/bar",
			repo:  "foo/bar",
			match: "",
		},
		{
			desc:  "normal expansion",
			input: "df 123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
		},
		{
			desc:  "expansion with #",
			input: "df#123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
		},
		{
			desc:  "space and # both",
			input: "df #123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
		},
		{
			desc:  "prefix match",
			input: "df123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
		},
		{
			desc:  "single digit issue",
			input: "df 1",
			repo:  "zerowidth/dotfiles",
			issue: "1",
			match: "df",
		},
		{
			desc:  "numeric suffix on match",
			input: "df2 34",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
		},
		{
			desc:  "numerix suffix with no space",
			input: "df234",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
		},
		{
			desc:  "fully qualified repo",
			input: "foo/bar 123",
			repo:  "foo/bar",
			issue: "123",
			match: "",
		},
		{
			desc:  "invalid issue",
			input: "df 0123",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			query: "0123",
		},
		{
			desc:  "retrieve query after expansion",
			input: "df foo",
			repo:  "zerowidth/dotfiles",
			match: "df",
			query: "foo",
		},
		{
			desc:  "treats unparsed issue as query",
			input: "df 123 foo",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			query: "123 foo",
		},
		{
			desc:  "treats issue with any other text as a query",
			input: "123 foo",
			issue: "",
			query: "123 foo",
		},
		{
			desc:  "retrieve query",
			input: "foo bar",
			query: "foo bar",
		},
		{
			desc:  "ignores whitespace after shorthand",
			input: "df ",
			repo:  "zerowidth/dotfiles",
			match: "df",
			query: "",
		},
		{
			desc:  "ignores whitespace after repo",
			input: "foo/bar ",
			repo:  "foo/bar",
			query: "",
		},
		{
			desc:  "extracts path component after shorthand",
			input: "df /foo",
			repo:  "zerowidth/dotfiles",
			match: "df",
			path:  "/foo",
		},
		{
			desc:  "extracts path component after repo",
			input: "foo/bar /baz",
			repo:  "foo/bar",
			path:  "/baz",
		},
		{
			desc:  "ignores path after issue number",
			input: "123 /foo",
			query: "123 /foo",
		},
		{
			desc:  "parses repo, not path",
			input: "foo/bar",
			repo:  "foo/bar",
		},
	}

	for _, tc := range repoTests {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			result := Parse(repoMap, tc.input)
			if result.Repo != tc.repo {
				t.Errorf("expected Repo %#v, got %#v", tc.repo, result.Repo)
			}
			if result.Match != tc.match {
				t.Errorf("expected Match %#v, got %#v", tc.match, result.Match)
			}
			if result.Issue != tc.issue {
				t.Errorf("expected Issue %#v, got %#v", tc.issue, result.Issue)
			}
			if result.Path != tc.path {
				t.Errorf("expected Path %#v, got %#v", tc.path, result.Path)
			}
			if result.Query != tc.query {
				t.Errorf("expected Query %#v, got %#v", tc.query, result.Query)
			}
		})
	}

}
