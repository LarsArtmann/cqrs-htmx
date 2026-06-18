package cqrshtmx_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
)

func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }

func erroringCommandHandler(msg string) func(_ context.Context, _ command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg)
	}
}
