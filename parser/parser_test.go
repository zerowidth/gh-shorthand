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
		input   string // the input
		repo    string // the expected repo match or expansion
		example string // what the case is an example of
	}{
		{"", "", "no input, no repo match"},
		{"df", "zerowidth/dotfiles", "match shorthand"},
		{" df", "", "no match, leading space"},
		{"foo/bar", "foo/bar", "fully qualified repo name"},
	}

	repoIssueTests := []struct {
		input   string // the input
		repo    string // the expected repo match or expansion
		issue   string // the expected issue match
		example string // what the case is an example of
	}{
		{"", "", "", "no issue no repo"},
		{"df 123", "zerowidth/dotfiles", "123", "normal expansion"},
		{"df#123", "zerowidth/dotfiles", "123", "expansion with #"},
		{"df #123", "zerowidth/dotfiles", "123", "space and # both"},
		{"df123", "zerowidth/dotfiles", "123", "prefix match"},
		{"df2 34", "zerowidth/dotfiles2", "34", "numeric suffix on shorthand"},
		{"df234", "zerowidth/dotfiles2", "34", "numerix suffix with no space"},
		{"foo/bar 123", "foo/bar", "123", "fully qualified repo"},
		{"df 0123", "zerowidth/dotfiles", "", "invalid issue"},
	}

	for _, tc := range repoTests {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, tc.example), func(t *testing.T) {
			result := Parse(repoMap, tc.input)
			if result.Repo != tc.repo {
				t.Errorf("expected repo %#v, got %#v", tc.repo, result.Repo)
			}
		})
	}

	for _, tc := range repoIssueTests {
		t.Run(fmt.Sprintf("Parse(%#v): %s", tc.input, tc.example), func(t *testing.T) {
			result := Parse(repoMap, tc.input)
			if result.Repo != tc.repo {
				t.Errorf("expected repo %#v, got %#v", tc.repo, result.Repo)
			}
			if result.Issue != tc.issue {
				t.Errorf("expected issue %#v, got %#v", tc.issue, result.Issue)
			}
		})
	}

}
