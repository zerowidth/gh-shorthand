package snippets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

type urlTestCase struct {
	input  string
	output string
}

func TestMarkdownLink(t *testing.T) {
	tests := map[string]urlTestCase{
		"repo url": {
			input:  "https://github.com/zw/df",
			output: "[zw/df](https://github.com/zw/df)",
		},
		"not a repo-only url": {
			input:  "https://github.com/zw/df/foo",
			output: "https://github.com/zw/df/foo",
		},
		"issue url": {
			input:  "https://github.com/zw/df/issues/1",
			output: "[zw/df#1](https://github.com/zw/df/issues/1)",
		},
		"not an issue url": {
			input:  "github.com/zw/df/issues",
			output: "github.com/zw/df/issues",
		},
		"issue url with anchor": {
			input:  "https://github.com/zw/df/issues/1#whatever",
			output: "[zw/df#1](https://github.com/zw/df/issues/1)",
		},
		"pull request url": {
			input:  "https://github.com/zw/df/pull/1",
			output: "[zw/df#1](https://github.com/zw/df/pull/1)",
		},
		"extra text is ignored": {
			input:  "foo bar https://github.com/zw/df/issues/1 baz",
			output: "[zw/df#1](https://github.com/zw/df/issues/1)",
		},

		"discussion url": {
			input:  "https://github.com/orgs/gh/teams/foo/discussions/1",
			output: "[@gh/foo#1](https://github.com/orgs/gh/teams/foo/discussions/1)",
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			rpcClient := rpc.NewClient(config.Config{})
			assert.Equal(t, tc.output, MarkdownLink(rpcClient, tc.input, false))
		})
	}
}

func TestIssueReference(t *testing.T) {
	tests := map[string]urlTestCase{
		"issue url": {
			input:  "https://github.com/zw/df/issues/1",
			output: "zw/df#1",
		},
		"not an issue url": {
			input:  "github.com/zw/df/issues",
			output: "github.com/zw/df/issues",
		},
		"issue url with anchor": {
			input:  "https://github.com/zw/df/issues/1#whatever",
			output: "zw/df#1",
		},
		"pull request url": {
			input:  "https://github.com/zw/df/pull/1",
			output: "zw/df#1",
		},
		"extra text is ignored": {
			input:  "foo bar https://github.com/zw/df/issues/1 baz",
			output: "zw/df#1",
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			assert.Equal(t, tc.output, IssueReference(tc.input))
		})
	}

}
