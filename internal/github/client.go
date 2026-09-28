// Package github commits generated schema files and tags releases directly
// against a GitHub repository using the upstream go-github client's Git
// Data API, rather than a local checkout or the git binary.
package github

import gogithub "github.com/google/go-github/v75/github"

// Client commits generated files and tags releases directly against a
// GitHub repository through the Git Data API, rather than a local checkout.
type Client struct {
	git    *gogithub.GitService
	owner  string
	repo   string
	branch string
}

// NewClient returns a Client for the given repository, authenticated with
// token. The token needs the "contents" write permission.
func NewClient(token, owner, repo, branch string) *Client {
	gh := gogithub.NewClient(nil).WithAuthToken(token)
	return &Client{
		git:    gh.Git,
		owner:  owner,
		repo:   repo,
		branch: branch,
	}
}
