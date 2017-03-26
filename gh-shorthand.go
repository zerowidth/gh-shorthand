package main

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
	"github.com/zerowidth/gh-shorthand/parser"
	"os"
	"strings"
)

func main() {
	var input string
	var items []alfred.Item

	if len(os.Args) < 2 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	path, _ := homedir.Expand("~/.gh-shorthand.yml")
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		items = []alfred.Item{errorItem("when loading ~/.gh-shorthand.yml", err.Error())}
	} else {
		items = generateItems(cfg, input)
	}

	printItems(items)
}

func generateItems(cfg *config.Config, input string) []alfred.Item {
	items := []alfred.Item{}
	result := parser.Parse(cfg.RepoMap, input)
	if result.Repo != "" {
		uid := "gh:" + result.Repo
		title := "Open " + result.Repo
		arg := "open https://github.com/" + result.Repo

		if result.Issue != "" {
			uid += "#" + result.Issue
			title += "#" + result.Issue
			arg += "/issues/" + result.Issue
		}

		if result.Match != "" {
			title += " (" + result.Match + ")"
		}

		items = append(items, alfred.Item{
			UID:   uid,
			Title: title + " on GitHub",
			Arg:   arg,
			Valid: true,
		})
	}
	return items
}

func errorItem(context, msg string) alfred.Item {
	return alfred.Item{
		Title:    fmt.Sprintf("Error %s"),
		Subtitle: msg,
		Valid:    false,
	}
}

func printItems(items []alfred.Item) {
	doc := alfred.Items{Items: items}
	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		panic(err.Error())
	}
}
