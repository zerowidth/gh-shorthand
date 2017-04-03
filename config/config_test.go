package config

import (
	"path/filepath"
	"reflect"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
)

var (
	configYaml = `---
repos:
  df: zerowidth/dotfiles
default_repo: zerowidth/default
project_dirs:
  - /foo/foo
  - ~/bar
  - relative
`
	invalidYaml = "---\nrepos: []"

	repoMap = map[string]string{
		"df": "zerowidth/dotfiles",
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

		if config.DefaultRepo != "zerowidth/default" {
			t.Errorf("expected DefaultRepo to be %q, got %q", "zerowidth/default", config.DefaultRepo)
		}

		if _, err := Load(invalidYaml); err == nil {
			t.Error("expected invalid YML to error, but no error occurred")
		}
	}
}

func TestLoadProjectDirs(t *testing.T) {
	config, _ := Load(configYaml)
	if len(config.ProjectDirs) != 3 {
		t.Errorf("expected ProjectDirs to have three entries, got %#v", config.ProjectDirs)
	} else {
		barPath, err := homedir.Expand("~/bar")
		if err != nil {
			t.Errorf("error expanding ~/bar: %#v", err.Error())
		}
		if config.ProjectDirs[0] != "/foo/foo" {
			t.Errorf("expected first project dir to be /foo/foo, got %#v", config.ProjectDirs[0])
		}
		if config.ProjectDirs[1] != barPath {
			t.Errorf("expected ~/bar to be expanded, got %#v", config.ProjectDirs[1])
		}
		if !filepath.IsAbs(config.ProjectDirs[2]) {
			t.Errorf("expected relative path to be expanded to absolute path, got %#v", config.ProjectDirs[2])
		}
	}

}

func TestLoadFromFile(t *testing.T) {
	config, _ := LoadFromFile("../fixtures/config.yml")
	if !reflect.DeepEqual(config.RepoMap, repoMap) {
		t.Errorf("expected repo map to be %#v, got %#v", repoMap, config.RepoMap)
	}

	if config.DefaultRepo != "zerowidth/default" {
		t.Errorf("expected DefaultRepo to be %q, got %q", "zerowidth/default", config.DefaultRepo)
	}

	if _, err := LoadFromFile("../fixtures/nonexistent.yml"); err == nil {
		t.Error("expected missing yaml file to error, but no error occurred")
	}
}
