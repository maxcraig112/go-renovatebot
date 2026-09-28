// Command renovate-schema-gen generates Go structs from Renovate's
// published JSON Schema and manages the git tags that pin each generated
// version.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "renovate-schema-gen",
		Short: "Generate Go structs from Renovate's config JSON Schema",
	}

	root.AddCommand(newGenerateCommand())
	root.AddCommand(newListVersionsCommand())
	root.AddCommand(newReleaseCommand())
	root.AddCommand(newBackfillCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
