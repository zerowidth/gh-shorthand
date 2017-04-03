package config

import (
	"io/ioutil"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"
)

// Config is a shorthand configuration, read from a yaml file
type Config struct {
	RepoMap     map[string]string `yaml:"repos"`
	DefaultRepo string            `yaml:"default_repo"`
	ProjectDirs []string          `yaml:"project_dirs"`
}

// Load a Config from a yaml string.
func Load(yml string) (*Config, error) {
	var config Config

	if err := yaml.Unmarshal([]byte(yml), &config); err != nil {
		return nil, err
	}

	// normalize the project directories
	paths := make([]string, len(config.ProjectDirs))
	for i, path := range config.ProjectDirs {
		expanded, err := homedir.Expand(path)
		if err != nil {
			return nil, err
		}
		absolute, err := filepath.Abs(expanded)
		if err != nil {
			return nil, err
		}
		paths[i] = absolute
	}
	config.ProjectDirs = paths

	return &config, nil
}

// LoadFromFile attempts to load a Config from a given yaml file.
func LoadFromFile(path string) (*Config, error) {
	yml, err := ioutil.ReadFile(path)
	if err != nil {
		return &Config{}, err
	}
	return Load(string(yml))
}
