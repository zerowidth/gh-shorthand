package snippets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zerowidth/gh-shorthand/pkg/rpc"
)

type urlTestCase struct {
	input  string
	output string
}

type rpcTestCase struct {
	input    string
	output   string
	endpoint string    // the endpoint that should be called
	query    string    // the query that should be used
	repo     rpc.Repo  // a repo to return
	issue    rpc.Issue // an issue to return
}

type fakeClient struct {
	endpoint string // to record the endpoint
	query    string // to record the query used
	repo     *rpc.Repo
	issue    *rpc.Issue
}

func (fc *fakeClient) Query(endpoint, query string) rpc.Result {
	// record the input
	fc.endpoint = endpoint
	fc.query = query

	// assemble the output
	repos := []rpc.Repo{}
	issues := []rpc.Issue{}
	if fc.repo != nil {
		repos = append(repos, *fc.repo)
	}
	if fc.issue != nil {
		issues = append(issues, *fc.issue)
	}

	return rpc.Result{Complete: true, Repos: repos, Issues: issues}
}

func TestMarkdownLink(t *testing.T) {
	tests := map[string]urlTestCase{
		"repo url": {
			input:  "https://github.com/zw/df",
			output: "[zw/df](https://github.com/zw/df)",
		},
		"repeated repo url": {
			input:  "[zw/df](https://github.com/zw/df)",
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
		"repeated issue url": {
			input:  "[zw/df#1](https://github.com/zw/df/issues/1)",
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
		"repeated pull request url": {
			input:  "[zw/df#1](https://github.com/zw/df/pull/1)",
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
		"expanded issue reference": {
			input:  "foo/bar#123",
			output: "[foo/bar#123](https://github.com/foo/bar/issues/123)",
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			rpcClient := rpc.NewClient("")
			assert.Equal(t, tc.output, MarkdownLink(rpcClient, tc.input, false))
		})
	}
}

func TestMarkdownLinkWithDescription(t *testing.T) {
	tests := map[string]rpcTestCase{
		"repo url": {
			input:    "https://github.com/zw/df",
			output:   "[zw/df: dotfiles](https://github.com/zw/df)",
			endpoint: "/repo",
			query:    "zw/df",
			repo:     rpc.Repo{Description: "dotfiles"},
		},
		"issue url": {
			input:    "https://github.com/zw/df/issues/1",
			output:   "[zw/df#1: a dotfiles issue](https://github.com/zw/df/issues/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "a dotfiles issue"},
		},
		"pull request url": {
			input:    "https://github.com/zw/df/pull/1",
			output:   "[zw/df#1: a dotfiles patch](https://github.com/zw/df/pull/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "a dotfiles patch"},
		},
		"expanded issue reference": {
			input:    "zw/df#1",
			output:   "[zw/df#1: a dotfiles patch](https://github.com/zw/df/issues/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "a dotfiles patch"},
		},
		"replaces [] with () in repo description": { // this breaks markdown in Bear.app
			input:    "https://github.com/zw/df",
			output:   "[zw/df: brackets ()](https://github.com/zw/df)",
			endpoint: "/repo",
			query:    "zw/df",
			repo:     rpc.Repo{Description: "brackets []"},
		},
		"replaces repeated :: with | in repo description": { // more weirdness in Bear.app with two sets of :: ::
			input:    "https://github.com/zw/df",
			output:   "[zw/df: A|Ruby|Constant](https://github.com/zw/df)",
			endpoint: "/repo",
			query:    "zw/df",
			repo:     rpc.Repo{Description: "A::Ruby::Constant"},
		},
		"does not replace single :: in repo description": { // but one :: is fine in Bear.app
			input:    "https://github.com/zw/df",
			output:   "[zw/df: Ruby::Constant](https://github.com/zw/df)",
			endpoint: "/repo",
			query:    "zw/df",
			repo:     rpc.Repo{Description: "Ruby::Constant"},
		},
		"replaces [] with () in issue title": { // this breaks markdown in Bear.app
			input:    "https://github.com/zw/df/issues/1",
			output:   "[zw/df#1: brackets ()](https://github.com/zw/df/issues/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "brackets []"},
		},
		"replaces repeated :: with | in issue title": { // more weirdness in Bear.app with two sets of :: ::
			input:    "https://github.com/zw/df/issues/1",
			output:   "[zw/df#1: A|Ruby|Constant](https://github.com/zw/df/issues/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "A::Ruby::Constant"},
		},
		"does not replace single :: in issue title": { // but one :: is fine in Bear.app
			input:    "https://github.com/zw/df/issues/1",
			output:   "[zw/df#1: Ruby::Constant](https://github.com/zw/df/issues/1)",
			endpoint: "/issue",
			query:    "zw/df#1",
			issue:    rpc.Issue{Title: "Ruby::Constant"},
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			client := &fakeClient{
				repo:  &tc.repo,
				issue: &tc.issue,
			}
			assert.Equal(t, tc.output, MarkdownLink(client, tc.input, true))
			assert.Equal(t, tc.endpoint, client.endpoint)
			assert.Equal(t, tc.query, client.query)
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
