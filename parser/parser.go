// shorthand parser
package parser

import (
	"regexp"
)

type Result struct {
	Repo string
}

func Parse(repoMap map[string]string, input string) Result {
	repo, exists := repoMap[input]
	if exists {
		return Result{repo}
	}
	re, _ := regexp.Compile(`^[A-Za-z0-9][-A-Za-z0-9]*/[\w\.\-]+$`) // user/repo
	if re.MatchString(input) {
		return Result{input}
	}
	return Result{}
}
