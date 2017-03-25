package main

import (
	"encoding/json"
	"fmt"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
	"os"
	"strings"
)

func main() {
	var input string
	if len(os.Args) < 2 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	fmt.Fprintf(os.Stderr, "input: %#v\n", input)

	item := alfred.Item{
		Title: "hello",
		Valid: false,
	}

	items := alfred.Items{Items: []alfred.Item{item}}
	if err := json.NewEncoder(os.Stdout).Encode(items); err != nil {
		panic(err.Error())
	}
}

func generateItems(cfg *config.Config, input string) (items []alfred.Item) {
	result := parser.Parse(cfg.RepoMap, input)
	if result.Repo != "" {
		items = append(items, alfred.Item{
			UID:   fmt.Sprintf("gh:%s", result.Repo),
			Title: fmt.Sprintf("Open %s on GitHub", result.Repo),
			Arg:   fmt.Sprintf("open https://github.com/%s", result.Repo),
			Valid: true,
		})
	}
	return items
}
