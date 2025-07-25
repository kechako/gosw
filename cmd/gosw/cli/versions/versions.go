// Package versions provides the install command for the gosw CLI.
package versions

import (
	"fmt"

	"github.com/kechako/gosw/env"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions",
		Short: "List installed Go versions",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			e := env.FromContext(cmd.Context())

			versions := e.InstalledVersions()
			for _, v := range versions {
				fmt.Println(v)
			}

			return nil
		},
	}

	return cmd
}
