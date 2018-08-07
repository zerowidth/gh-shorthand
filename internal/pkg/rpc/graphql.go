package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"golang.org/x/oauth2"
)

// GitHubClient wraps a githubv4 graphql client connection
type GitHubClient struct {
	client *githubv4.Client
}

// NewGitHubClient returns a GitHub graphqlv4 client wrapper from a config
func NewGitHubClient(cfg config.Config) *GitHubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.ApiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &GitHubClient{
		client: githubv4.NewClient(httpClient),
	}
}

// GetRepoDescription retrieves a repository description by name and owner
func (g *GitHubClient) GetRepoDescription(nameWithOwner string) (string, error) {
	owner, name, err := splitRepo(nameWithOwner)
	if err != nil {
		return "", err
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = g.client.Query(ctx, &query, vars)
	return query.Repository.Description, err
}

func splitRepo(nameWithOwner string) (string, string, error) {
	split := strings.SplitN(nameWithOwner, "/", 2)
	if len(split) < 2 || len(split[1]) == 0 {
		return "", "", fmt.Errorf("incomplete repo owner/name: %v", nameWithOwner)
	}
	return split[0], split[1], nil
}
