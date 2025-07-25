// Package cli provides a simple command-line interface for managing Go environment.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/kechako/gosw/cmd/gosw/cli/clean"
	"github.com/kechako/gosw/cmd/gosw/cli/clierrors"
	"github.com/kechako/gosw/cmd/gosw/cli/install"
	"github.com/kechako/gosw/cmd/gosw/cli/uninstall"
	"github.com/kechako/gosw/cmd/gosw/cli/update"
	"github.com/kechako/gosw/cmd/gosw/cli/use"
	"github.com/kechako/gosw/cmd/gosw/cli/versions"
	"github.com/kechako/gosw/env"
	"github.com/spf13/cobra"
)

const (
	appName    = "gosw"
	appVersion = "2.0.0"
)

func Main() {
	defaultRoot, err := getDefaultRoot()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	cmd := &cobra.Command{
		Use:     appName,
		Version: appVersion,
		Short:   "gosw is a simple command-line interface for managing Go environment",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			root, err := cmd.Flags().GetString("root")
			if err != nil {
				root = defaultRoot
			}
			e, err := env.New(
				env.WithEnvRoot(root),
			)
			if err != nil {
				return clierrors.Exit(err, 1)
			}

			cmd.SetContext(env.NewContext(cmd.Context(), e))

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		clean.Command(),
		install.Command(),
		versions.Command(),
		uninstall.Command(),
		update.Command(),
		use.Command(),
	)

	cmd.PersistentFlags().String("root", defaultRoot, "Set the root directory for gosw")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		code := 1
		var exitCoder clierrors.ExitCoder
		if errors.As(err, &exitCoder) {
			code = exitCoder.ExitCode()
		}

		printError(err)

		if code != 0 {
			os.Exit(code)
		}
	}
}

func getDefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	_ = home
	return filepath.Join("/usr/local/go"), nil
	//return filepath.Join(home, ".local/share/gosw"), nil
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}
