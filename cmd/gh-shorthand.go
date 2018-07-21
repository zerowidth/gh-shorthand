package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/zerowidth/gh-shorthand/internal/pkg/completion"
)

func main() {
	var input string

	if len(os.Args) == 1 {
		input = ""
	} else {
		input = strings.Join(os.Args[1:], " ")
	}

	result := completion.Complete(input)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err.Error())
	}
}
