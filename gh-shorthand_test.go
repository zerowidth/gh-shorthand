package main

import (
	"fmt"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"testing"
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
	items := generateItems(cfg, "")
	if len(items) > 0 {
		t.Errorf("expected default result to be empty, got %#v", items)
	}
}

type testCase struct {
	desc    string         // test case description
	input   string         // input string
	uid     string         // the results must contain an entry with this uid
	valid   bool           // and with the valid flag set to this
	title   string         // the expected title
	arg     string         // expected argument
	auto    string         // expected autocomplete arg
	cfg     *config.Config // config to use instead of the default cfg
	exclude string         // exclude any item with this UID or title
}

func TestItems(t *testing.T) {
	// Based on input, the resulting items must include one that matches either
	// the given UID or title. All items are also validated for correctness and
	// uniqueness by UID.
	testCases := []testCase{
		// basic parsing tests
		{
			desc:  "open a shorthand repo",
			input: " df",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles",
		},
		{
			desc:  "open a shorthand repo and issue",
			input: " df 123",
			uid:   "gh:zerowidth/dotfiles#123",
			valid: true,
			title: "Open zerowidth/dotfiles#123 (df#123) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles/issues/123",
		},
		{
			desc:  "open a fully qualified repo",
			input: " foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar on GitHub",
			arg:   "open https://github.com/foo/bar",
		},
		{
			desc:  "open a fully qualified repo and issue",
			input: " foo/bar 123",
			uid:   "gh:foo/bar#123",
			valid: true,
			title: "Open foo/bar#123 on GitHub",
			arg:   "open https://github.com/foo/bar/issues/123",
		},

		// default repo
		{
			desc:  "open an issue with the default repo",
			input: " 123",
			uid:   "gh:zerowidth/default#123",
			valid: true,
			title: "Open zerowidth/default#123 (default repo) on GitHub",
			arg:   "open https://github.com/zerowidth/default/issues/123",
		},
		{
			// validate uniqueness of UIDs when colliding with autocomplete
			desc:  "open the default repo when default is also in map",
			cfg:   defaultInMap,
			input: " ",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (default repo) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles",
		},
		{
			desc:    "includes no default if remaining input isn't otherwise valid",
			input:   " foo",
			exclude: "gh:zerowidth/default",
		},

		// repo autocomplete
		{
			desc:  "autocomplete 'd', first match",
			input: " d",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df) on GitHub",
			arg:   "open https://github.com/zerowidth/dotfiles",
			auto:  " df",
		},
		{
			desc:  "autocomplete 'd', second match",
			input: " d",
			uid:   "gh:zerowidth/df2",
			valid: true,
			title: "Open zerowidth/df2 (df2) on GitHub",
			arg:   "open https://github.com/zerowidth/df2",
			auto:  " df2",
		},
		{
			desc:  "autocomplete 'd', open-ended",
			input: " d",
			title: "Open d... on GitHub",
			valid: false,
		},
		{
			desc:  "autocomplete unmatched user prefix",
			input: " foo/",
			title: "Open foo/... on GitHub",
			valid: false,
		},
		{
			desc:  "does not autocomplete with fully-qualified repo",
			input: " foo/bar",
			exclude: "Open foo/bar... on GitHub",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("generateItems(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			if tc.cfg == nil {
				tc.cfg = cfg
			}
			items := generateItems(tc.cfg, tc.input)

			validateItems(t, tc, items)

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

		})
	}
}

func validateItems(t *testing.T, tc testCase, items []alfred.Item) {
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
			_, exists := uids[item.UID]
			if exists {
				t.Errorf("non-unique UID %#v in %+v", item.UID, items)
			} else {
				uids[item.UID] = true
			}
		}
	}

}

// Try to find item by uid or title
func findMatchingItem(uid, title string, items []alfred.Item) *alfred.Item {
	for _, item := range items {
		if item.Title == title || (item.UID != "" && item.UID == uid) {
			return &item
		}
	}
	return nil
}
