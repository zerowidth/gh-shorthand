package rpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"golang.org/x/oauth2"
)

const graphqlTimeout = 10 * time.Second

// GitHubClient wraps a githubv4 graphql client connection
type GitHubClient struct {
	client *githubv4.Client
}

// NewGitHubClient returns a GitHub graphqlv4 client wrapper from a config
func NewGitHubClient(cfg config.Config) *GitHubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.APIToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &GitHubClient{
		client: githubv4.NewClient(httpClient),
	}
}

// GetRepo retrieves a repo's information
func (g *GitHubClient) GetRepo(res *Result, repo string) error {
	r := Repo{}
	owner, name, err := splitRepo(repo)
	if err != nil {
		return err
	}
	var query struct {
		Repository struct {
			Description string
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	vars := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}
	err = g.query(&query, vars)
	r.Description = query.Repository.Description
	res.Repos = append(res.Repos, r)
	return err
}

// GetIssue retrieve's an issue's information
func (g *GitHubClient) GetIssue(res *Result, issue string) error {
	i := Issue{}
	owner, name, number, err := splitIssue(issue)
	if err != nil {
		return err
	}
	var query struct {
		Repository struct {
			IssueOrPullRequest struct {
				TypeName string `graphql:"__typename"`
				Issue    struct {
					State string
					Title string
				} `graphql:"...on Issue"`
				PullRequest struct {
					State string
					Title string
				} `graphql:"...on PullRequest"`
			} `graphql:"issueOrPullRequest(number:$number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	vars := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"number": githubv4.Int(number),
	}
	err = g.query(&query, vars)
	i.Type = query.Repository.IssueOrPullRequest.TypeName
	if query.Repository.IssueOrPullRequest.TypeName == "PullRequest" {
		i.State = query.Repository.IssueOrPullRequest.PullRequest.State
		i.Title = query.Repository.IssueOrPullRequest.PullRequest.Title
	} else {
		i.State = query.Repository.IssueOrPullRequest.Issue.State
		i.Title = query.Repository.IssueOrPullRequest.Issue.Title
	}
	res.Issues = append(res.Issues, i)
	return err
}

// wrap query with a timeout
func (g *GitHubClient) query(q interface{}, vars map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), graphqlTimeout)
	defer cancel()
	return g.client.Query(ctx, q, vars)
}

// Splits owner/repo into owner, repo
func splitRepo(nameWithOwner string) (string, string, error) {
	split := strings.SplitN(nameWithOwner, "/", 2)
	if len(split) < 2 || len(split[1]) == 0 {
		return "", "", fmt.Errorf("incomplete repo owner/name: %v", nameWithOwner)
	}
	return split[0], split[1], nil
}

// Splits owner/repo#number into owner, repo, and number
func splitIssue(issue string) (string, string, int, error) {
	owner, name, err := splitRepo(issue)
	if err != nil {
		return "", "", 0, err
	}
	split := strings.SplitN(name, "#", 2)
	if len(split) < 2 || len(split[1]) == 0 {
		return "", "", 0, fmt.Errorf("incomplete issue owner/name#issue: %v", issue)
	}
	number, err := strconv.Atoi(split[1])
	if err != nil {
		return "", "", 0, err
	}
	return owner, split[0], number, nil
}
