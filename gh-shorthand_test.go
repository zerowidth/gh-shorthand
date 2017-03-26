package main

import (
	"fmt"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"testing"
)

var cfg = &config.Config{
	RepoMap: map[string]string{
		"df": "zerowidth/dotfiles",
	},
}

func TestDefaults(t *testing.T) {
	items := generateItems(cfg, "")
	if len(items) > 0 {
		t.Errorf("expected default result to be empty, got %#v", items)
	}
}

type testCase struct {
	input string
	uid   string // the results must contain an entry with this uid
	valid bool   // and with the valid flag set to this
	title string // the expected title
	desc  string // test case description
}

func TestItems(t *testing.T) {
	// Based on input, the resulting items must include one that matches either
	// the given UID or title. All items are also validated for correctness and
	// uniqueness by UID.
	testCases := []testCase{
		{
			input: "df",
			uid:   "gh:zerowidth/dotfiles",
			valid: true,
			title: "Open zerowidth/dotfiles (df) on GitHub",
			desc:  "open a shorthand repo",
		},
		{
			input: "foo/bar",
			uid:   "gh:foo/bar",
			valid: true,
			title: "Open foo/bar on GitHub",
			desc:  "open a fully qualified repo",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("generateItems(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			items := generateItems(cfg, tc.input)

			validateItems(t, tc, items)

			item := findMatchingItem(tc, items)
			if item != nil {
				if tc.uid != "" && item.UID != tc.uid {
					t.Errorf("expected %+v uid to be %q", item, tc.uid)
				}

				if tc.title != "" && item.Title != tc.title {
					t.Errorf("expected %+v title to be %q", item, tc.title)
				}

				if item.Valid != tc.valid {
					t.Errorf("expected %+v valid to be %t", item, tc.valid)
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
func findMatchingItem(tc testCase, items []alfred.Item) *alfred.Item {
	for _, item := range items {
		if item.Title == tc.title || item.UID == tc.uid {
			return &item
		}
	}
	return nil
}
