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
		input string // the input
		repo  string // the expected repo match or expansion
		match string // the matched repo shorthand
		desc  string // description of the test case
	}{
		{"", "", "", "no input, no repo match"},
		{"df", "zerowidth/dotfiles", "df", "match match"},
		{" df", "", "", "no match, leading space"},
		{"foo/bar", "foo/bar", "", "fully qualified repo name"},
	}

	repoIssueTests := []struct {
		input string // the input
		repo  string // the expected repo match or expansion
		issue string // the expected issue match
		match string // the matched repo shorthand
		query string // the remaining query text after parsing/expansion
		desc  string // description of the test case
	}{
		{
			input: "",
			desc:  "no issue, no repo",
		},
		{
			input: "df 123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			desc:  "normal expansion",
		},
		{
			input: "df#123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			desc:  "expansion with #",
		},
		{
			input: "df #123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			desc:  "space and # both",
		},
		{
			input: "df123",
			repo:  "zerowidth/dotfiles",
			issue: "123",
			match: "df",
			desc:  "prefix match",
		},
		{
			input: "df 1",
			repo:  "zerowidth/dotfiles",
			issue: "1",
			match: "df",
			desc:  "single digit issue",
		},
		{
			input: "df2 34",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
			desc:  "numeric suffix on match",
		},
		{
			input: "df234",
			repo:  "zerowidth/dotfiles2",
			issue: "34",
			match: "df2",
			desc:  "numerix suffix with no space",
		},
		{
			input: "foo/bar 123",
			repo:  "foo/bar",
			issue: "123",
			match: "",
			desc:  "fully qualified repo",
		},
		{
			input: "df 0123",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			desc:  "invalid issue",
		},
		{
			input: "df foo",
			repo:  "zerowidth/dotfiles",
			match: "df",
			query: "foo",
			desc:  "retrieve query after expansion",
		},
		{
			input: "df 123 foo",
			repo:  "zerowidth/dotfiles",
			issue: "",
			match: "df",
			query: "123 foo",
			desc:  "treats unparsed issue as query",
		},
		{
			input: "123 foo",
			issue: "",
			query: "123 foo",
			desc:  "treats issue with any other text as a query",
		},
		{
			input: "foo bar",
			query: "foo bar",
			desc:  "retrieve query",
		},
	}

	for _, tc := range repoTests {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			result := Parse(repoMap, tc.input)
			if result.Repo != tc.repo {
				t.Errorf("expected repo %#v, got %#v", tc.repo, result.Repo)
			}
			if result.Match != tc.match {
				t.Errorf("expected match %#v, got %#v", tc.match, result.Match)
			}
		})
	}

	for _, tc := range repoIssueTests {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			result := Parse(repoMap, tc.input)
			if result.Repo != tc.repo {
				t.Errorf("expected repo %#v, got %#v", tc.repo, result.Repo)
			}
			if result.Issue != tc.issue {
				t.Errorf("expected issue %#v, got %#v", tc.issue, result.Issue)
			}
			if result.Match != tc.match {
				t.Errorf("expected match %#v, got %#v", tc.match, result.Match)
			}
		})
	}

}
