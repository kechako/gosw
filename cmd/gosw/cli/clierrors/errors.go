// Package clierrors provides error handling utilities for command-line interfaces.
package clierrors

import "fmt"

type ExitCoder interface {
	error
	ExitCode() int
}

type exitError struct {
	Err  error
	Code int
}

func Exit(err error, code int) error {
	return &exitError{Err: err, Code: code}
}

func (e *exitError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}

	return e.Err.Error()
}

func (e *exitError) ExitCode() int {
	if e == nil {
		return 0
	}
	return e.Code
}

func (e *exitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
