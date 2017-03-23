package parser

import (
	"testing"
)

var repoMap = map[string]string{
	"df":  "zerowidth/dotfiles",
	"df2": "zerowidth/dotfiles2", // prefix collision
	"lg2": "libgit2/libgit2",
}

func TestParse(t *testing.T) {
	repoTests := []struct {
		input string // the input
		repo  string // the expected repo match or expansion
	}{
		{"", ""},                     // no input, no repo match
		{"df", "zerowidth/dotfiles"}, // match shorthand
		{" df", ""},                  // no match, leading space
		{"foo/bar", "foo/bar"},       // fully qualified
	}

	repoIssueTests := []struct {
		input string // the input
		repo  string // the expected repo match or expansion
		issue string // the expected issue match
	}{
		{"", "", ""}, // no issue nor repo
		{"df 123", "zerowidth/dotfiles", "123"},
		{"df#123", "zerowidth/dotfiles", "123"},
		{"df #123", "zerowidth/dotfiles", "123"}, // space and hash
		{"df123", "zerowidth/dotfiles", "123"},   // prefix match
		{"df2 34", "zerowidth/dotfiles2", "34"},
		{"df234", "zerowidth/dotfiles2", "34"}, // prefix match
		{"lg2 123", "libgit2/libgit2", "123"},
		{"lg2123", "libgit2/libgit2", "123"},  // prefix match
		{"foo/bar 123", "foo/bar", "123"},     // fully qualified
		{"df 0123", "zerowidth/dotfiles", ""}, // invalid issue
	}

	for _, tc := range repoTests {
		result := Parse(repoMap, tc.input)
		if result.Repo != tc.repo {
			t.Errorf("input %#v: expected repo %#v, got %#v", tc.input, tc.repo, result.Repo)
		}
	}

	for _, tc := range repoIssueTests {
		result := Parse(repoMap, tc.input)
		if result.Repo != tc.repo {
			t.Errorf("input %#v: expected repo %#v, got %#v", tc.input, tc.repo, result.Repo)
		}
		if result.Issue != tc.issue {
			t.Errorf("input %#v: expected issue %#v, got %#v", tc.input, tc.issue, result.Issue)
		}
	}

}
