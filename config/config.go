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
	UserMap     map[string]string `yaml:"users"`
	DefaultRepo string            `yaml:"default_repo"`
	ProjectDirs []string          `yaml:"project_dirs"`
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
func Load(yml string) (*Config, error) {
	var config Config

	if err := yaml.Unmarshal([]byte(yml), &config); err != nil {
		return nil, err
	}

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
