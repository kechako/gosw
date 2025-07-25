// Package uninstall provides the install command for the gosw CLI.
package uninstall

import (
	"errors"
	"strings"

	"github.com/kechako/gosw/env"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall [flags] <version>",
		Short: "Uninstall a specific Go version",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			e := env.FromContext(cmd.Context())
			versions := e.InstalledVersions()

			completions := make([]cobra.Completion, 0, len(versions))
			for _, version := range versions {
				if strings.HasPrefix(version.String(), toComplete) {
					completions = append(completions, cobra.Completion(version.String()))
				}
			}
			return completions, cobra.ShellCompDirectiveNoSpace
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			e := env.FromContext(cmd.Context())

			v, err := env.ParseVersion(args[0])
			if err != nil {
				return errors.New("version syntax is not valid")
			}

			if err := e.Uninstall(v); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
