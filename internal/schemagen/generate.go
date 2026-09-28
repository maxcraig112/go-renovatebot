// Package schemagen turns a renovate-schema.json document into a single
// generated Go source file containing the equivalent structs.
package schemagen

import (
	"bytes"
	"fmt"

	"github.com/atombender/go-jsonschema/pkg/generator"
	"github.com/atombender/go-jsonschema/pkg/schemas"
)

// PackageName is the Go package declared in the generated file.
const PackageName = "renovatebot"

// RootTypeName is the name of the generated struct representing the root
// of a renovate.json configuration file.
const RootTypeName = "Config"

// outputFileName is only used internally to key the generator's output map;
// it is never written to disk under this name.
const outputFileName = "types_gen.go"

// Generate reads a renovate-schema.json document and returns the generated
// Go source for it.
//
// The schema is parsed directly from memory rather than from a file path.
// go-jsonschema resolves local file references with net/url.Parse, which
// misreads a Windows absolute path such as "C:\..." as a URL with scheme
// "C"; going through the library's in-memory API avoids that entirely.
func Generate(schemaJSON []byte) ([]byte, error) {
	stripped, err := stripSelfReferences(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("schemagen: stripping self-references: %w", err)
	}

	schema, err := schemas.FromJSONReader(bytes.NewReader(stripped))
	if err != nil {
		return nil, fmt.Errorf("schemagen: parsing schema: %w", err)
	}

	config := generator.Config{
		SchemaMappings: []generator.SchemaMapping{
			{
				SchemaID:    schema.ID,
				PackageName: PackageName,
				RootType:    RootTypeName,
				OutputName:  outputFileName,
			},
		},
		DefaultPackageName: PackageName,
		DefaultOutputName:  outputFileName,
		Tags:               []string{"json"},
		Warner:             func(string) {}, // renovate's schema has known benign warnings; suppress them
	}

	gen, err := generator.New(config)
	if err != nil {
		return nil, fmt.Errorf("schemagen: creating generator: %w", err)
	}

	if err := gen.AddFile("renovate-schema.json", schema); err != nil {
		return nil, fmt.Errorf("schemagen: generating types: %w", err)
	}

	sources, err := gen.Sources()
	if err != nil {
		return nil, fmt.Errorf("schemagen: rendering sources: %w", err)
	}

	src, ok := sources[outputFileName]
	if !ok {
		return nil, fmt.Errorf("schemagen: generator produced no output named %q", outputFileName)
	}

	return src, nil
}
