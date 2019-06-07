package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetRepo(t *testing.T) {
	result := &Result{}
	result.SetRepo("foo/bar")
	assert.Equal(t, "foo/bar", result.Repo())

}

func TestSetRepoWithMissingRepo(t *testing.T) {
	result := *&Result{}
	result.SetRepo("foo/")
	assert.False(t, result.HasRepo())
	assert.True(t, result.HasUser())
	assert.Equal(t, "foo", result.User)
}
