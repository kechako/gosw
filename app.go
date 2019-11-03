package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/kechako/gosw/goenv"
	"github.com/urfave/cli"
)

type App struct {
	app *cli.App
	env *goenv.Env
}

func NewApp() *App {
	app := &App{}
	app.init()

	return app
}

func (app *App) init() {
	a := cli.NewApp()
	a.Name = appName
	a.Version = appVersion
	a.Usage = "Switch version of Go."
	a.Authors = []cli.Author{
		{Name: "Ryosuke Akiyama", Email: "r@554.jp"},
	}
	a.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Value:  "/usr/local/go",
			Usage:  "Root directory to install Go",
			EnvVar: "GOSW_ROOT",
		},
	}

	a.Before = func(c *cli.Context) error {
		env, err := goenv.New(
			goenv.WithEnvRoot(c.GlobalString("root")),
		)
		if err != nil {
			return err
		}

		app.env = env
		return nil
	}

	a.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "List installed versions",
			Action: app.listCommand,
		},
		{
			Name:   "switch",
			Usage:  "Switch current Go version",
			Action: app.switchCommand,
		},
		{
			Name:   "update",
			Usage:  "update download list",
			Action: app.updateCommand,
		},
		{
			Name:   "dllist",
			Usage:  "list downloads",
			Action: app.dlListCommand,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "verbose",
				},
			},
		},
		{
			Name:   "install",
			Usage:  "Install specified version of Go",
			Action: app.installCommand,
		},
		{
			Name:   "uninstall",
			Usage:  "Uninstall specified version of Go",
			Action: app.uninstallCommand,
		},
	}

	a.OnUsageError = func(context *cli.Context, err error, isSubcommand bool) error {
		// exit with code 2 if it's a usage error
		return &exitError{Err: err, Code: 2}
	}

	app.app = a
}

func (a *App) Run(args []string) error {
	if err := a.app.Run(os.Args); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			return exitErr
		}

		return &exitError{
			Err:  err,
			Code: 1,
		}
	}

	return nil
}

func (app *App) listCommand(c *cli.Context) error {
	versions := app.env.InstalledVersions()
	for _, v := range versions {
		fmt.Println(v)
	}

	return nil
}

func (app *App) switchCommand(c *cli.Context) error {
	v, err := goenv.ParseVersion(c.Args().First())
	if err != nil {
		return errors.New("version syntax is not valid")
	}

	if err := app.env.Switch(v); err != nil {
		return err
	}

	return nil
}

func (app *App) installCommand(c *cli.Context) error {
	v, err := goenv.ParseVersion(c.Args().First())
	if err != nil {
		return errors.New("version syntax is not valid")
	}

	if err := app.env.Install(v); err != nil {
		return err
	}

	return nil
}

func (app *App) uninstallCommand(c *cli.Context) error {
	v, err := goenv.ParseVersion(c.Args().First())
	if err != nil {
		return errors.New("version syntax is not valid")
	}

	if err := app.env.Uninstall(v); err != nil {
		return err
	}

	return nil
}

func (app *App) updateCommand(c *cli.Context) error {
	if err := app.env.UpdateDownloadList(); err != nil {
		return err
	}

	return nil
}

func (app *App) dlListCommand(c *cli.Context) error {
	releases, err := app.env.Releases()
	if err != nil {
		return err
	}

	verbose := c.Bool("verbose")

	for _, r := range releases {
		if verbose {
			fmt.Printf("%s %t %s %d %s\n",
				r.Version,
				r.Stable,
				r.Filename,
				r.Size,
				r.ChecksumSHA256,
			)
		} else {
			fmt.Println(r.Version)
		}
	}

	return nil
}
