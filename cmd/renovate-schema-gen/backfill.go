package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ghclient "github.com/maxcraig112/go-renovatebot/internal/github"
	"github.com/maxcraig112/go-renovatebot/internal/npmregistry"
	"github.com/maxcraig112/go-renovatebot/internal/pipeline"
	"github.com/maxcraig112/go-renovatebot/internal/schemagen"
	"github.com/maxcraig112/go-renovatebot/internal/versiontag"
)

const (
	defaultOwner  = "maxcraig112"
	defaultRepo   = "go-renovatebot"
	defaultBranch = "main"
)

func newBackfillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Generate and tag every past Renovate release that published a schema",
		Long: "backfill walks every published Renovate npm version and, for each one that\n" +
			"published renovate-schema.json, generates the Go structs, commits them, and\n" +
			"tags the commit with the matching version, directly against GitHub through\n" +
			"the Git Data API. It is safe to interrupt and rerun: versions that already\n" +
			"have a matching tag are skipped.\n\n" +
			"Requires a GITHUB_TOKEN environment variable with the \"contents\" write\n" +
			"permission on the target repository.",
	}

	from := cmd.Flags().String("from", "", "Skip versions before this npm version (inclusive)")
	to := cmd.Flags().String("to", "", "Skip versions after this npm version (inclusive)")
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
		return runBackfill(cmd.Context(), gh, *from, *to, *output)
	}

	return cmd
}

type backfillOutcome int

const (
	outcomeGenerated backfillOutcome = iota
	outcomeAlreadyTagged
	outcomeNoSchema
	outcomeFailed
)

func runBackfill(ctx context.Context, gh *ghclient.Client, from, to, output string) error {
	client := npmregistry.NewClient()

	versions, err := client.Versions(ctx, pipeline.NpmPackage)
	if err != nil {
		return err
	}

	counts := map[backfillOutcome]int{}

	for _, version := range versions {
		if !inRange(version, from, to) {
			continue
		}

		outcome, err := backfillVersion(ctx, gh, client, version, output)
		if err != nil {
			return fmt.Errorf("backfill: %w", err)
		}
		counts[outcome]++
	}

	fmt.Printf(
		"done: %d generated, %d already tagged, %d had no published schema, %d failed\n",
		counts[outcomeGenerated], counts[outcomeAlreadyTagged], counts[outcomeNoSchema], counts[outcomeFailed],
	)
	return nil
}

func inRange(version, from, to string) bool {
	if from != "" && versiontag.Less(version, from) {
		return false
	}
	if to != "" && versiontag.Less(to, version) {
		return false
	}
	return true
}

// backfillVersion generates, commits, and tags a single Renovate version.
// The returned error is only non-nil for failures that should abort the
// whole backfill (GitHub API failures); problems specific to one version
// are reported via the returned outcome instead.
func backfillVersion(
	ctx context.Context, gh *ghclient.Client, client *npmregistry.Client, version, output string,
) (backfillOutcome, error) {
	tag := versiontag.Tag(version)

	exists, err := gh.TagExists(ctx, tag)
	if err != nil {
		return outcomeFailed, err
	}
	if exists {
		return outcomeAlreadyTagged, nil
	}

	schemaJSON, err := pipeline.FetchSchema(ctx, client, version)
	if errors.Is(err, pipeline.ErrSchemaNotPublished) {
		return outcomeNoSchema, nil
	}
	if err != nil {
		fmt.Printf("skipping %s: %v\n", version, err)
		return outcomeFailed, nil
	}

	src, err := schemagen.Generate(schemaJSON)
	if err != nil {
		fmt.Printf("skipping %s: generating structs: %v\n", version, err)
		return outcomeFailed, nil
	}

	message := fmt.Sprintf("chore: regenerate schema for renovate %s", version)
	sha, err := gh.CommitAndTag(ctx, output, src, message, tag)
	if err != nil {
		return outcomeFailed, err
	}

	fmt.Printf("tagged %s -> %s\n", tag, sha)
	return outcomeGenerated, nil
}
