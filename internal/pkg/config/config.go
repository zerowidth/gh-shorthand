package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"
)

// Filename is the default file location for gh-shorthand config
var Filename = "~/.gh-shorthand.yml"

// Config is a shorthand configuration, read from a yaml file
type Config struct {
	RepoMap     map[string]string `yaml:"repos"`
	UserMap     map[string]string `yaml:"users"`
	DefaultRepo string            `yaml:"default_repo"`
	ProjectDirs []string          `yaml:"project_dirs"`
	APIToken    string            `yaml:"api_token"`
	SocketPath  string            `yaml:"socket_path"`
}

func (config Config) ProjectDirMap() (dirs map[string]string) {
	dirs = map[string]string{}
	for _, path := range config.ProjectDirs {
		expanded, err := homedir.Expand(path)
		if err != nil {
			continue
		}
		absolute, err := filepath.Abs(expanded)
		if err != nil {
			continue
		}
		dirs[path] = absolute
	}
	return
}

// Load a Config from a yaml string.
// Returns an empty config if an error occurs.
func Load(yml string) (Config, error) {
	var config Config

	if err := yaml.Unmarshal([]byte(yml), &config); err != nil {
		return Config{}, err
	}

	for k, v := range config.RepoMap {
		if !validRepoFormat(v) {
			return config, fmt.Errorf("repo shorthand %q: %q not in owner/name format", k, v)
		}
	}

	if len(config.DefaultRepo) > 0 {
		if !validRepoFormat(config.DefaultRepo) {
			return config, fmt.Errorf("default repo %q not in owner/name format", config.DefaultRepo)
		}
	}

	return config, nil
}

// LoadFromFile attempts to load a Config from a given yaml file.
// This always returns an empty config.
func LoadFromFile(path string) (Config, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return Config{}, err
	}

	yml, err := ioutil.ReadFile(realpath)
	if err != nil {
		return Config{}, err
	}

	return Load(string(yml))
}

// LoadFromDefault loads and validates a config.
//
// This is a convenience for the server subcommands. Exits with an error if
// config can't be loaded.
func LoadFromDefault() Config {
	cfg, err := LoadFromFile(Filename)
	if err != nil {
		log.Fatal("couldn't load config", err)
	}
	return cfg
}

func validRepoFormat(s string) bool {
	split := strings.Split(s, "/")
	if len(split) != 2 || len(split[0]) == 0 || len(split[1]) == 0 {
		return false
	}
	return true
}
