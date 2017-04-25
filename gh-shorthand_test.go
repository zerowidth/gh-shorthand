package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
)

var cfg = &config.Config{
	DefaultRepo: "zerowidth/default",
	RepoMap: map[string]string{
		"df":  "zerowidth/dotfiles",
		"df2": "zerowidth/df2",
	},
	UserMap: map[string]string{
		"zw": "zerowidth",
	},
	ProjectDirs: []string{"fixtures/work", "fixtures/projects"},
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
		"zw": "zerowidth",
	},
}

var emptyConfig = &config.Config{}

type completeTestCase struct {
	input   string         // input string
	uid     string         // the results must contain an entry with this uid
	valid   bool           // and with the valid flag set to this
	title   string         // the expected title
	arg     string         // expected argument
	auto    string         // expected autocomplete arg
	cfg     *config.Config // config to use instead of the default cfg
	exclude string         // exclude any item with this UID or title
	copy    string         // the clipboard copy string, if applicable
}

func (tc *completeTestCase) testItem(t *testing.T) {
	if tc.cfg == nil {
		tc.cfg = cfg
	}
	env := envVars{}
	result := alfred.NewFilterResult()

	appendParsedItems(result, tc.cfg, env, tc.input)

	validateItems(t, result.Items)

	if len(tc.exclude) > 0 {
		item := findMatchingItem(tc.exclude, tc.exclude, result.Items)
		if item != nil {
			t.Errorf("%+v\nexpected no item with UID or Title %q", result.Items, tc.exclude)
		}
		return
	}

	item := findMatchingItem(tc.uid, tc.title, result.Items)
	if item != nil {
		if len(tc.uid) > 0 && item.UID != tc.uid {
			t.Errorf("%+v\nexpected UID %q to be %q", item, item.UID, tc.uid)
		}

		if len(tc.title) > 0 && item.Title != tc.title {
			t.Errorf("%+v\nexpected Title %q to be %q", item, item.Title, tc.title)
		}

		if item.Valid != tc.valid {
			t.Errorf("%+v\nexpected Valid %t to be %t", item, item.Valid, tc.valid)
		}

		if len(tc.arg) > 0 && item.Arg != tc.arg {
			t.Errorf("%+v\nexpected Arg %q to be %q", item, item.Arg, tc.arg)
		}

		if len(tc.auto) > 0 && item.Autocomplete != tc.auto {
			t.Errorf("%+v\nexpected Autocomplete %q to be %q", item, item.Autocomplete, tc.auto)
		}

		if len(tc.copy) > 0 && (item.Text == nil || item.Text.Copy != tc.copy) {
			t.Errorf("%+v\nexpected Text.Copy %+v to be %q", item, item.Text, tc.copy)
		}
	} else {
		t.Errorf("expected item with uid %q and/or title %q in %+v", tc.uid, tc.title, result.Items)
	}
}

func TestCompleteItems(t *testing.T) {
	fixturePath, _ := filepath.Abs("fixtures")

	// Based on input, the resulting items must include one that matches either
	// the given UID or title. All items are also validated for correctness and
	// uniqueness by UID.
	for desc, tc := range map[string]completeTestCase{

		// defaults
		"empty input shows open repo/issue default": {
			input: "",
			title: "Open repositories and issues on GitHub",
			auto:  " ",
		},
		"empty input shows issue list/search default": {
			input: "",
			title: "List and search issues on GitHub",
			auto:  "i ",
		},
		"empty input shows new issue default": {
			input: "",
			title: "New issue on GitHub",
			auto:  "n ",
		},
		"empty input shows commit default": {
			input: "",
			title: "Find a commit in a GitHub repository",
			auto:  "c ",
		},
		"empty input shows markdown link default": {
			input: "",
			title: "Insert Markdown link to a GitHub repository or issue",
			auto:  "m ",
		},
		"empty input shows issue reference default": {
			input: "",
			title: "Insert issue reference shorthand for a GitHub repository or issue",
			auto:  "r ",
		},
		"empty input shows edit project default": {
			input: "",
			title: "Edit a project",
			auto:  "e ",
		},
		"empty input shows open finder default": {
			input: "",
			title: "Open a project directory in Finder",
			auto:  "o ",
		},
		"empty input shows open terminal default": {
			input: "",
			title: "Open terminal in a project",
			auto:  "t ",
		},
		"a mode char by itself shows the default repo": {
			input: "m",
			uid:   "ghm:zerowidth/default",
			valid: true,
		},
		"a mode char followed by a space shows the default repo": {
			input: "m ",
			uid:   "ghm:zerowidth/default",
			valid: true,
		},
		"a mode char followed by a non-space shows nothing": {
			input:   "mx",
			exclude: "ghm:zerowidth/default",
		},

		// basic parsing tests
		"open a shorthand repo": {
			input: " df",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles",
		},
		"open a shorthand repo and issue": {
			input: " df 123",
			uid:   "gh:zerowidth/dotfiles#123",
			valid: true,
			title: "Open zerowidth/dotfiles#123 (df#123)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/123",
		},
		"open a fully qualified repo": {
			input: " foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar",
			arg:   "open https://github.com/foo/bar",
		},
		"open a fully qualified repo and issue": {
			input: " foo/bar 123",
			uid:   "gh:foo/bar#123",
			valid: true,
			title: "Open foo/bar#123",
			arg:   "open https://github.com/foo/bar/issues/123",
		},
		"open a shorthand user with repo": {
			input: " zw/foo",
			uid:   "gh:zerowidth/foo",
			valid: true,
			title: "Open zerowidth/foo (zw)",
			arg:   "open https://github.com/zerowidth/foo",
		},
		"open a shorthand user with repo and issue": {
			input: " zw/foo 123",
			uid:   "gh:zerowidth/foo#123",
			valid: true,
			title: "Open zerowidth/foo#123 (zw)",
			arg:   "open https://github.com/zerowidth/foo/issues/123",
		},
		"no match if any unparsed query remains after shorthand": {
			input:   " df foo",
			exclude: "gh:zerowidth/dotfiles",
		},
		"no match if any unparsed query remains after repo": {
			input:   " foo/bar baz",
			exclude: "gh:foo/bar",
		},
		"ignores trailing whitespace for shorthand": {
			input: " df ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
		},
		"ignores trailing whitespace for repo": {
			input: " foo/bar ",
			uid:   "gh:foo/bar",
			valid: true,
		},
		"open path on matched shorthand repo": {
			input: " df /foo",
			uid:   "gh:zerowidth/dotfiles/foo",
			valid: true,
			title: "Open zerowidth/dotfiles/foo (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/foo",
		},
		"open direct path when not prefixed with repo": {
			input: " /foo",
			uid:   "gh:/foo",
			valid: true,
			title: "Open /foo",
			arg:   "open https://github.com/foo",
		},
		"don't open direct path when matching user prefix": {
			input:   " zw/",
			exclude: "gh:/",
		},
		"prefer repo shorthand to user prefix": {
			input: " zw/foo",
			cfg:   userRepoCollision,
			uid:   "gh:zerowidth/dotfiles/foo",
			valid: true,
		},

		// issue index/search
		"open issues index on a shorthand repo": {
			input: "i df",
			uid:   "ghi:zerowidth/dotfiles",
			valid: true,
			title: "Open issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
		},
		"open issues index on a repo": {
			input: "i foo/bar",
			uid:   "ghi:foo/bar",
			valid: true,
			title: "Open issues for foo/bar",
			arg:   "open https://github.com/foo/bar/issues",
		},
		"search issues on a repo": {
			input: "i a/b foo bar",
			uid:   "ghis:a/b",
			valid: true,
			title: "Search issues in a/b for foo bar",
			arg:   "open https://github.com/a/b/search?utf8=✓&type=Issues&q=foo%20bar",
		},
		"search issues on a shorhthand repo": {
			input: "i df foo bar",
			uid:   "ghis:zerowidth/dotfiles",
			valid: true,
			title: "Search issues in zerowidth/dotfiles (df) for foo bar",
			arg:   "open https://github.com/zerowidth/dotfiles/search?utf8=✓&type=Issues&q=foo%20bar",
		},
		"search issues for a numeric string on a repo": {
			input: "i a/b 12345",
			uid:   "ghis:a/b",
			valid: true,
			title: "Search issues in a/b for 12345",
		},

		// new issue
		"open a new issue in a shorthand repo": {
			input: "n df",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new",
		},
		"open a new issue in a repo": {
			input: "n a/b",
			uid:   "ghn:a/b",
			valid: true,
			title: "New issue in a/b",
			arg:   "open https://github.com/a/b/issues/new",
		},
		"open a new issue with a query": {
			input: "n df foo bar",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df): foo bar",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new?title=foo%20bar",
		},
		"search for a commit in a repo": {
			input: "c df deadbeef",
			uid:   "ghc:zerowidth/dotfiles",
			valid: true,
			title: "Find commit in zerowidth/dotfiles (df) with SHA1 deadbeef",
			arg:   "open https://github.com/zerowidth/dotfiles/search?utf8=✓&type=Issues&q=deadbeef",
		},
		"search for a commit in a repo with no query yet": {
			input: "c df",
			valid: false,
			title: "Find commit in zerowidth/dotfiles (df) with SHA1...",
		},
		"search for a commit in a repo with partial query": {
			input: "c df abcde",
			valid: false,
			title: "Find commit in zerowidth/dotfiles (df) with SHA1 abcde...",
		},
		"search for a commit with a numeric SHA1": {
			input: "c df 1234567",
			valid: true,
			uid:   "ghc:zerowidth/dotfiles",
			title: "Find commit in zerowidth/dotfiles (df) with SHA1 1234567",
		},
		"markdown link with a repo": {
			input: "m foo/bar",
			uid:   "ghm:foo/bar",
			valid: true,
			title: "Insert Markdown link to foo/bar",
			arg:   "paste [foo/bar](https://github.com/foo/bar)",
			copy:  "[foo/bar](https://github.com/foo/bar)",
		},
		"markdown link with a repo and issue": {
			input: "m foo/bar 123",
			uid:   "ghm:foo/bar#123",
			title: "Insert Markdown link to foo/bar#123",
			valid: true,
			arg:   "paste [foo/bar#123](https://github.com/foo/bar/issues/123)",
			copy:  "[foo/bar#123](https://github.com/foo/bar/issues/123)",
		},
		"markdown link with shorthand repo and issue": {
			input: "m df 123",
			uid:   "ghm:zerowidth/dotfiles#123",
			title: "Insert Markdown link to zerowidth/dotfiles#123 (df#123)",
			valid: true,
			arg:   "paste [zerowidth/dotfiles#123](https://github.com/zerowidth/dotfiles/issues/123)",
		},
		"issue reference with a repo and issue": {
			input: "r foo/bar 123",
			uid:   "ghr:foo/bar#123",
			valid: true,
			title: "Insert issue reference to foo/bar#123",
			arg:   "paste foo/bar#123",
			copy:  "foo/bar#123",
		},
		"issue reference with shorthand repo and issue": {
			input: "r df 123",
			uid:   "ghr:zerowidth/dotfiles#123",
			valid: true,
			title: "Insert issue reference to zerowidth/dotfiles#123 (df#123)",
			arg:   "paste zerowidth/dotfiles#123",
		},
		"issue references with no issue has no valid item": {
			input:   "r df",
			exclude: "ghr:zerowidth/dotfiles",
		},
		"issue references with repo has autocomplete value": {
			input: "r df",
			title: "Insert issue reference to zerowidth/dotfiles#... (df)",
			auto:  "r df ",
		},
		"issue reference with user-shorthand repo has autocomplete value": {
			input: "r zw/foo",
			title: "Insert issue reference to zerowidth/foo#... (zw)",
			auto:  "r zw/foo ",
		},

		// default repo
		"open an issue with the default repo": {
			input: " 123",
			uid:   "gh:zerowidth/default#123",
			valid: true,
			title: "Open zerowidth/default#123",
			arg:   "open https://github.com/zerowidth/default/issues/123",
		},
		"open the default repo when default is also in map": {
			cfg:   defaultInMap,
			input: " ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles",
			arg:   "open https://github.com/zerowidth/dotfiles",
		},
		"includes no default if remaining input isn't otherwise valid": {
			input:   " foo",
			exclude: "gh:zerowidth/default",
		},
		"does not use default repo with path alone": {
			input:   " /foo",
			exclude: "gh:zerowidth/default/foo",
		},
		"show issues for a default repo": {
			input: "i ",
			uid:   "ghi:zerowidth/default",
			valid: true,
			title: "Open issues for zerowidth/default",
		},
		"search issues with a query in the default repo": {
			input: "i foo",
			uid:   "ghis:zerowidth/default",
			valid: true,
			title: "Search issues in zerowidth/default for foo",
		},
		"new issue in the default repo": {
			input: "n ",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default",
		},
		"new issue with a title in the default repo": {
			input: "n foo",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default: foo",
		},
		"markdown link with default repo": {
			input: "m ",
			uid:   "ghm:zerowidth/default",
			valid: true,
			title: "Insert Markdown link to zerowidth/default",
		},
		"markdown link with default repo and issue": {
			input: "m 123",
			uid:   "ghm:zerowidth/default#123",
			valid: true,
			title: "Insert Markdown link to zerowidth/default#123",
		},
		"issue reference with no issue, using default repo": {
			input: "r ",
			title: "Insert issue reference to zerowidth/default#...",
			valid: false,
		},
		"issue reference with issue, using default repo": {
			input: "r 123",
			uid:   "ghr:zerowidth/default#123",
			title: "Insert issue reference to zerowidth/default#123",
			valid: true,
		},

		// repo autocomplete
		"no autocomplete for empty input": {
			input:   " ",
			exclude: "gh:zerowidth/dotfiles",
		},
		"autocomplete 'd', first match": {
			input: " d",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles",
			auto:  " df",
		},
		"autocomplete 'd', second match": {
			input: " d",
			uid:   "gh:zerowidth/df2",
			valid: true,
			title: "Open zerowidth/df2 (df2)",
			arg:   "open https://github.com/zerowidth/df2",
			auto:  " df2",
		},
		"autocomplete 'z', matching user shorthand": {
			input: " z",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		"autocomplete when user shorthand matches exactly": {
			input: " zw",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		"autocomplete when user shorthand has trailing slash": {
			input: " zw/",
			title: "Open zerowidth/... (zw)",
			valid: false,
			auto:  " zw/",
		},
		"no autocomplete when user shorthand has text following the slash": {
			input:   " zw/foo",
			exclude: "Open zerowidth/... (zw)",
		},
		"autocomplete 'd', open-ended": {
			input: " d",
			title: "Open d...",
			valid: false,
		},
		"autocomplete open-ended when no default": {
			cfg:   emptyConfig,
			input: " ",
			title: "Open ...",
			valid: false,
		},
		"autocomplete unmatched user prefix": {
			input: " foo/",
			title: "Open foo/...",
			valid: false,
		},
		"does not autocomplete with fully-qualified repo": {
			input:   " foo/bar",
			exclude: "Open foo/bar...",
		},
		"no autocomplete when input has space": {
			input:   " foo bar",
			exclude: "Open foo bar...",
		},

		// issue index autocomplete
		"autocompletes for issue index": {
			input: "i d",
			uid:   "ghi:zerowidth/dotfiles",
			valid: true,
			title: "Open issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
			auto:  "i df",
		},
		"autocompletes issue index with input so far": {
			input: "i foo",
			valid: false,
			title: "Open issues for foo...",
			auto:  "i foo",
		},
		"autocomplete issues open-ended when no default": {
			cfg:   emptyConfig,
			input: "i ",
			title: "Open issues for ...",
			valid: false,
		},
		"autocomplete user for issues": {
			input: "i z",
			title: "Open issues for zerowidth/... (zw)",
			auto:  "i zw/",
		},

		// new issue autocomplete
		"autocompletes for new issue": {
			input: "n d",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new",
			auto:  "n df",
		},
		"autocomplete user for new issue": {
			input: "n z",
			title: "New issue in zerowidth/... (zw)",
			auto:  "n zw/",
		},
		"autocompletes new issue with input so far": {
			input: "n foo",
			valid: false,
			title: "New issue in foo...",
			auto:  "n foo",
		},
		"autocomplete new issue open-ended when no default": {
			cfg:   emptyConfig,
			input: "n ",
			title: "New issue in ...",
			valid: false,
		},

		// commit search autocomplete
		"autocompletes commit search": {
			input: "c d",
			valid: false,
			title: "Find commit in zerowidth/dotfiles (df) with SHA1...",
			auto:  "c df ",
		},
		"autocompletes commit with open-ended": {
			input: "c d",
			valid: false,
			title: "Find commit in d...",
			auto:  "c d",
		},
		"autocomplete commit open-ended when no default": {
			cfg:   emptyConfig,
			input: "c ",
			title: "Find commit in ...",
			valid: false,
		},

		"edit project includes fixtures/work/work-foo": {
			input: "e ",
			uid:   "ghe:fixtures/work/work-foo",
			valid: true,
			title: "Edit fixtures/work/work-foo",
			arg:   "edit " + fixturePath + "/work/work-foo",
			copy:  fixturePath + "/work/work-foo",
		},
		"edit project includes fixtures/projects/project-bar": {
			input: "e ",
			uid:   "ghe:fixtures/projects/project-bar",
			valid: true,
			title: "Edit fixtures/projects/project-bar",
			arg:   "edit " + fixturePath + "/projects/project-bar",
		},
		"edit project includes symlinked dir in fixtures": {
			input: "e linked",
			uid:   "ghe:fixtures/projects/linked",
			valid: true,
			arg:   "edit " + fixturePath + "/projects/linked",
		},
		"edit project does not include symlinked file in fixtures": {
			input:   "e linked",
			exclude: "ghe:fixtures/projects/linked-file",
		},
		"open finder includes fixtures/work/work-foo": {
			input: "o ",
			uid:   "gho:fixtures/work/work-foo",
			valid: true,
			title: "Open Finder in fixtures/work/work-foo",
			arg:   "finder " + fixturePath + "/work/work-foo",
			copy:  fixturePath + "/work/work-foo",
		},
		"open finder includes fixtures/projects/project-bar": {
			input: "o ",
			uid:   "gho:fixtures/projects/project-bar",
			valid: true,
			title: "Open Finder in fixtures/projects/project-bar",
			arg:   "finder " + fixturePath + "/projects/project-bar",
		},
		"open terminal includes fixtures/work/work-foo": {
			input: "t ",
			uid:   "ght:fixtures/work/work-foo",
			valid: true,
			title: "Open terminal in fixtures/work/work-foo",
			arg:   "term " + fixturePath + "/work/work-foo",
			copy:  fixturePath + "/work/work-foo",
		},
		"open terminal includes fixtures/projects/project-bar": {
			input: "t ",
			uid:   "ght:fixtures/projects/project-bar",
			valid: true,
			title: "Open terminal in fixtures/projects/project-bar",
			arg:   "term " + fixturePath + "/projects/project-bar",
		},
		"edit project excludes files (listing only directories)": {
			input:   "e ",
			exclude: "ghe:fixtures/work/ignored-file",
		},

		// issue reference / markdown link autocomplete
		"autocompletes for markdown links": {
			input: "m d",
			uid:   "ghm:zerowidth/dotfiles",
			title: "Insert Markdown link to zerowidth/dotfiles (df)",
			valid: true,
			arg:   "paste [zerowidth/dotfiles](https://github.com/zerowidth/dotfiles)",
		},
		"autocomplete user for markdown link": {
			input: "m z",
			title: "Insert Markdown link to zerowidth/... (zw)",
			auto:  "m zw/",
		},
		"autocompletes for issue references with shorthand": {
			input: "r d",
			valid: false,
			title: "Insert issue reference to zerowidth/dotfiles#... (df#...)",
			auto:  "r df ",
		},
		"autocompletes for issue references with repos alone": {
			input: "r foo/bar",
			valid: false,
			title: "Insert issue reference to foo/bar#...",
			auto:  "r foo/bar ",
		},
		"autocomplete user for issue reference": {
			input: "r z",
			title: "Insert issue reference to zerowidth/... (zw)",
			auto:  "r zw/",
		},
		"autocompletes for issue references incomplete input": {
			input: "r foo",
			valid: false,
			title: "Insert issue reference to foo...",
			auto:  "r foo",
		},
		"does not autocomplete with open-ended when a repo is present": {
			input:   "r foo/bar ",
			exclude: "Insert issue reference to foo/bar...",
		},

		// edit/open/auto filtering
		"edit project with input matches directories": {
			input: "e work-foo",
			uid:   "ghe:fixtures/work/work-foo",
			valid: true,
		},
		"edit project with input excludes non-matches": {
			input:   "e work-foo",
			exclude: "ghe:fixtures/projects/project-bar",
		},
		"edit project with input fuzzy-matches directories": {
			input: "e wf",
			uid:   "ghe:fixtures/work/work-foo",
			valid: true,
		},
		"edit project with input excludes non-fuzzy matches": {
			input:   "e wf",
			exclude: "ghe:fixtures/projects/project-bar",
		},
	} {
		t.Run(fmt.Sprintf("appendParsedItems(%#v): %s", tc.input, desc), tc.testItem)
	}
}

// validateItems validates alfred items, checking for UID uniqueness and
// required fields.
func validateItems(t *testing.T, items alfred.Items) {
	uids := map[string]bool{}
	for _, item := range items {
		if len(item.Title) == 0 {
			t.Errorf("%+v is missing a title", item)
		}
		if item.Valid {
			if len(item.UID) == 0 {
				t.Errorf("%+v is valid but missing its uid", item)
			}
			if len(item.Arg) == 0 {
				t.Errorf("%+v is valid but missing its arg", item)
			}
		} else if len(item.UID) > 0 {
			t.Errorf("%+v is not valid but has a UID", item)
		}
		if len(item.UID) > 0 {
			if _, ok := uids[item.UID]; ok {
				t.Errorf("non-unique UID %#v in %+v", item.UID, items)
			} else {
				uids[item.UID] = true
			}
		}
		if len(item.Arg) > 5 && strings.HasPrefix(item.Arg, "open ") {
			url := item.Arg[5:]
			if item.Text == nil || item.Text.Copy != url {
				t.Errorf("expected item text to have url %s in %+v", url, item.Text)
			}
		}
	}

}

// Try to find item by uid or title
func findMatchingItem(uid, title string, items alfred.Items) *alfred.Item {
	for _, item := range items {
		if item.Title == title || (len(item.UID) > 0 && item.UID == uid) {
			return item
		}
	}
	return nil
}

func TestFinalizeResult(t *testing.T) {
	result := alfred.NewFilterResult()
	result.AppendItems(
		&alfred.Item{Title: "bother, invalid", Valid: false},
		&alfred.Item{Title: "valid", Valid: true},
		&alfred.Item{Title: "also valid", Valid: true},
		&alfred.Item{Title: "an invalid item", Valid: false},
	)
	finalizeResult(result)

	// test that Rerun only gets set if a variable's been set
	if result.Rerun != 0 {
		t.Errorf("expected result %#v to not have a Rerun value", result)
	}
	result = alfred.NewFilterResult()
	result.SetVariable("foo", "bar")
	finalizeResult(result)
	if result.Rerun != rerunAfter {
		t.Errorf("expected result %#v to have Rerun of %f", result, rerunAfter)
	}
}

func TestFindProjectDirs(t *testing.T) {
	fixturePath, _ := filepath.Abs("fixtures/projects")
	dirList := findProjectDirs(fixturePath)
	dirs := make(map[string]struct{}, len(dirList))
	for _, d := range dirList {
		dirs[d] = struct{}{}
	}

	if _, ok := dirs["project-bar"]; !ok {
		t.Errorf("expected normal directory to be found in %+v", dirList)
	}

	if _, ok := dirs["linked"]; !ok {
		t.Errorf("expected symlinked directory to be found in %+v", dirList)
	}

	if _, ok := dirs["linked-file"]; ok {
		t.Errorf("did not expect symlinked file to be found in %+v", dirList)
	}
}
