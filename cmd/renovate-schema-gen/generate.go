package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/maxcraig112/go-renovatebot/internal/npmregistry"
	"github.com/maxcraig112/go-renovatebot/internal/pipeline"
	"github.com/maxcraig112/go-renovatebot/internal/schemagen"
)

const defaultOutputPath = "types_gen.go"

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Go structs for a single Renovate release",
	}

	version := cmd.Flags().String("version", "", `Renovate npm version to generate, e.g. 44.116.0, or "latest"`)
	output := cmd.Flags().String("output", defaultOutputPath, "Path to write the generated Go file to")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *version == "" {
			return fmt.Errorf("--version is required")
		}
		return generateOne(cmd.Context(), *version, *output)
	}

	return cmd
}

// generateOne fetches and writes the generated struct file for a single
// Renovate version. It performs no git operations.
func generateOne(ctx context.Context, version, output string) error {
	client := npmregistry.NewClient()

	if version == "latest" {
		resolved, err := client.LatestVersion(ctx, pipeline.NpmPackage)
		if err != nil {
			return err
		}
		version = resolved
	}

	schemaJSON, err := pipeline.FetchSchema(ctx, client, version)
	if err != nil {
		return err
	}

	src, err := schemagen.Generate(schemaJSON)
	if err != nil {
		return fmt.Errorf("generating structs for %s: %w", version, err)
	}

	if err := os.WriteFile(output, src, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", output, err)
	}

	fmt.Printf("generated %s from renovate@%s\n", output, version)
	return nil
}
