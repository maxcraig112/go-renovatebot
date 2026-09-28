package github

import (
	"context"
	"crypto/sha1"
	"fmt"

	gogithub "github.com/google/go-github/v75/github"
)

// TagExists reports whether the given tag already exists in the repository.
func (c *Client) TagExists(ctx context.Context, tag string) (bool, error) {
	_, resp, err := c.git.GetRef(ctx, c.owner, c.repo, "tags/"+tag)
	if resp != nil && resp.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("github: checking tag %s: %w", tag, err)
	}
	return true, nil
}

// CommitAndTag writes content at path on top of the branch's current HEAD
// and tags the result with tag. If content is identical to what is already
// committed at path, no new commit is made and tag is applied to the
// existing HEAD instead. It returns the commit SHA the tag now points to.
func (c *Client) CommitAndTag(ctx context.Context, path string, content []byte, message, tag string) (string, error) {
	headRef, _, err := c.git.GetRef(ctx, c.owner, c.repo, "heads/"+c.branch)
	if err != nil {
		return "", fmt.Errorf("github: getting HEAD of %s: %w", c.branch, err)
	}
	headSHA := headRef.GetObject().GetSHA()

	baseCommit, _, err := c.git.GetCommit(ctx, c.owner, c.repo, headSHA)
	if err != nil {
		return "", fmt.Errorf("github: getting commit %s: %w", headSHA, err)
	}

	unchanged, err := c.pathMatchesContent(ctx, baseCommit.GetTree().GetSHA(), path, content)
	if err != nil {
		return "", err
	}

	targetSHA := headSHA
	if !unchanged {
		targetSHA, err = c.commitFile(ctx, baseCommit, path, content, message)
		if err != nil {
			return "", err
		}
	}

	if err := c.createTag(ctx, tag, targetSHA); err != nil {
		return "", err
	}
	return targetSHA, nil
}

// pathMatchesContent reports whether path already contains content in the
// tree rooted at treeSHA, checked by comparing git blob hashes so no extra
// API call is needed to fetch the existing blob's bytes.
func (c *Client) pathMatchesContent(ctx context.Context, treeSHA, path string, content []byte) (bool, error) {
	tree, _, err := c.git.GetTree(ctx, c.owner, c.repo, treeSHA, true)
	if err != nil {
		return false, fmt.Errorf("github: getting tree %s: %w", treeSHA, err)
	}

	for _, entry := range tree.Entries {
		if entry.GetPath() == path {
			return entry.GetSHA() == gitBlobSHA(content), nil
		}
	}
	return false, nil
}

func (c *Client) commitFile(
	ctx context.Context, baseCommit *gogithub.Commit, path string, content []byte, message string,
) (string, error) {
	blob, _, err := c.git.CreateBlob(ctx, c.owner, c.repo, gogithub.Blob{
		Content:  gogithub.Ptr(string(content)),
		Encoding: gogithub.Ptr("utf-8"),
	})
	if err != nil {
		return "", fmt.Errorf("github: creating blob for %s: %w", path, err)
	}

	tree, _, err := c.git.CreateTree(ctx, c.owner, c.repo, baseCommit.GetTree().GetSHA(), []*gogithub.TreeEntry{
		{
			Path: gogithub.Ptr(path),
			Mode: gogithub.Ptr("100644"),
			Type: gogithub.Ptr("blob"),
			SHA:  blob.SHA,
		},
	})
	if err != nil {
		return "", fmt.Errorf("github: creating tree: %w", err)
	}

	commit, _, err := c.git.CreateCommit(ctx, c.owner, c.repo, gogithub.Commit{
		Message: gogithub.Ptr(message),
		Tree:    tree,
		Parents: []*gogithub.Commit{{SHA: baseCommit.SHA}},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("github: creating commit: %w", err)
	}

	if _, _, err := c.git.UpdateRef(ctx, c.owner, c.repo, "heads/"+c.branch, gogithub.UpdateRef{
		SHA: commit.GetSHA(),
	}); err != nil {
		return "", fmt.Errorf("github: updating %s to %s: %w", c.branch, commit.GetSHA(), err)
	}

	return commit.GetSHA(), nil
}

func (c *Client) createTag(ctx context.Context, tag, sha string) error {
	_, _, err := c.git.CreateRef(ctx, c.owner, c.repo, gogithub.CreateRef{
		Ref: "refs/tags/" + tag,
		SHA: sha,
	})
	if err != nil {
		return fmt.Errorf("github: creating tag %s at %s: %w", tag, sha, err)
	}
	return nil
}

// gitBlobSHA computes the git object hash for a blob, the same value the
// GitHub API returns for a tree entry's SHA.
func gitBlobSHA(content []byte) string {
	h := sha1.New() //nolint:gosec // git's object hash is SHA-1 by definition, not used for security here
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}
