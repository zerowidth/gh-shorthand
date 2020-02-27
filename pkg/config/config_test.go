package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configYaml = `---
repos:
  df: zerowidth/dotfiles
users:
  zw: zerowidth
default_repo: zerowidth/default
project_dirs:
  - /foo/foo
  - ~/bar
  - relative
`
	invalidYaml = "---\nrepos: []"

	// invalid repo, it's not "owner/name"
	invalidRepo = `---
repos:
	foo: bar
`

	invalidDefaultRepo = `---
default_repo: foo
`

	repoMap = map[string]string{
		"df": "zerowidth/dotfiles",
	}

	userMap = map[string]string{
		"zw": "zerowidth",
	}
)

func TestLoad(t *testing.T) {
	config, err := Load(configYaml)
	if err != nil {
		t.Errorf("could not load config yaml: %q", err.Error())
	} else {
		if !reflect.DeepEqual(config.RepoMap, repoMap) {
			t.Errorf("expected RepoMap to be %#v, got %#v", repoMap, config.RepoMap)
		}

		if !reflect.DeepEqual(config.UserMap, userMap) {
			t.Errorf("expected UserMap to be %#v, got %#v", userMap, config.UserMap)
		}

		if config.DefaultRepo != "zerowidth/default" {
			t.Errorf("expected DefaultRepo to be %q, got %q", "zerowidth/default", config.DefaultRepo)
		}

		if _, err := Load(invalidYaml); err == nil {
			t.Error("expected invalid YML to error, but no error occurred")
		}

		if _, err := Load(invalidRepo); err == nil {
			t.Error("expected invalid repos to result in an error, but no error occurred")
		}
	}
}

func TestLoadInvalidDefault(t *testing.T) {
	_, err := Load(invalidDefaultRepo)
	if assert.Error(t, err) {
		assert.Equal(t, "default repo \"foo\" not in owner/name format", err.Error())
	}
}

func TestLoadFromFile(t *testing.T) {
	config, err := LoadFromFile("testdata/config.yml")
	assert.NoError(t, err)

	if !reflect.DeepEqual(config.RepoMap, repoMap) {
		t.Errorf("expected repo map to be %#v, got %#v", repoMap, config.RepoMap)
	}

	if config.DefaultRepo != "zerowidth/default" {
		t.Errorf("expected DefaultRepo to be %q, got %q", "zerowidth/default", config.DefaultRepo)
	}

	if _, err := LoadFromFile("testdata/nonexistent.yml"); err == nil {
		t.Error("expected missing yaml file to error, but no error occurred")
	}
}
