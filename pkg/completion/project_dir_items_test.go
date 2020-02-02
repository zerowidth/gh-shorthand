package completion

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindProjectDirs(t *testing.T) {
	assert := assert.New(t)

	fixturePath, _ := filepath.Abs("testdata/projects")
	dirs, err := findProjectDirs(fixturePath)
	assert.NoError(err)
	assert.Contains(dirs, "project-bar", "normal directory in\n%v", dirs)
	assert.Contains(dirs, "linked", "symlinked directory in\n%v", dirs)
	assert.NotContains(dirs, "linked-file", "symlinked file in\n%v", dirs)
}

func TestFindInvalidProjectDirs(t *testing.T) {
	assert := assert.New(t)

	fixturePath, err := filepath.Abs("testdata/invalid")
	assert.NoError(err)

	_, err = findProjectDirs(fixturePath)
	if assert.Error(err) {
		assert.Contains(err.Error(), "no such file")
	}
}
