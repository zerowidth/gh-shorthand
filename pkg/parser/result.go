package parser

import "strings"

// Result is a result from the new parser
type Result struct {
	User          string
	Name          string
	UserShorthand string
	RepoShorthand string
	Issue         string
	Path          string
	Query         string
}

// SetRepo overrides owner and name on the result from an `owner/name` string.
func (r *Result) SetRepo(repo string) {
	parts := strings.SplitN(repo, "/", 2)
	r.User = parts[0]
	if len(parts) > 1 {
		r.Name = parts[1]
	}
}

// HasRepo checks if the result has a fully qualified repo, either from a
// matched repo shorthand, or from an explicit owner/name.
func (r *Result) HasRepo() bool {
	return len(r.User) > 0 && len(r.Name) > 0
}

// HasUser checks if the result has a matched user
func (r *Result) HasUser() bool {
	return len(r.User) > 0
}

// HasIssue checks if the result has a matched issue
func (r *Result) HasIssue() bool {
	return len(r.Issue) > 0
}

// HasPath checks if the result has a matched path
func (r *Result) HasPath() bool {
	return len(r.Path) > 0
}

// HasQuery checks if the result has a matched path
func (r *Result) HasQuery() bool {
	return len(r.Query) > 0
}

// Repo returns the repo defined in the result, if present
func (r *Result) Repo() string {
	if r.HasRepo() {
		return r.User + "/" + r.Name
	}
	return ""
}

// Annotation is a helper for displaying details about a match. Returns a string
// with a leading space, noting the matched shorthand and issue if applicable.
func (r *Result) Annotation() string {
	var annotation string
	if len(r.RepoShorthand) > 0 {
		annotation += " (" + r.RepoShorthand
		if r.HasIssue() {
			annotation += "#" + r.Issue
		}
		annotation += ")"
	} else if len(r.UserShorthand) > 0 {
		annotation += " (" + r.UserShorthand + ")"
	}
	return annotation
}
