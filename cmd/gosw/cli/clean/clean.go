// Package clean provides the install command for the gosw CLI.
package clean

import (
	"github.com/kechako/gosw/env"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Short: "Clean up downloaded archives",
		Use:   "clean",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			e := env.FromContext(cmd.Context())

			if err := e.Clean(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
