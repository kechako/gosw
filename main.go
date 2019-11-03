package main

import (
	"errors"
	"fmt"
	"os"
)

const (
	appName    = "gosw"
	appVersion = "0.1.0"
)

type exitError struct {
	Err  error
	Code int
}

func (err *exitError) Error() string {
	if err.Err == nil {
		return ""
	}

	return err.Err.Error()
}

func (err *exitError) Unwrap() error {
	return err.Err
}

func main() {
	app := NewApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
	}
}
