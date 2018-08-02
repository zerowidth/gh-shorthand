package snippets

import (
	"regexp"
)

var issueRegex = regexp.MustCompile(`(https://github\.com/([^/]+)/([^/]+)/(issues|pull)/(\d+))#?`)
var discussionRegex = regexp.MustCompile(`(https://github\.com/orgs/([^/]+)/teams/([^/]+)/discussions/(\d+))#?`)

// MarkdownLink looks for a github issue or PR URL and converts it to a markdown link.
//
// "https://github.com/zerowidth/camper_van/issues/1" becomes a markdown link
// with link text "zerowidth/camper_van#1".
func MarkdownLink(input string) string {
	issueMatches := issueRegex.FindStringSubmatchIndex(input)
	discussionMatches := discussionRegex.FindStringSubmatchIndex(input)
	if issueMatches == nil {
		if discussionMatches == nil {
			return input
		}
		template := "[@$2/$3#$4]($1)"
		result := []byte{}
		result = discussionRegex.ExpandString(result, template, input, discussionMatches)
		return string(result)
	}

	template := "[$2/$3#$5]($1)"
	result := []byte{}
	result = issueRegex.ExpandString(result, template, input, issueMatches)
	return string(result)
}

// IssueReference looks for a github issue and converts it to an issue reference.
//
// "https://github.com/zerowidth/camper_van/issues/1" becomes
// "zerowidth/camper_van#1".
func IssueReference(input string) string {
	matches := issueRegex.FindStringSubmatchIndex(input)
	if matches == nil {
		return input
	}

	template := "$2/$3#$5"
	result := []byte{}
	result = issueRegex.ExpandString(result, template, input, matches)
	return string(result)
}
