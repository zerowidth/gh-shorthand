package completion

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindProjectDirs(t *testing.T) {
	assert := assert.New(t)

	dirs, err := findProjectDirs("testdata/projects")
	assert.NoError(err)
	assert.Contains(dirs, "testdata/projects/project-bar", "normal directory in\n%v", dirs)
	assert.Contains(dirs, "testdata/projects/linked", "symlinked directory in\n%v", dirs)
	assert.NotContains(dirs, "testdata/projects/linked-file", "symlinked file in\n%v", dirs)
}

func TestFindProjectDirsGlob(t *testing.T) {
	assert := assert.New(t)

	dirs, err := findProjectDirs("testdata/*")
	assert.NoError(err)
	assert.Contains(dirs, "testdata/work/work-foo", "normal directory in\n%v", dirs)
	assert.Contains(dirs, "testdata/projects/project-bar", "normal directory in\n%v", dirs)
	assert.Contains(dirs, "testdata/projects/linked", "symlinked directory in\n%v", dirs)
	assert.NotContains(dirs, "testdata/projects/linked-file", "symlinked file in\n%v", dirs)
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

func TestProjectDirItemsWithFuzzySearch(t *testing.T) {
	fixturePath, err := filepath.Abs("testdata")
	require.NoError(t, err)
	dirs := projectDirItems([]string{"testdata/projects"}, "tdprojbar", modeEdit)
	require.Len(t, dirs, 1)
	assert.Equal(t, fixturePath+"/projects/project-bar", dirs[0].Arg)
}

func TestProjectDirItemsWithGlob(t *testing.T) {
	fixturePath, err := filepath.Abs("testdata")
	require.NoError(t, err)
	dirs := projectDirItems([]string{fixturePath + "/*"}, "", modeEdit)
	require.Len(t, dirs, 3)
	dirs = projectDirItems([]string{fixturePath + "/w*"}, "", modeEdit)
	require.Len(t, dirs, 1)
	assert.Equal(t, fixturePath+"/work/work-foo", dirs[0].Arg)
}
