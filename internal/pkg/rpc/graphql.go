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

	var r Repo
	r.Description = query.Repository.Description
	res.Repos = append(res.Repos, r)

	return err
}

// GetIssue retrieves an issue's information
func (g *GitHubClient) GetIssue(res *Result, issue string) error {
	owner, name, number, err := splitIssue(issue)
	if err != nil {
		return err
	}
	var query struct {
		Repository struct {
			IssueOrPullRequest issueOrPullRequest `graphql:"issueOrPullRequest(number:$number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	vars := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"number": githubv4.Int(number),
	}
	err = g.query(&query, vars)
	res.Issues = append(res.Issues, query.Repository.IssueOrPullRequest.toIssue())
	return err
}

// GetIssues retrieves issues from the search API given a query
func (g *GitHubClient) GetIssues(res *Result, query string) error {
	var search struct {
		Search struct {
			Nodes []issueOrPullRequest
		} `graphql:"search(query:$query, type:ISSUE, first:20)"`
	}
	vars := map[string]interface{}{
		"query": githubv4.String(query),
	}
	err := g.query(&search, vars)
	for _, n := range search.Search.Nodes {
		res.Issues = append(res.Issues, n.toIssue())
	}
	return err
}

// GetProject retrieves a project for either an org or a repo
func (g *GitHubClient) GetProject(res *Result, query string) error {
	user, repo, number, err := splitProject(query)
	if err != nil {
		return err
	}
	if len(repo) == 0 {
		return g.getOrgProject(res, user, number)
	}
	return g.getRepoProject(res, user, repo, number)
}

func (g *GitHubClient) getOrgProject(res *Result, org string, number int) error {
	var q struct {
		Organization struct {
			Project struct {
				Name  string
				State string
			} `graphql:"project(number:$number)"`
		} `graphql:"organization(login:$login)"`
	}
	vars := map[string]interface{}{
		"login":  githubv4.String(org),
		"number": githubv4.Int(number),
	}
	err := g.query(&q, vars)

	var project Project
	project.Name = q.Organization.Project.Name
	project.State = q.Organization.Project.State
	res.Projects = append(res.Projects, project)

	return err
}

func (g *GitHubClient) getRepoProject(res *Result, user, repo string, number int) error {
	var q struct {
		Repository struct {
			Project struct {
				Name  string
				State string
			} `graphql:"project(number:$number)"`
		} `graphql:"repository(owner:$owner,name:$name)"`
	}
	vars := map[string]interface{}{
		"owner":  githubv4.String(user),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}
	err := g.query(&q, vars)

	var project Project
	project.Name = q.Repository.Project.Name
	project.State = q.Repository.Project.State
	res.Projects = append(res.Projects, project)

	return err
}

// wrap query with a timeout
func (g *GitHubClient) query(q interface{}, vars map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), graphqlTimeout)
	defer cancel()
	return g.client.Query(ctx, q, vars)
}

type issueFragment struct {
	State      string
	Title      string
	Number     int
	Repository struct {
		Name  string
		Owner struct {
			Login string
		}
	}
}

type issueOrPullRequest struct {
	Type        string        `graphql:"__typename"`
	Issue       issueFragment `graphql:"...on Issue"`
	PullRequest issueFragment `graphql:"...on PullRequest"`
}

func (ip issueOrPullRequest) toIssue() Issue {
	if ip.Type == "PullRequest" {
		return ip.PullRequest.toIssue(ip.Type)
	}
	return ip.Issue.toIssue(ip.Type)
}

func (f issueFragment) toIssue(t string) Issue {
	var i Issue
	i.Type = t
	i.State = f.State
	i.Title = f.Title
	i.Repo = fmt.Sprintf("%s/%s", f.Repository.Owner.Login, f.Repository.Name)
	i.Number = fmt.Sprintf("%d", f.Number)
	return i
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

// Splits owner/repo/number into owner, repo, number. Repo is optional.
func splitProject(project string) (string, string, int, error) {
	var user, repo, num string
	var number int
	split := strings.SplitN(project, "/", 3)
	if len(split) < 2 {
		return "", "", 0, fmt.Errorf("incomplete project owner/<repo>/number: %v", project)
	}
	if len(split) == 2 {
		user = split[0]
		num = split[1]
	} else {
		user = split[0]
		repo = split[1]
		num = split[2]
	}
	number, err := strconv.Atoi(num)
	if err != nil {
		return "", "", 0, err
	}
	return user, repo, number, nil
}
