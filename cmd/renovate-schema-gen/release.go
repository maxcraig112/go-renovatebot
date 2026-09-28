package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ghclient "github.com/maxcraig112/go-renovatebot/internal/github"
	"github.com/maxcraig112/go-renovatebot/internal/npmregistry"
	"github.com/maxcraig112/go-renovatebot/internal/pipeline"
)

func newReleaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Generate structs for one Renovate release and publish them as a tagged commit",
		Long: "release fetches the schema for a single Renovate version, generates the Go\n" +
			"structs, and commits and tags them directly against GitHub through the Git\n" +
			"Data API. It is idempotent: if the tag already exists, it does nothing.\n\n" +
			"Requires a GITHUB_TOKEN environment variable with the \"contents\" write\n" +
			"permission on the target repository. This is what the hourly workflow runs.",
	}

	version := cmd.Flags().String("version", "latest", `Renovate npm version to publish, e.g. 44.116.0, or "latest"`)
	output := cmd.Flags().String("output", defaultOutputPath, "Path to write the generated file to in the repository tree")
	owner := cmd.Flags().String("owner", defaultOwner, "GitHub repository owner")
	repo := cmd.Flags().String("repo", defaultRepo, "GitHub repository name")
	branch := cmd.Flags().String("branch", defaultBranch, "Branch to commit onto")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required")
		}
		gh := ghclient.NewClient(token, *owner, *repo, *branch)
		return runRelease(cmd.Context(), gh, *version, *output)
	}

	return cmd
}

func runRelease(ctx context.Context, gh *ghclient.Client, version, output string) error {
	client := npmregistry.NewClient()

	if version == "latest" {
		resolved, err := client.LatestVersion(ctx, pipeline.NpmPackage)
		if err != nil {
			return err
		}
		version = resolved
	}

	outcome, err := backfillVersion(ctx, gh, client, version, output)
	if err != nil {
		return err
	}

	switch outcome {
	case outcomeGenerated:
		fmt.Printf("published renovate@%s\n", version)
	case outcomeAlreadyTagged:
		fmt.Printf("renovate@%s is already published, nothing to do\n", version)
	case outcomeNoSchema:
		return fmt.Errorf("renovate@%s did not publish a schema", version)
	case outcomeFailed:
		return fmt.Errorf("failed to publish renovate@%s", version)
	}
	return nil
}
