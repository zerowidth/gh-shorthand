package completion

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/pkg/alfred"
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
	ProjectDirs: []string{"../../../test/fixtures/work", "../../../test/fixtures/projects"},
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

var invalidDir = &config.Config{
	ProjectDirs: []string{"../../../test/fixtures/nonexistent"},
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
	cmdMod  string         // cmd modifier action, if applicable
	altMod  string         // alt modifier action, if applicable
}

func (tc *completeTestCase) testItem(t *testing.T) {
	if tc.cfg == nil {
		tc.cfg = defaultCfg
	}
	result := alfred.NewFilterResult()

	env := Environment{
		Query: tc.input,
		Start: time.Now(),
	}
	appendParsedItems(result, *tc.cfg, env)

	validateItems(t, result.Items)

	if len(tc.exclude) > 0 {
		item := findMatchingItem(tc.exclude, tc.exclude, result.Items)
		if item != nil {
			t.Errorf("%s\nexpected no item with UID or Title %q", result.Items, tc.exclude)
		}
		return
	}

	if len(tc.uid) == 0 && len(tc.title) == 0 {
		t.Skip("skipping, uid/title/exclude not specified")
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

		if len(tc.cmdMod) > 0 || len(tc.altMod) > 0 {
			if item.Mods == nil {
				t.Errorf("%+v\nexpected Mods to be have a value", item)
			} else {

				if len(tc.cmdMod) > 0 {
					if item.Mods.Cmd == nil {
						t.Errorf("%+v\nexpected Mods.Cmd to have a value", item)
					} else {
						if tc.cmdMod != item.Mods.Cmd.Arg {
							t.Errorf("%+v\nexpected Mods.Cmd.Arg to be %s, was %+v", item, tc.cmdMod, item.Mods.Cmd.Arg)
						}
					}
				}

				if len(tc.altMod) > 0 {
					if item.Mods.Alt == nil {
						t.Errorf("%+v\nexpected Mods.Alt to have a value", item)
					} else {
						if tc.altMod != item.Mods.Alt.Arg {
							t.Errorf("%+v\nexpected Mods.Alt.Arg to be %s, was %+v", item, tc.altMod, item.Mods.Alt.Arg)
						}
					}
				}
			}
		}

	} else {
		t.Errorf("expected item with uid %q and/or title %q in %s", tc.uid, tc.title, result.Items)
	}
}

func TestCompleteItems(t *testing.T) {
	fixturePath, _ := filepath.Abs("../../../test/fixtures")

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
			title: "List and search issues in a GitHub repository",
			auto:  "i ",
		},
		"empty input shows project list/search default": {
			input: "",
			title: "List and open projects on GitHub repositories or organizations",
			auto:  "p ",
		},
		"empty input shows new issue default": {
			input: "",
			title: "New issue in a GitHub repository",
			auto:  "n ",
		},
		"empty input shows edit project default": {
			input: "",
			title: "Open a project",
			auto:  "e ",
		},
		"empty input shows search issues default": {
			input: "",
			title: "Search issues across GitHub",
			auto:  "s ",
		},
		"a mode char by itself shows the default repo": {
			input: "i",
			uid:   "ghi:zerowidth/default",
			valid: true,
		},
		"a mode char followed by a space shows the default repo": {
			input: "i ",
			uid:   "ghi:zerowidth/default",
			valid: true,
		},
		"a mode char followed by a non-space shows nothing": {
			input:   "ix",
			exclude: "ghi:zerowidth/default",
		},

		// basic parsing tests
		"open a shorthand repo": {
			input:  " df",
			uid:    "gh:zerowidth/dotfiles",
			valid:  true,
			title:  "Open zerowidth/dotfiles (df)",
			arg:    "open https://github.com/zerowidth/dotfiles",
			cmdMod: "paste [zerowidth/dotfiles](https://github.com/zerowidth/dotfiles)",
		},
		"open a shorthand repo and issue": {
			input:  " df 123",
			uid:    "gh:zerowidth/dotfiles#123",
			valid:  true,
			title:  "Open zerowidth/dotfiles#123 (df#123)",
			arg:    "open https://github.com/zerowidth/dotfiles/issues/123",
			altMod: "paste zerowidth/dotfiles#123",
			cmdMod: "paste [zerowidth/dotfiles#123](https://github.com/zerowidth/dotfiles/issues/123)",
		},
		"open a fully qualified repo": {
			input: " foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar",
			arg:   "open https://github.com/foo/bar",
		},
		"open a fully qualified repo and issue": {
			input:  " foo/bar 123",
			uid:    "gh:foo/bar#123",
			valid:  true,
			title:  "Open foo/bar#123",
			arg:    "open https://github.com/foo/bar/issues/123",
			altMod: "paste foo/bar#123",
			cmdMod: "paste [foo/bar#123](https://github.com/foo/bar/issues/123)",
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
		"requires exact prefix for repo shorthand": {
			input: " dfx/foo",
			uid:   "gh:dfx/foo",
			valid: true,
			title: "Open dfx/foo",
		},
		"requires exact prefix for user shorthand": {
			input: " zwx/foo",
			uid:   "gh:zwx/foo",
			valid: true,
			title: "Open zwx/foo",
		},

		// issue index/search
		"open issues index on a shorthand repo": {
			input: "i df",
			uid:   "ghi:zerowidth/dotfiles",
			valid: true,
			title: "List issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
		},
		"open issues index on a repo": {
			input: "i foo/bar",
			uid:   "ghi:foo/bar",
			valid: true,
			title: "List issues for foo/bar",
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
		"search issues globally with no query": {
			input: "s ",
			title: "Search issues for...",
			valid: false,
			auto:  "s ",
		},
		"search issues globally with a query": {
			input: "s foo bar",
			title: "Search issues for foo bar",
			valid: true,
			arg:   "open https://github.com/search?utf8=✓&type=Issues&q=foo%20bar",
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
			title: "List issues for zerowidth/default",
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

		// projects
		"project listing with explicit repo": {
			input: "p zerowidth/dotfiles",
			uid:   "ghp:zerowidth/dotfiles",
			title: "List projects in zerowidth/dotfiles",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects",
		},
		"specific project with explicit repo": {
			input: "p zerowidth/dotfiles 10",
			uid:   "ghp:zerowidth/dotfiles/10",
			title: "Open project #10 in zerowidth/dotfiles",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects/10",
		},
		"project listing with shorthand repo": {
			input: "p df",
			uid:   "ghp:zerowidth/dotfiles",
			title: "List projects in zerowidth/dotfiles (df)",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects",
		},
		"specific project with shorthand repo": {
			input: "p df 10",
			uid:   "ghp:zerowidth/dotfiles/10",
			title: "Open project #10 in zerowidth/dotfiles (df#10)",
			valid: true,
			arg:   "open https://github.com/zerowidth/dotfiles/projects/10",
		},
		"project listing with org": {
			input: "p zerowidth",
			uid:   "ghp:zerowidth",
			title: "List projects for zerowidth",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects",
		},
		"specific project with org": {
			input: "p zerowidth 10",
			uid:   "ghp:zerowidth/10",
			title: "Open project #10 for zerowidth",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects/10",
		},
		"project listing with user shorthand": {
			input: "p zw",
			uid:   "ghp:zerowidth",
			title: "List projects for zerowidth (zw)",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects",
		},
		"specific project with user shorthand": {
			input: "p zw 10",
			uid:   "ghp:zerowidth/10",
			title: "Open project #10 for zerowidth (zw)",
			valid: true,
			arg:   "open https://github.com/orgs/zerowidth/projects/10",
		},
		"specific project with numeric username treated as project": {
			input: "p 123",
			uid:   "ghp:zerowidth/default/123",
			valid: true,
		},
		"specific project with numeric username but default repo treated as user": {
			input: "p 123",
			cfg:   emptyConfig,
			uid:   "ghp:123",
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
			title: "List issues for zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues",
			auto:  "i df",
		},
		"autocompletes issue index with input so far": {
			input: "i foo",
			valid: false,
			title: "List issues for foo...",
			auto:  "i foo",
		},
		"autocomplete issues open-ended when no default": {
			cfg:   emptyConfig,
			input: "i ",
			title: "List issues for ...",
			valid: false,
		},
		"autocomplete user for issues": {
			input: "i z",
			title: "List issues for zerowidth/... (zw)",
			auto:  "i zw/",
		},

		// project autocomplete
		"autocompletes for repo projects": {
			input: "p d",
			uid:   "ghp:zerowidth/dotfiles",
			valid: true,
			title: "List projects in zerowidth/dotfiles (df)",
			auto:  "p df",
		},
		"autocompletes user projects": {
			input: "p z",
			uid:   "ghp:zerowidth",
			valid: true,
			title: "List projects for zerowidth (zw)",
			auto:  "p zw",
		},
		"autocompletes projects with input so far": {
			input: "p foo",
			title: "List projects for foo...",
			valid: false,
		},
		"autocomplete open-ended projects when no default": {
			input: "p ",
			cfg:   emptyConfig,
			title: "List projects for ...",
			valid: false,
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

		"edit project includes fixtures/work/work-foo": {
			input:  "e ",
			uid:    "ghe:../../../test/fixtures/work/work-foo",
			valid:  true,
			title:  "../../../test/fixtures/work/work-foo",
			arg:    "edit " + fixturePath + "/work/work-foo",
			copy:   fixturePath + "/work/work-foo",
			cmdMod: "term " + fixturePath + "/work/work-foo",
			altMod: "finder " + fixturePath + "/work/work-foo",
		},
		"edit project includes fixtures/projects/project-bar": {
			input:  "e ",
			uid:    "ghe:../../../test/fixtures/projects/project-bar",
			valid:  true,
			title:  "../../../test/fixtures/projects/project-bar",
			arg:    "edit " + fixturePath + "/projects/project-bar",
			cmdMod: "term " + fixturePath + "/projects/project-bar",
			altMod: "finder " + fixturePath + "/projects/project-bar",
		},
		"edit project includes symlinked dir in fixtures": {
			input:  "e linked",
			uid:    "ghe:../../../test/fixtures/projects/linked",
			valid:  true,
			arg:    "edit " + fixturePath + "/projects/linked",
			cmdMod: "term " + fixturePath + "/projects/linked",
			altMod: "finder " + fixturePath + "/projects/linked",
		},
		"edit project does not include symlinked file in fixtures": {
			input:   "e linked",
			exclude: "ghe:../../../test/fixtures/projects/linked-file",
		},
		"edit project shows error for invalid directory": {
			input: "e foo",
			cfg:   invalidDir,
			title: "Invalid project directory: ../../../test/fixtures/nonexistent",
		},
		"edit project excludes files (listing only directories)": {
			input:   "e ",
			exclude: "ghe:../../../test/fixtures/work/ignored-file",
		},

		// edit/open/auto filtering
		"edit project with input matches directories": {
			input: "e work-foo",
			uid:   "ghe:../../../test/fixtures/work/work-foo",
			valid: true,
		},
		"edit project with input excludes non-matches": {
			input:   "e work-foo",
			exclude: "ghe:../../../test/fixtures/projects/project-bar",
		},
		"edit project with input fuzzy-matches directories": {
			input: "e wf",
			uid:   "ghe:../../../test/fixtures/work/work-foo",
			valid: true,
		},
		"edit project with input excludes non-fuzzy matches": {
			input:   "e wf",
			exclude: "ghe:../../../test/fixtures/projects/project-bar",
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
				t.Errorf("non-unique UID %#v in %s", item.UID, items)
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
	fixturePath, _ := filepath.Abs("../../../test/fixtures/projects")
	dirList, err := findProjectDirs(fixturePath)
	dirs := make(map[string]struct{}, len(dirList))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
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
