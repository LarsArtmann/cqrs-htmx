package cqrshtmx_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
)

func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }
