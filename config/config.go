package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config is a shorthand configuration, read from a yaml file
type Config struct {
	RepoMap     map[string]string `yaml:"repos"`
	DefaultRepo string            `yaml:"default_repo"`
}

// Load a Config from a yaml string.
func Load(yml string) (*Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(yml), &config)
	if err != nil {
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
