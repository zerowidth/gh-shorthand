package cmd

import (
	"fmt"
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
}

var defaultInMap = &config.Config{
	DefaultRepo: "zerowidth/dotfiles",
	RepoMap: map[string]string{
		"df": "zerowidth/dotfiles",
	},
}

var emptyConfig = &config.Config{}

func TestDefaults(t *testing.T) {
	items := completeItems(cfg, "")
	if len(items) > 0 {
		t.Errorf("expected default result to be empty, got %#v", items)
	}
}

type completeTestCase struct {
	input   string         // input string
	uid     string         // the results must contain an entry with this uid
	valid   bool           // and with the valid flag set to this
	title   string         // the expected title
	arg     string         // expected argument
	auto    string         // expected autocomplete arg
	cfg     *config.Config // config to use instead of the default cfg
	exclude string         // exclude any item with this UID or title
}

func (tc *completeTestCase) testItem(t *testing.T) {
	if tc.cfg == nil {
		tc.cfg = cfg
	}
	items := completeItems(tc.cfg, tc.input)

	validateItems(t, items)

	if tc.exclude != "" {
		item := findMatchingItem(tc.exclude, tc.exclude, items)
		if item != nil {
			t.Errorf("%+v\nexpected no item with UID or Title %q", items, tc.exclude)
		}
		return
	}

	item := findMatchingItem(tc.uid, tc.title, items)
	if item != nil {
		if tc.uid != "" && item.UID != tc.uid {
			t.Errorf("%+v\nexpected UID %q to be %q", item, item.UID, tc.uid)
		}

		if tc.title != "" && item.Title != tc.title {
			t.Errorf("%+v\nexpected Title %q to be %q", item, item.Title, tc.title)
		}

		if item.Valid != tc.valid {
			t.Errorf("%+v\nexpected Valid %t to be %t", item, item.Valid, tc.valid)
		}

		if tc.arg != "" && item.Arg != tc.arg {
			t.Errorf("%+v\nexpected Arg %q to be %q", item, item.Arg, tc.arg)
		}

		if tc.auto != "" && item.Autocomplete != tc.auto {
			t.Errorf("%+v\nexpected Autocomplete %q to be %q", item, item.Autocomplete, tc.auto)
		}
	} else {
		t.Errorf("expected item with uid %q and/or title %q in %+v", tc.uid, tc.title, items)
	}
}

func TestCompleteItems(t *testing.T) {
	// Based on input, the resulting items must include one that matches either
	// the given UID or title. All items are also validated for correctness and
	// uniqueness by UID.
	// rm thixs var
	for desc, tc := range map[string]completeTestCase{
		// basic parsing tests
		"open a shorthand repo": {
			input: " df",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles",
		},
		"open a shorthand repo and issue": {
			input: " df 123",
			uid:   "gh:zerowidth/dotfiles#123",
			valid: true,
			title: "Open zerowidth/dotfiles#123 (df#123) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/123",
		},
		"open a fully qualified repo": {
			input: " foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar on GitHub",
			arg:   "open https://github.com/foo/bar",
		},
		"open a fully qualified repo and issue": {
			input: " foo/bar 123",
			uid:   "gh:foo/bar#123",
			valid: true,
			title: "Open foo/bar#123 on GitHub",
			arg:   "open https://github.com/foo/bar/issues/123",
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
			title: "Open zerowidth/dotfiles/foo (df) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles/foo",
		},
		"open direct path when not prefixed with repo": {
			input: " /foo",
			uid:   "gh:/foo",
			valid: true,
			title: "Open /foo on GitHub",
			arg:   "open https://github.com/foo",
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

		// default repo
		"open an issue with the default repo": {
			input: " 123",
			uid:   "gh:zerowidth/default#123",
			valid: true,
			title: "Open zerowidth/default#123 (default repo) on GitHub",
			arg:   "open https://github.com/zerowidth/default/issues/123",
		},
		"open the default repo when default is also in map": {
			cfg:   defaultInMap,
			input: " ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (default repo) on GitHub",
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
			title: "Open issues for zerowidth/default (default repo)",
		},
		"search issues with a query in the default repo": {
			input: "i foo",
			uid:   "ghis:zerowidth/default",
			valid: true,
			title: "Search issues in zerowidth/default (default repo) for foo",
		},
		"new issue in the default repo": {
			input: "n ",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default (default repo)",
		},
		"new issue with a title in the default repo": {
			input: "n foo",
			uid:   "ghn:zerowidth/default",
			valid: true,
			title: "New issue in zerowidth/default (default repo): foo",
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
			title: "Open zerowidth/dotfiles (df) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles",
			auto:  " df",
		},
		"autocomplete 'd', second match": {
			input: " d",
			uid:   "gh:zerowidth/df2",
			valid: true,
			title: "Open zerowidth/df2 (df2) on GitHub",
			arg:   "open https://github.com/zerowidth/df2",
			auto:  " df2",
		},
		"autocomplete 'd', open-ended": {
			input: " d",
			title: "Open d... on GitHub",
			valid: false,
		},
		"autocomplete unmatched user prefix": {
			input: " foo/",
			title: "Open foo/... on GitHub",
			valid: false,
		},
		"does not autocomplete with fully-qualified repo": {
			input:   " foo/bar",
			exclude: "Open foo/bar... on GitHub",
		},
		"no autocomplete when input has space": {
			input:   " foo bar",
			exclude: "Open foo bar... on GitHub",
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

		// new issue autocomplete
		"autocompletes for new issue": {
			input: "n d",
			uid:   "ghn:zerowidth/dotfiles",
			valid: true,
			title: "New issue in zerowidth/dotfiles (df)",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/new",
			auto:  "n df",
		},
		"autocompletes new issue with input so far": {
			input: "n foo",
			valid: false,
			title: "New issue in foo...",
			auto:  "n foo",
		},
	} {
		t.Run(fmt.Sprintf("generateItems(%#v): %s", tc.input, desc), tc.testItem)
	}
}

func validateItems(t *testing.T, items []*alfred.Item) {
	uids := map[string]bool{}
	for _, item := range items {
		if item.Title == "" {
			t.Errorf("%+v is missing a title", item)
		}
		if item.Valid {
			if item.UID == "" {
				t.Errorf("%+v is valid but missing its uid", item)
			}
			if item.Arg == "" {
				t.Errorf("%+v is valid but missing its arg", item)
			}
		}
		if item.UID != "" {
			if _, ok := uids[item.UID]; ok {
				t.Errorf("non-unique UID %#v in %+v", item.UID, items)
			} else {
				uids[item.UID] = true
			}
		}
	}

}

// Try to find item by uid or title
func findMatchingItem(uid, title string, items []*alfred.Item) *alfred.Item {
	for _, item := range items {
		if item.Title == title || (item.UID != "" && item.UID == uid) {
			return item
		}
	}
	return nil
}
