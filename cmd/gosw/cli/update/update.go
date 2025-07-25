// Package update provides the install command for the gosw CLI.
package update

import (
	"github.com/kechako/gosw/env"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the list of available Go versions",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			e := env.FromContext(cmd.Context())

			if err := e.UpdateDownloadList(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
