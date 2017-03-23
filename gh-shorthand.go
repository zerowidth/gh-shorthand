package main

import (
	"encoding/json"
	"fmt"
	"github.com/zerowidth/gh-shorthand/alfred"
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
