package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var repoMap = map[string]string{
	"df": "zerowidth/dotfiles",
}

func testExpandRepo(t *testing.T, input string, repo string) {
	result := Parse(repoMap, input)
	assert.Equal(t, repo, result.Repo)
}

func TestParseWithRepoMapping(t *testing.T) {
	testExpandRepo(t, "", "")
}
