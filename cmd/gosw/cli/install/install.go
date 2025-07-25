// Package install provides the install command for the gosw CLI.
package install

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/kechako/gosw/env"
	"github.com/kechako/table"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [flags] [--list | --list-all | <version>]",
		Short: "Install a specific Go version or list available versions",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			list, _ := cmd.Flags().GetBool("list")
			if list {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			e := env.FromContext(cmd.Context())
			releases, err := e.Releases()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			completions := make([]cobra.Completion, 0, len(releases))
			for _, r := range releases {
				version := r.Version.String()
				if strings.HasPrefix(version, toComplete) {
					completions = append(completions, cobra.Completion(r.Version.String()))
				}
			}
			return completions, cobra.ShellCompDirectiveNoSpace
		},
		Args: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			listAll, _ := cmd.Flags().GetBool("list-all")
			var n int
			if list || listAll {
				n = 0
			} else {
				n = 1
			}

			if err := cobra.ExactArgs(n)(cmd, args); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			e := env.FromContext(cmd.Context())

			list, _ := cmd.Flags().GetBool("list")
			listAll, _ := cmd.Flags().GetBool("list-all")
			if !list && !listAll {
				v, err := env.ParseVersion(args[0])
				if err != nil {
					return errors.New("version syntax is not valid")
				}

				if err := e.Install(v); err != nil {
					return err
				}

				return nil
			}

			var releases []*env.Release
			var err error
			if listAll {
				releases, err = e.Releases()
			} else if list {
				releases, err = e.RecentReleases()
			}
			if err != nil {
				return err
			}

			verbose, _ := cmd.Flags().GetBool("verbose")

			if verbose {
				t := table.New(
					&table.Column{Title: "Version", Alignment: table.AlignLeft},
					&table.Column{Title: "Stable", Alignment: table.AlignCenter},
					&table.Column{Title: "Filename", Alignment: table.AlignLeft},
					&table.Column{Title: "Size", Alignment: table.AlignRight},
					&table.Column{Title: "Checksum SHA256", Alignment: table.AlignLeft},
				)
				for _, r := range releases {
					t.AddRow(
						table.String(r.Version.String()),
						table.Bool(r.Stable),
						table.String(r.Filename),
						table.String(formatBytes(r.Size)),
						table.String(r.ChecksumSHA256),
					)
				}
				t.Format(os.Stdout)
			} else {
				for _, r := range releases {
					fmt.Println(r.Version)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolP("list", "l", false, "List recent available versions")
	cmd.Flags().BoolP("list-all", "L", false, "List all available versions")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information about versions")

	return cmd
}

func formatBytes(value int64) string {
	bytes := float64(value)

	// snprintf below uses %4.2f, so 1023.99 MiB should be shown as 1.00 GiB
	bytesAbs := math.Abs(bytes) / 1023.995 * 1024

	const kib = uint64(1024)
	const mib = uint64(1024 * kib)
	const gib = uint64(1024 * mib)
	const tib = uint64(1024 * gib)
	const pib = uint64(1024 * tib)
	const eib = uint64(1024 * pib)

	var divisor uint64
	var unit string

	if bytesAbs >= float64(eib) {
		divisor = eib
		unit = "EiB"
	} else if bytesAbs >= float64(pib) {
		divisor = pib
		unit = "PiB"
	} else if bytesAbs >= float64(tib) {
		divisor = tib
		unit = "TiB"
	} else if bytesAbs >= float64(gib) {
		divisor = gib
		unit = "GiB"
	} else if bytesAbs >= float64(mib) {
		divisor = mib
		unit = "MiB"
	} else if bytesAbs >= float64(kib) {
		divisor = kib
		unit = "KiB"
	} else {
		divisor = 1
		unit = "bytes"
	}

	if divisor == 1 {
		return fmt.Sprintf("%4d %s", int(bytes), unit)
	} else {
		v := bytes / float64(divisor)
		if math.Abs(v) >= 100000.0 {
			return fmt.Sprintf("%4.2e %s", v, unit)
		} else {
			return fmt.Sprintf("%4.2f %s", v, unit)
		}
	}
}
