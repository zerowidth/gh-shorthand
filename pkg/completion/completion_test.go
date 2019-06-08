package completion

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zerowidth/gh-shorthand/pkg/alfred"
	"github.com/zerowidth/gh-shorthand/pkg/config"
)

var defaultCfg = &config.Config{
	DefaultRepo: "zerowidth/default",
	RepoMap: map[string]string{
		"df":  "zerowidth/dotfiles",
		"df2": "zerowidth/df2",
	},
	UserMap: map[string]string{
		"zw": "zerowidth",
	},
	ProjectDirs: []string{"testdata/work", "testdata/projects"},
}

var defaultInMap = &config.Config{
	DefaultRepo: "zerowidth/dotfiles",
	RepoMap: map[string]string{
		"df": "zerowidth/dotfiles",
	},
}

var userRepoCollision = &config.Config{
	RepoMap: map[string]string{
		"zw": "zerowidth/dotfiles",
	},
	UserMap: map[string]string{
		"zw": "zeedub",
	},
}

var invalidDir = &config.Config{
	ProjectDirs: []string{"testdata/nonexistent"},
}

var emptyConfig = &config.Config{}

type completeTestCase struct {
	test         string         // test name
	input        string         // input string
	uid          string         // the results must contain an entry with this uid
	valid        bool           // and with the valid flag set to this
	title        string         // the expected title
	arg          string         // expected argument
	action       string         // expected action
	auto         string         // expected autocomplete arg
	cfg          *config.Config // config to use instead of the default cfg
	exclude      string         // exclude any item with this UID or title
	copy         string         // the clipboard copy string, if applicable
	cmdModArg    string         // cmd modifier argument, if applicable
	cmdModAction string         // cmd modifier action, if applicable
	altModArg    string         // alt modifier argument, if applicable
	altModAction string         // alt modifier action, if applicable
}

func (tc *completeTestCase) testItem(t *testing.T) {
	cfg := tc.cfg
	if cfg == nil {
		cfg = defaultCfg
	}

	env := Environment{
		Query: tc.input,
		Start: time.Now(),
	}

	result := Complete(*cfg, env)
	validateItems(t, result.Items)

	if len(tc.exclude) > 0 {
		_, ok := findMatchingItem(tc.exclude, tc.exclude, result.Items)
		if ok {
			t.Errorf("%s\nexpected no item with UID or Title %q", result.Items, tc.exclude)
		}
		return
	}

	if len(tc.uid) == 0 && len(tc.title) == 0 {
		t.Skip("skipping, uid/title/exclude not specified")
	}

	item, ok := findMatchingItem(tc.uid, tc.title, result.Items)
	if !ok {
		t.Logf("expected item with uid %q and/or title %q in %s", tc.uid, tc.title, result.Items)
		t.FailNow()
	}
	if len(tc.uid) > 0 {
		assert.Equal(t, tc.uid, item.UID, "item.UID in\n%+v", item)
	} else {
		t.Logf("no uid check for %#v", tc)
	}

	if len(tc.title) > 0 {
		assert.Equal(t, tc.title, item.Title, "item.Title in\n%#v", item)
	}

	assert.Equal(t, tc.valid, item.Valid, "item.Valid in\n%#v", item)

	if len(tc.arg) > 0 {
		assert.Equal(t, tc.arg, item.Arg, "item.Arg in\n%+v", item)
		// if assert.NotNil(t, item.Variables, "item.Variables") &&
		if item.Variables != nil &&
			assert.Contains(t, item.Variables, "action", "item.Variables['action']") {
			assert.Equal(t, tc.action, item.Variables["action"], "item.Variables['action']")
		}
	}

	if len(tc.auto) > 0 {
		assert.Equal(t, tc.auto, item.Autocomplete, "item.Autocomplete in\n%#v", item)
	}

	if len(tc.copy) > 0 {
		if assert.NotNil(t, item.Text, "item.Text in\n%#v", item) {
			assert.Equal(t, tc.copy, item.Text.Copy, "item.Text.Copy in\n%#v", item)
		}
	}

	if len(tc.cmdModArg) > 0 || len(tc.altModArg) > 0 {
		if assert.NotNil(t, item.Mods, "item.Mods in\n#%v", item) {
			if len(tc.cmdModArg) > 0 && assert.NotNil(t, item.Mods.Cmd, "item.Mods.Cmd") {
				assert.Equal(t, tc.cmdModArg, item.Mods.Cmd.Arg, "item.Mods.Cmd.Arg")
				// if assert.NotNil(t, item.Mods.Cmd.Variables, "item.Mods.Cmd.Variables") &&
				if item.Mods.Cmd.Variables != nil &&
					assert.Contains(t, item.Mods.Cmd.Variables, "action", "item.Mods.Cmd.Variables['action']") {
					assert.Equal(t, tc.cmdModAction, item.Mods.Cmd.Variables["action"], "item.Mods.Cmd.Variables['action']")
				}
			}

			if len(tc.altModArg) > 0 && assert.NotNil(t, item.Mods.Alt, "item.Mods.Alt") {
				assert.Equal(t, tc.altModArg, item.Mods.Alt.Arg, "items.Mods.Alt.Arg")
				// if assert.NotNil(t, item.Mods.Alt.Variables, "item.Mods.Alt.Variables") &&
				if item.Mods.Alt.Variables != nil &&
					assert.Contains(t, item.Mods.Alt.Variables, "action", "item.Mods.Alt.Variables['action']") {
					assert.Equal(t, tc.altModAction, item.Mods.Alt.Variables["action"], "item.Mods.Alt.Variables['action']")
				}
			}
		}
	}

	assert.NotContains(t, item.Subtitle, "rpc error")
}

func TestCompleteItems(t *testing.T) {
	fixturePath, _ := filepath.Abs("testdata")

	// Based on input, the resulting items must include one that matches either
	// the given UID or title. All items are also validated for correctness and
	// uniqueness by UID.
	for _, tc := range []completeTestCase{

		// defaults
		{
			test:  "empty input shows open repo/issue default",
			input: "",
			title: "Open repositories and issues on GitHub",
			auto:  " ",
		},
		{
			test:  "empty input shows issue list/search default",
			input: "",
			title: "List and search issues in a GitHub repository",
			auto:  "i ",
		},
		{
			test:  "empty input shows project list/search default",
			input: "",
			title: "List and open projects on GitHub repositories or organizations",
			auto:  "p ",
		},
		{
			test:  "empty input shows new issue default",
			input: "",
			title: "New issue in a GitHub repository",
			auto:  "n ",
		},
		{
			test:  "empty input shows edit project default",
			input: "",
			title: "Open a project",
			auto:  "e ",
		},
		{
			test:  "empty input shows search issues default",
			input: "",
			title: "Search issues across GitHub",
			auto:  "s ",
		},
		{
			test:  "a mode char by itself shows the default repo",
			input: "i",
			uid:   "ghi:zerowidth/default",
			valid: true,
		},
		{
			test:  "a mode char followed by a space shows the default repo",
			input: "i ",
			uid:   "ghi:zerowidth/default",
			valid: true,
		},
		{
			test:    "a mode char followed by a non-space shows nothing",
			input:   "ix",
			exclude: "ghi:zerowidth/default",
		},

		// basic parsing tests
		{
			test:      "open a shorthand repo",
			input:     " df",
			uid:       "gh:zerowidth/dotfiles",
			valid:     true,
			title:     "Open zerowidth/dotfiles (df)",
			arg:       "open https://github.com/zerowidth/dotfiles",
			copy:      "https://github.com/zerowidth/dotfiles",
			cmdModArg: "paste [zerowidth/dotfiles](https://github.com/zerowidth/dotfiles)",
		},
		{
			test:      "open a shorthand repo and issue",
			input:     " df 123",
			uid:       "gh:zerowidth/dotfiles#123",
			valid:     true,
			title:     "Open zerowidth/dotfiles#123 (df#123)",
			arg:       "open https://github.com/zerowidth/dotfiles/issues/123",
			copy:      "https://github.com/zerowidth/dotfiles/issues/123",
			altModArg: "paste zerowidth/dotfiles#123",
			cmdModArg: "paste [zerowidth/dotfiles#123](https://github.com/zerowidth/dotfiles/issues/123)",
		},
		{
			test:  "open a fully qualified repo",
			input: " foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar",
			arg:   "open https://github.com/foo/bar",
			copy:  "https://github.com/foo/bar",
		},
		{
			test:      "open a fully qualified repo and issue",
			input:     " foo/bar 123",
			uid:       "gh:foo/bar#123",
			valid:     true,
			title:     "Open foo/bar#123",
			arg:       "open https://github.com/foo/bar/issues/123",
			copy:      "https://github.com/foo/bar/issues/123",
			altModArg: "paste foo/bar#123",
			cmdModArg: "paste [foo/bar#123](https://github.com/foo/bar/issues/123)",
		},
		{
			test:  "open a shorthand user with repo",
			input: " zw/foo",
			uid:   "gh:zerowidth/foo",
			valid: true,
			title: "Open zerowidth/foo (zw)",
			arg:   "open https://github.com/zerowidth/foo",
			copy:  "https://github.com/zerowidth/foo",
		},
		{
			test:  "open a shorthand user with repo and issue",
			input: " zw/foo 123",
			uid:   "gh:zerowidth/foo#123",
			valid: true,
			title: "Open zerowidth/foo#123 (zw)",
			arg:   "open https://github.com/zerowidth/foo/issues/123",
			copy:  "https://github.com/zerowidth/foo/issues/123",
		},
		{
			test:    "no match if any unparsed query remains after shorthand",
			input:   " df foo",
			exclude: "gh:zerowidth/dotfiles",
		},
		{
			test:    "no match if any unparsed query remains after repo",
			input:   " foo/bar baz",
			exclude: "gh:foo/bar",
		},
		{
			test:  "ignores trailing whitespace for shorthand",
			input: " df ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
		},
		{
			test:  "ignores trailing whitespace for repo",
			input: " foo/bar ",
			uid:   "gh:foo/bar",
			valid: true,
		},
		{
			test:  "open path on matched shorthand repo",
			input: " df /foo",
			uid:   "gh:zerowidth/dotfiles/foo",
			valid: true,
			title: "Open zerowidth/dotfiles/foo (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/foo",
			copy:  "https://github.com/zerowidth/dotfiles/foo",
		},
		{
			test:    "don't open direct path when not prefixed with repo",
			input:   " /foo",
			exclude: "gh:/foo",
		},
		{
			test:    "don't open direct path when matching user prefix",
			input:   " zw/",
			exclude: "gh:/",
		},
		{
			test:  "prefer colliding repo expansion with shorthand alone",
			input: " zw",
			cfg:   userRepoCollision,
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
		},
		{
			test:  "prefer colliding user expansion with trailing repo name",
			input: " zw/foo",
			cfg:   userRepoCollision,
			uid:   "gh:zeedub/foo",
			valid: true,
		},
		{
			test:  "requires exact prefix for repo shorthand",
			input: " dfx/foo",
			uid:   "gh:dfx/foo",
			valid: true,
			title: "Open dfx/foo",
		},
		{
			test:  "requires exact prefix for user shorthand",
			input: " zwx/foo",
			uid:   "gh:zwx/foo",
			valid: true,
			title: "Open zwx/foo",
		},

		// issue index/search
		{
			test:  "open issues index on a shorthand repo",
			input: "i df",
			uid:   "ghi:zerowidth/dotfiles",
			valid: true,
			title: "List issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
			copy:  "https://github.com/zerowidth/dotfiles/issues",
		},
		{
			test:  "open issues index on a repo",
			input: "i foo/bar",
			uid:   "ghi:foo/bar",
			valid: true,
			title: "List issues for foo/bar",
			arg:   "open https://github.com/foo/bar/issues",
			copy:  "https://github.com/foo/bar/issues",
		},
		{
			test:  "search issues on a repo",
			input: "i a/b foo bar",
			uid:   "ghis:a/b",
			valid: true,
			title: "Search issues in a/b for foo bar",
			arg:   "open https://github.com/a/b/search?utf8=✓&type=Issues&q=foo%20bar",
			copy:  "https://github.com/a/b/search?utf8=✓&type=Issues&q=foo%20bar",
		},
		{
			test:  "search issues on a shorhthand repo",
			input: "i df foo bar",
			uid:   "ghis:zerowidth/dotfiles",
			valid: true,
			title: "Search issues in zerowidth/dotfiles (df) for foo bar",
			arg:   "open https://github.com/zerowidth/dotfiles/search?utf8=✓&type=Issues&q=foo%20bar",
			copy:  "https://github.com/zerowidth/dotfiles/search?utf8=✓&type=Issues&q=foo%20bar",
		},
		{
			test:  "search issues for a numeric string on a repo",
			input: "i a/b 12345",
			uid:   "ghis:a/b",
			valid: true,
			title: "Search issues in a/b for 12345",
		},

		// new issue
		{
			test:  "open a new issue in a shorthand repo",
			input: "n df",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new",
			copy:  "https://github.com/zerowidth/dotfiles/issues/new",
		},
		{
			test:  "open a new issue in a repo",
			input: "n a/b",
			uid:   "ghn:a/b",
			valid: true,
			title: "New issue in a/b",
			arg:   "open https://github.com/a/b/issues/new",
			copy:  "https://github.com/a/b/issues/new",
		},
		{
			test:  "open a new issue with a query",
			input: "n df foo bar",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df): foo bar",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new?title=foo%20bar",
			copy:  "https://github.com/zerowidth/dotfiles/issues/new?title=foo%20bar",
		},
		{
			test:  "search issues globally with no query",
			input: "s ",
			title: "Search issues for...",
			valid: false,
			auto:  "s ",
		},
		{
			test:  "search issues globally with a query",
			input: "s foo bar",
			title: "Search issues for foo bar",
			valid: true,
			arg:   "open https://github.com/search?utf8=✓&type=Issues&q=foo%20bar",
			copy:  "https://github.com/search?utf8=✓&type=Issues&q=foo%20bar",
		},

		// default repo
		{
			test:  "open an issue with the default repo",
			input: " 123",
			uid:   "gh:zerowidth/default#123",
			valid: true,
			title: "Open zerowidth/default#123",
			arg:   "open https://github.com/zerowidth/default/issues/123",
			copy:  "https://github.com/zerowidth/default/issues/123",
		},
		{
			test:  "open the default repo when default is also in map",
			cfg:   defaultInMap,
			input: " ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles",
			arg:   "open https://github.com/zerowidth/dotfiles",
			copy:  "https://github.com/zerowidth/dotfiles",
		},
		{
			test:    "includes no default if remaining input isn't otherwise valid",
			input:   " foo",
			exclude: "gh:zerowidth/default",
		},
		{
			test:  "show issues for a default repo",
			input: "i ",
			uid:   "ghi:zerowidth/default",
			valid: true,
			title: "List issues for zerowidth/default",
		},
		{
			test:  "search issues with a query in the default repo",
			input: "i foo",
			uid:   "ghis:zerowidth/default",
			valid: true,
			title: "Search issues in zerowidth/default for foo",
		},
		{
			test:  "new issue in the default repo",
			input: "n ",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default",
		},
		{
			test:  "new issue with a title in the default repo",
			input: "n foo",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default: foo",
		},

		// projects
		{
			test:  "project listing with explicit repo",
			input: "p zerowidth/dotfiles",
			uid:   "ghp:zerowidth/dotfiles",
			title: "List projects in zerowidth/dotfiles",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects",
			copy:  "https://github.com/zerowidth/dotfiles/projects",
		},
		{
			test:  "specific project with explicit repo",
			input: "p zerowidth/dotfiles 10",
			uid:   "ghp:zerowidth/dotfiles/10",
			title: "Open project #10 in zerowidth/dotfiles",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects/10",
			copy:  "https://github.com/zerowidth/dotfiles/projects/10",
		},
		{
			test:  "project listing with shorthand repo",
			input: "p df",
			uid:   "ghp:zerowidth/dotfiles",
			title: "List projects in zerowidth/dotfiles (df)",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects",
			copy:  "https://github.com/zerowidth/dotfiles/projects",
		},
		{
			test:  "specific project with shorthand repo",
			input: "p df 10",
			uid:   "ghp:zerowidth/dotfiles/10",
			title: "Open project #10 in zerowidth/dotfiles (df#10)",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects/10",
			copy:  "https://github.com/zerowidth/dotfiles/projects/10",
		},
		{
			test:  "project listing with org",
			input: "p zerowidth",
			uid:   "ghp:zerowidth",
			title: "List projects for zerowidth",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects",
			copy:  "https://github.com/orgs/zerowidth/projects",
		},
		{
			test:  "specific project with org",
			input: "p zerowidth 10",
			uid:   "ghp:zerowidth/10",
			title: "Open project #10 for zerowidth",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects/10",
			copy:  "https://github.com/orgs/zerowidth/projects/10",
		},
		{
			test:  "project listing with user shorthand",
			input: "p zw",
			uid:   "ghp:zerowidth",
			title: "List projects for zerowidth (zw)",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects",
			copy:  "https://github.com/orgs/zerowidth/projects",
		},
		{
			test:  "specific project with user shorthand",
			input: "p zw 10",
			uid:   "ghp:zerowidth/10",
			title: "Open project #10 for zerowidth (zw)",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects/10",
			copy:  "https://github.com/orgs/zerowidth/projects/10",
		},
		{
			test:  "specific project with numeric username treated as project",
			input: "p 123",
			uid:   "ghp:zerowidth/default/123",
			valid: true,
		},
		{
			test:  "specific project with numeric username but default repo treated as user",
			input: "p 123",
			cfg:   emptyConfig,
			uid:   "ghp:123",
			valid: true,
		},

		// repo autocomplete
		{
			test:    "no autocomplete for empty input",
			input:   " ",
			exclude: "gh:zerowidth/dotfiles",
		},
		{
			test:  "autocomplete 'd', first match",
			input: " d",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles",
			copy:  "https://github.com/zerowidth/dotfiles",
			auto:  " df",
		},
		{
			test:  "autocomplete 'd', second match",
			input: " d",
			uid:   "gh:zerowidth/df2",
			valid: true,
			title: "Open zerowidth/df2 (df2)",
			arg:   "open https://github.com/zerowidth/df2",
			copy:  "https://github.com/zerowidth/df2",
			auto:  " df2",
		},
		{
			test:  "autocomplete 'z', matching user shorthand",
			input: " z",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		{
			test:  "autocomplete when user shorthand matches exactly",
			input: " zw",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		{
			test:  "autocomplete when user shorthand has trailing slash",
			input: " zw/",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		{
			test:    "no autocomplete when user shorthand has text following the slash",
			input:   " zw/foo",
			exclude: "Open zerowidth/... (zw)",
		},
		{
			test:  "autocomplete 'd', open-ended",
			input: " d",
			title: "Open d...",
			valid: false,
		},
		{
			test:  "autocomplete open-ended when no default",
			cfg:   emptyConfig,
			input: " ",
			title: "Open ...",
			valid: false,
		},
		{
			test:  "autocomplete unmatched user prefix",
			input: " foo/",
			title: "Open foo/...",
			valid: false,
		},
		{
			test:    "does not autocomplete with fully-qualified repo",
			input:   " foo/bar",
			exclude: "Open foo/bar...",
		},
		{
			test:    "no autocomplete when input has space",
			input:   " foo bar",
			exclude: "Open foo bar...",
		},

		// issue index autocomplete
		{
			test:  "autocompletes for issue index",
			input: "i d",
			uid:   "ghi:zerowidth/dotfiles",
			valid: true,
			title: "List issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
			copy:  "https://github.com/zerowidth/dotfiles/issues",
			auto:  "i df",
		},
		{
			test:  "autocompletes issue index with input so far",
			input: "i foo",
			valid: false,
			title: "List issues for foo...",
			auto:  "i foo",
		},
		{
			test:  "autocomplete issues open-ended when no default",
			cfg:   emptyConfig,
			input: "i ",
			title: "List issues for ...",
			valid: false,
		},
		{
			test:  "autocomplete user for issues",
			input: "i z",
			title: "List issues for zerowidth/... (zw)",
			auto:  "i zw/",
		},

		// project autocomplete
		{
			test:  "autocompletes for repo projects",
			input: "p d",
			uid:   "ghp:zerowidth/dotfiles",
			valid: true,
			title: "List projects in zerowidth/dotfiles (df)",
			auto:  "p df",
		},
		{
			test:  "autocompletes user projects",
			input: "p z",
			uid:   "ghp:zerowidth",
			valid: true,
			title: "List projects for zerowidth (zw)",
			auto:  "p zw",
		},
		{
			test:  "autocompletes projects with input so far",
			input: "p foo",
			title: "List projects for foo...",
			valid: false,
		},
		{
			test:  "autocomplete open-ended projects when no default",
			input: "p ",
			cfg:   emptyConfig,
			title: "List projects for ...",
			valid: false,
		},

		// new issue autocomplete
		{
			test:  "autocompletes for new issue",
			input: "n d",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new",
			copy:  "https://github.com/zerowidth/dotfiles/issues/new",
			auto:  "n df",
		},
		{
			test:  "autocomplete user for new issue",
			input: "n z",
			title: "New issue in zerowidth/... (zw)",
			auto:  "n zw/",
		},
		{
			test:  "autocompletes new issue with input so far",
			input: "n foo",
			valid: false,
			title: "New issue in foo...",
			auto:  "n foo",
		},
		{
			test:  "autocomplete new issue open-ended when no default",
			cfg:   emptyConfig,
			input: "n ",
			title: "New issue in ...",
			valid: false,
		},

		{
			test:      "edit project includes fixtures/work/work-foo",
			input:     "e ",
			uid:       "ghe:testdata/work/work-foo",
			valid:     true,
			title:     "testdata/work/work-foo",
			arg:       "edit " + fixturePath + "/work/work-foo",
			copy:      fixturePath + "/work/work-foo",
			cmdModArg: "term " + fixturePath + "/work/work-foo",
			altModArg: "finder " + fixturePath + "/work/work-foo",
		},
		{
			test:      "edit project includes fixtures/projects/project-bar",
			input:     "e ",
			uid:       "ghe:testdata/projects/project-bar",
			valid:     true,
			title:     "testdata/projects/project-bar",
			arg:       "edit " + fixturePath + "/projects/project-bar",
			cmdModArg: "term " + fixturePath + "/projects/project-bar",
			altModArg: "finder " + fixturePath + "/projects/project-bar",
		},
		{
			test:      "edit project includes symlinked dir in fixtures",
			input:     "e linked",
			uid:       "ghe:testdata/projects/linked",
			valid:     true,
			arg:       "edit " + fixturePath + "/projects/linked",
			cmdModArg: "term " + fixturePath + "/projects/linked",
			altModArg: "finder " + fixturePath + "/projects/linked",
		},
		{
			test:    "edit project does not include symlinked file in fixtures",
			input:   "e linked",
			exclude: "ghe:testdata/projects/linked-file",
		},
		{
			test:  "edit project shows error for invalid directory",
			input: "e foo",
			cfg:   invalidDir,
			title: "Invalid project directory: testdata/nonexistent",
		},
		{
			test:    "edit project excludes files (listing only directories)",
			input:   "e ",
			exclude: "ghe:testdata/work/ignored-file",
		},

		// edit/open/auto filtering
		{
			test:  "edit project with input matches directories",
			input: "e work-foo",
			uid:   "ghe:testdata/work/work-foo",
			valid: true,
		},
		{
			test:    "edit project with input excludes non-matches",
			input:   "e work-foo",
			exclude: "ghe:testdata/projects/project-bar",
		},
		{
			test:  "edit project with input fuzzy-matches directories",
			input: "e wf",
			uid:   "ghe:testdata/work/work-foo",
			valid: true,
		},
		{
			test:    "edit project with input excludes non-fuzzy matches",
			input:   "e wf",
			exclude: "ghe:testdata/projects/project-bar",
		},
	} {
		t.Run(fmt.Sprintf("Complete(%#v): %s", tc.input, tc.test), tc.testItem)
	}
}

// validateItems validates alfred items, checking for UID uniqueness and
// required fields.
func validateItems(t *testing.T, items alfred.Items) {
	uids := map[string]bool{}
	for _, item := range items {
		assert.NotEmpty(t, item.Title, "item.Title in\n%#v", item)
		if item.Valid {
			assert.NotEmpty(t, item.UID, "item.UID in valid\n%#v", item)
			assert.NotEmpty(t, item.Arg, "item.Arg in valid\n%#v", item)
		} else {
			assert.Empty(t, item.UID, "item.UID should not be set unless item is valid\n%#v", item)
		}
		if len(item.UID) > 0 {
			if _, ok := uids[item.UID]; ok {
				t.Errorf("non-unique UID %#v in %s", item.UID, items)
			} else {
				uids[item.UID] = true
			}
		}
		if strings.HasPrefix(item.Arg, "open ") {
			url := item.Arg[5:]
			if assert.NotNil(t, item.Text, "item.Text in\n%#v", item) {
				assert.Equal(t, url, item.Text.Copy, "item.Text.Copy in\n%#v", item)
			}
		}
	}

}

// Try to find item by uid or title
func findMatchingItem(uid, title string, items alfred.Items) (alfred.Item, bool) {
	for _, item := range items {
		if item.Title == title || (len(item.UID) > 0 && item.UID == uid) {
			return item, true
		}
	}
	return alfred.Item{}, false
}

func TestFinalizeResult(t *testing.T) {
	c := completion{
		result: alfred.NewFilterResult(),
	}
	c.finalizeResult()

	// test that Rerun only gets set if a variable's been set
	assert.Zero(t, c.result.Rerun, "c.result.Rerun should not have a value")

	c.retry = true
	c.finalizeResult()
	assert.Equal(t, rerunAfter, c.result.Rerun, "c.result.Rerun in result\n%#v", c.result)
}

func TestFindProjectDirs(t *testing.T) {
	fixturePath, _ := filepath.Abs("testdata/projects")
	dirList, err := findProjectDirs(fixturePath)
	dirs := make(map[string]struct{}, len(dirList))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	for _, d := range dirList {
		dirs[d] = struct{}{}
	}
	assert.Contains(t, dirs, "project-bar", "normal directory in\n%v", dirList)
	assert.Contains(t, dirs, "linked", "symlinked directory in\n%v", dirList)
	assert.NotContains(t, dirs, "linked-file", "symlinked file in\n%v", dirList)
}
