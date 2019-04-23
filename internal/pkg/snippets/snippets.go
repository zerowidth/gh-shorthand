package snippets

import (
	"fmt"
	"regexp"
	"time"

	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

var issueRegex = regexp.MustCompile(`(https://github\.com/([^/]+)/([^/]+)/(issues|pull)/(\d+))#?`)
var discussionRegex = regexp.MustCompile(`(https://github\.com/orgs/([^/]+)/teams/([^/]+)/discussions/(\d+))#?`)

// MarkdownLink looks for a github issue or PR URL and converts it to a markdown link.
//
// "https://github.com/zerowidth/camper_van/issues/1" becomes a markdown link
// with link text "zerowidth/camper_van#1".
func MarkdownLink(cfg config.Config, input string, includeDesc bool) string {
	issueMatches := issueRegex.FindStringSubmatch(input)
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

	repo := fmt.Sprintf("%s/%s", issueMatches[2], issueMatches[3])
	issue := issueMatches[5]
	mdLink := fmt.Sprintf("[%s#%s](%s)", repo, issue, issueMatches[1])

	if includeDesc {
		rpcc := make(chan (rpc.Result), 1)

		go func() {
			q := fmt.Sprintf("%s#%s", repo, issue)
			for {
				res := rpc.Query(cfg, "/issue", q)
				if res.Complete {
					rpcc <- res
					return
				}
				<-time.After(100 * time.Millisecond)
			}
		}()

		select {
		case res := <-rpcc:
			if len(res.Error) > 0 {
				mdLink = fmt.Sprintf("%s (rpc error: %s)", mdLink, res.Error)
			} else if len(res.Issues) > 0 {
				mdLink = fmt.Sprintf("[%s#%s: %s](%s)", repo, issue, res.Issues[0].Title, issueMatches[1])
			} else {
				mdLink += " (rpc error: no issue returned)"
			}
		case <-time.After(5 * time.Second):
			mdLink += " (rpc timed out)"
		}
	}

	return mdLink
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
