package snippets

import (
	"fmt"
	"regexp"
	"time"

	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

var repoRegex = regexp.MustCompile(`(https://github\.com/([^/]+)/([^/]+)\b)(.?)`)
var issueRegex = regexp.MustCompile(`(https://github\.com/([^/]+)/([^/]+)/(issues|pull)/(\d+))#?`)
var discussionRegex = regexp.MustCompile(`(https://github\.com/orgs/([^/]+)/teams/([^/]+)/discussions/(\d+))#?`)

// MarkdownLink looks for a github issue or PR URL and converts it to a markdown link.
//
// "https://github.com/zerowidth/camper_van/issues/1" becomes a markdown link
// with link text "zerowidth/camper_van#1".
func MarkdownLink(rpcClient rpc.Client, input string, includeDesc bool) string {
	issueMatches := issueRegex.FindStringSubmatch(input)
	discussionMatches := discussionRegex.FindStringSubmatchIndex(input)
	repoMatches := repoRegex.FindStringSubmatch(input)

	if discussionMatches != nil {
		template := "[@$2/$3#$4]($1)"
		result := []byte{}
		result = discussionRegex.ExpandString(result, template, input, discussionMatches)
		return string(result)
	}

	if issueMatches != nil {
		url := issueMatches[1]
		repo := fmt.Sprintf("%s/%s", issueMatches[2], issueMatches[3])
		issue := issueMatches[5]
		return formatIssue(rpcClient, url, repo, issue, includeDesc)
	}

	// Don't want to match a repo url with anything after it, but can't do a
	// negative lookahead to ignore a trailing /. Capture and check here instead.
	if repoMatches != nil && repoMatches[4] != "/" {
		url := repoMatches[1]
		repo := fmt.Sprintf("%s/%s", repoMatches[2], repoMatches[3])
		return formatRepo(rpcClient, url, repo, includeDesc)
	}

	return input
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

func formatIssue(rpcClient rpc.Client, url, repo, issue string, includeDesc bool) string {
	mdLink := fmt.Sprintf("[%s#%s](%s)", repo, issue, url)

	if includeDesc {
		resultChan := make(chan (rpc.Result), 1)

		go func() {
			for {
				res := rpcClient.Query("/issue", fmt.Sprintf("%s#%s", repo, issue))
				if res.Complete {
					resultChan <- res
					return
				}
				<-time.After(100 * time.Millisecond)
			}
		}()

		select {
		case res := <-resultChan:
			if len(res.Error) > 0 {
				mdLink = fmt.Sprintf("%s (rpc error: %s)", mdLink, res.Error)
			} else if len(res.Issues) > 0 {
				mdLink = fmt.Sprintf("[%s#%s: %s](%s)", repo, issue, res.Issues[0].Title, url)
			} else {
				mdLink += " (rpc error: no data returned)"
			}
		case <-time.After(5 * time.Second):
			mdLink += " (rpc timed out)"
		}
	}

	return mdLink
}

func formatRepo(rpcClient rpc.Client, url, repo string, includeDesc bool) string {
	mdLink := fmt.Sprintf("[%s](%s)", repo, url)

	if includeDesc {
		resultChan := make(chan (rpc.Result), 1)

		go func() {
			for {
				res := rpcClient.Query("/repo", repo)
				if res.Complete {
					resultChan <- res
					return
				}
				<-time.After(100 * time.Millisecond)
			}
		}()

		select {
		case res := <-resultChan:
			if len(res.Error) > 0 {
				mdLink = fmt.Sprintf("%s (rpc error: %s)", mdLink, res.Error)
			} else if len(res.Repos) > 0 {
				mdLink = fmt.Sprintf("[%s: %s](%s)", repo, res.Repos[0].Description, url)
			} else {
				mdLink += " (rpc error: no data returned)"
			}
		case <-time.After(5 * time.Second):
			mdLink += " (rpc timed out)"
		}
	}

	return mdLink
}
