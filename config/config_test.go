package config

import (
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

var configYaml = `---
repos:
  df: zerowidth/dotfiles
`
var invalidYaml = "---\nrepos: []"

var repoMap = map[string]string{
	"df": "zerowidth/dotfiles",
}

func TestLoad(t *testing.T) {
	config, _ := Load(configYaml)
	assert.Equal(t, repoMap, config.RepoMap)

	_, err := Load(invalidYaml)
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile("cannot unmarshal"), err.Error())
}

func TestLoadFromFile(t *testing.T) {
	config, _ := LoadFromFile("../fixtures/config.yml")
	assert.Equal(t, repoMap, config.RepoMap)

	_, err := LoadFromFile("../fixtures/nonexistent.yml")
	assert.Regexp(t, regexp.MustCompile("no such file"), err.Error())
}
