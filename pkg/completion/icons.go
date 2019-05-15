package completion

import (
	"fmt"

	"github.com/zerowidth/gh-shorthand/pkg/alfred"
)

var (
	// githubIcon      = octicon("mark-github")
	repoIcon = octicon("repo")
	// pullRequestIcon = octicon("git-pull-request")
	issueListIcon = octicon("list-ordered")
	pathIcon      = octicon("browser")
	issueIcon     = octicon("issue-opened")
	projectIcon   = octicon("project")
	newIssueIcon  = octicon("bug")
	editorIcon    = octicon("file-code")
	finderIcon    = octicon("file-directory")
	terminalIcon  = octicon("terminal")
	markdownIcon  = octicon("markdown")
	searchIcon    = octicon("search")
	// commitIcon    = octicon("git-commit")

	issueIconOpen         = octicon("issue-opened_open")
	issueIconClosed       = octicon("issue-closed_closed")
	pullRequestIconOpen   = octicon("git-pull-request_open")
	pullRequestIconClosed = octicon("git-pull-request_closed")
	pullRequestIconMerged = octicon("git-merge_merged")
	projectIconOpen       = octicon("project_open")
	projectIconClosed     = octicon("project_closed")
)

// An octicon is relative to the alfred workflow, so this tells alfred to
// retrieve icons from there rather than relative to this go binary.
func octicon(name string) *alfred.Icon {
	return &alfred.Icon{
		Path: fmt.Sprintf("octicons-%s.png", name),
	}
}

func issueStateIcon(kind, state string) *alfred.Icon {
	switch kind {
	case "Issue":
		if state == "OPEN" {
			return issueIconOpen
		}
		return issueIconClosed
	case "PullRequest":
		switch state {
		case "OPEN":
			return pullRequestIconOpen
		case "CLOSED":
			return pullRequestIconClosed
		case "MERGED":
			return pullRequestIconMerged
		}
	}
	return issueIcon // sane default
}

func projectStateIcon(state string) *alfred.Icon {
	if state == "OPEN" {
		return projectIconOpen
	}
	return projectIconClosed
}
