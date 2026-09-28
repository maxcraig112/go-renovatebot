// Package pipeline wires together npmregistry and schemagen: given a
// Renovate release version, it produces the generated Go source for that
// release's config schema.
package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/maxcraig112/go-renovatebot/internal/npmregistry"
)

// NpmPackage is the npm package whose releases are tracked.
const NpmPackage = "renovate"

// schemaPathInTarball is where renovate-schema.json lives inside the
// package tarball, relative to the tarball root.
const schemaPathInTarball = "package/renovate-schema.json"

// FetchSchema downloads renovate-schema.json for the given Renovate npm
// version.
func FetchSchema(ctx context.Context, client *npmregistry.Client, version string) ([]byte, error) {
	data, err := client.FetchTarballFile(ctx, NpmPackage, version, schemaPathInTarball)
	if errors.Is(err, npmregistry.ErrFileNotFound) {
		return nil, fmt.Errorf("%w: %s@%s", ErrSchemaNotPublished, NpmPackage, version)
	}
	if err != nil {
		return nil, fmt.Errorf("pipeline: fetching schema for %s@%s: %w", NpmPackage, version, err)
	}
	return data, nil
}
