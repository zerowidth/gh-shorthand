package main

import (
	"fmt"
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

func TestItems(t *testing.T) {
	testCases := []struct {
		input string
		uid   string // the results must contain an entry with this uid
		valid bool   // and with the valid flag set to this
		desc  string // test case description
	}{
		{"df", "gh:zerowidth/dotfiles", true, "open a shorthand repo"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("generateItems(%#v): %s", tc.input, tc.desc), func(t *testing.T) {
			items := generateItems(cfg, tc.input)

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
			}

			if tc.uid != "" {
				for _, item := range items {
					if item.UID == tc.uid && item.Valid == tc.valid {
						return
					}
				}
				t.Errorf("expected %+v to include item with uid %s and valid %t", items, tc.uid, tc.valid)
			}

		})
	}
}
