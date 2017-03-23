package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	RepoMap map[string]string `yaml:"repos"`
}

func Load(yml string) (*Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(yml), &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func LoadFromFile(path string) (*Config, error) {
	yml, err := ioutil.ReadFile(path)
	if err != nil {
		return &Config{}, err
	}
	return Load(string(yml))
}
