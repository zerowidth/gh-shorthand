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

func TestProjectDirMap(t *testing.T) {
	config, _ := Load(configYaml)
	dirs := config.ProjectDirMap()
	if len(dirs) != 3 {
		t.Errorf("expected ProjectDirs() to have three entries in %#v", dirs)
	} else {

		dir, ok := dirs["/foo/foo"]
		if !ok {
			t.Errorf("expected /foo/foo in %#v", dirs)
		} else if !filepath.IsAbs(dir) {
			t.Errorf("expected expanded /foo/foo %s to be an absolute path", dir)
		}

		barPath, err := homedir.Expand("~/bar")
		if err != nil {
			t.Errorf("error expanding ~/bar: %#v", err.Error())
		} else {
			dir, ok := dirs["~/bar"]
			if !ok {
			} else if dir != barPath {
				t.Errorf("expected ~/bar to be expanded to %s, got %s", barPath, dir)
			}
		}

		dir, ok = dirs["relative"]
		if !ok {
			t.Errorf("expected relative dir in %#v", dirs)
		} else if !filepath.IsAbs(dir) {
			t.Errorf("expected relative path to be expanded to absolute path, got %s", dir)
		}
	}
}

func TestLoadFromFile(t *testing.T) {
	config, _ := LoadFromFile("../../../test/fixtures/config.yml")
	if !reflect.DeepEqual(config.RepoMap, repoMap) {
		t.Errorf("expected repo map to be %#v, got %#v", repoMap, config.RepoMap)
	}

	if config.DefaultRepo != "zerowidth/default" {
		t.Errorf("expected DefaultRepo to be %q, got %q", "zerowidth/default", config.DefaultRepo)
	}

	if _, err := LoadFromFile("../../../test/fixtures/nonexistent.yml"); err == nil {
		t.Error("expected missing yaml file to error, but no error occurred")
	}
}
