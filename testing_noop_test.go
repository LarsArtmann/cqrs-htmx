package cqrshtmx_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }

func erroringCommandHandler(msg string) func(_ context.Context, _ command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg)
	}
}

// erroringCommandHandlerWith returns a command handler that always returns the
// given (pre-classified) error. Use it when a test needs a specific error
// family (e.g. Conflict) rather than the default Transient classification of a
// raw errors.New.
func erroringCommandHandlerWith(err error) func(_ context.Context, _ command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return err
	}
}
