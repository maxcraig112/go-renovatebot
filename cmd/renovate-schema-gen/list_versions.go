package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/maxcraig112/go-renovatebot/internal/npmregistry"
	"github.com/maxcraig112/go-renovatebot/internal/pipeline"
)

func newListVersionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-versions",
		Short: "List every published Renovate npm version, ascending",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := npmregistry.NewClient()

			versions, err := client.Versions(cmd.Context(), pipeline.NpmPackage)
			if err != nil {
				return err
			}

			for _, v := range versions {
				fmt.Println(v)
			}
			return nil
		},
	}

	return cmd
}
