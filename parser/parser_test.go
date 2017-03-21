package parser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var repoMap = map[string]string{
	"df":  "zerowidth/dotfiles",
	"lg2": "libgit2/libgit2",
}

func testExpandRepo(t *testing.T, input string, repo string) {
	result := Parse(repoMap, input)
	msg := fmt.Sprintf("input %#v expected repo %#v to be %#v", input, result.Repo, repo)
	assert.Equal(t, repo, result.Repo, msg)
}

func testExpandRepoIssue(t *testing.T, input string, repo string, issue string) {
	result := Parse(repoMap, input)
	msg := fmt.Sprintf("input %#v expected repo %#v to be %#v", input, result.Repo, repo)
	assert.Equal(t, repo, result.Repo, msg)
	msg = fmt.Sprintf("input %#v expected issue %#v to be %#v", input, result.Issue, issue)
	assert.Equal(t, issue, result.Issue, msg)
}

func TestParse(t *testing.T) {
	testExpandRepo(t, "", "")
	testExpandRepo(t, "df", "zerowidth/dotfiles") // match shorthand
	testExpandRepo(t, " df", "")                  // no match, leading space
	testExpandRepo(t, "foo/bar", "foo/bar")       // fully qualified

	testExpandRepoIssue(t, "", "", "") // no issue nor repo
	testExpandRepoIssue(t, "df 123", "zerowidth/dotfiles", "123")
	testExpandRepoIssue(t, "df#123", "zerowidth/dotfiles", "123")
	testExpandRepoIssue(t, "df #123", "zerowidth/dotfiles", "123") // space and hash
	testExpandRepoIssue(t, "df123", "zerowidth/dotfiles", "123")   // prefix match
	testExpandRepoIssue(t, "lg2 123", "libgit2/libgit2", "123")
	testExpandRepoIssue(t, "lg2123", "libgit2/libgit2", "123") // prefix match
	testExpandRepoIssue(t, "foo/bar 123", "foo/bar", "123")    // fully qualified
	testExpandRepoIssue(t, "df 0123", "zerowidth/dotfiles", "") // invalid issue
}
