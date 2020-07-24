package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	rpcEnabled = "---\napi_token: abcdefg"

	repoMap = map[string]string{
		"df": "zerowidth/dotfiles",
	}

	userMap = map[string]string{
		"zw": "zerowidth",
	}
)

func TestLoad(t *testing.T) {
	config, err := Load(configYaml)
	require.NoError(t, err)
	assert.Equal(t, repoMap, config.RepoMap)
	assert.Equal(t, userMap, config.UserMap)
	assert.Equal(t, "zerowidth/default", config.DefaultRepo)

	_, err = Load(invalidYaml)
	assert.Error(t, err)

	_, err = Load(invalidRepo)
	assert.Error(t, err)
}

func TestRPCConfig(t *testing.T) {
	config, err := Load(configYaml)
	assert.NoError(t, err)
	assert.False(t, config.RPCEnabled())

	config, err = Load(rpcEnabled)
	assert.NoError(t, err)

	assert.True(t, config.RPCEnabled())
	assert.Equal(t, "abcdefg", config.APIToken)
	assert.Equal(t, "/tmp/gh-shorthand.sock", config.SocketPath, "should have a default value")
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

	assert.Equal(t, repoMap, config.RepoMap)
	assert.Equal(t, "zerowidth/default", config.DefaultRepo)

	_, err = LoadFromFile("testdata/nonexistent.yml")
	assert.Error(t, err)
}

func TestNoEditor(t *testing.T) {
	config := Config{}
	_, err := config.OpenEditorScript()
	assert.Error(t, err)
}

func TestEditor(t *testing.T) {
	cfg := Config{
		Editor: "/usr/local/bin/code -n",
	}
	s, err := cfg.OpenEditorScript()
	assert.NoError(t, err)
	assert.Equal(t, `/usr/local/bin/code -n "$path"`, s)
}

func TestEditorScript(t *testing.T) {
	cfg := Config{
		EditorScript: `exec /usr/local/bin/zsh -c '/usr/local/mvim "$path"'`,
	}
	s, err := cfg.OpenEditorScript()
	assert.NoError(t, err)
	assert.Equal(t, `exec /usr/local/bin/zsh -c '/usr/local/mvim "$path"'`, s)
}

func TestEditorScriptTakesPrecedence(t *testing.T) {
	cfg := Config{
		Editor:       "/usr/local/bin/code -n",
		EditorScript: `exec /usr/local/bin/zsh -c '/usr/local/mvim "$path"'`,
	}
	s, err := cfg.OpenEditorScript()
	assert.NoError(t, err)
	assert.Equal(t, `exec /usr/local/bin/zsh -c '/usr/local/mvim "$path"'`, s)
}
