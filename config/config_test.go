package config

import (
	"reflect"
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
	if !reflect.DeepEqual(config.RepoMap, repoMap) {
		t.Errorf("expected repo map to be %#v, got %#v", repoMap, config.RepoMap)
	}

	if _, err := Load(invalidYaml); err == nil {
		t.Error("expected invalid YML to error, but no error occurred")
	}
}

func TestLoadFromFile(t *testing.T) {
	config, _ := LoadFromFile("../fixtures/config.yml")
	if !reflect.DeepEqual(config.RepoMap, repoMap) {
		t.Errorf("expected repo map to be %#v, got %#v", repoMap, config.RepoMap)
	}

	if _, err := LoadFromFile("../fixtures/nonexistent.yml"); err == nil {
		t.Error("expected missing yaml file to error, but no error occurred")
	}
}
