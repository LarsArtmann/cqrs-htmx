package cqrshtmx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func validateDispatch[T any](
	getter func(*handlerConfig) func(*http.Request) (T, error),
	setter func(*handlerConfig, func(*http.Request) (T, error)),
	validator func(T) error,
	label string,
) HandlerOption {
	return func(config *handlerConfig) {
		original := getter(config)
		if original == nil {
			slog.Warn("cqrs-htmx: "+label+" applied before decoder",
				slog.String("hint", "apply after DecodeJSON/DecodeForm"))

			return
		}

		setter(config, func(r *http.Request) (T, error) {
			val, err := original(r)
			if err != nil {
				var zero T

				return zero, err
			}

			if valErr := validator(val); valErr != nil {
				var zero T

				return zero, errorfamily.WrapRejection(valErr, "cqrshtmx.validate.failed", "request validation failed")
			}

			return val, nil
		})
	}
}

func getCommandDecoder(config *handlerConfig) func(*http.Request) (command.Command, error) {
	return config.commandDecoder
}

func getQueryDecoder(config *handlerConfig) func(*http.Request) (query.Query, error) {
	return config.queryDecoder
}

// ValidateCommand wraps the command decoder with a validation step.
// The validator receives the decoded command and may return an error.
// Validation errors are wrapped with ErrValidationFailed.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.DecodeJSON(...),
//	    cqrshtmx.ValidateCommand(func(cmd command.Command) error {
//	        // e.g., check required fields
//	        return nil
//	    }),
//	)
func ValidateCommand(validator func(command.Command) error) HandlerOption {
	return validateDispatch(
		getCommandDecoder,
		setCommandDecoder,
		validator,
		"ValidateCommand",
	)
}

// ValidateQuery wraps the query decoder with a validation step.
// The validator receives the decoded query and may return an error.
// Validation errors are wrapped with ErrValidationFailed.
func ValidateQuery(validator func(query.Query) error) HandlerOption {
	return validateDispatch(
		getQueryDecoder,
		setQueryDecoder,
		validator,
		"ValidateQuery",
	)
}

// WithTimeout sets a per-handler timeout override.
// If > 0, it takes precedence over the App-level Config.Timeout.
// Zero or negative means fall back to App config (default: no timeout).
func WithTimeout(d time.Duration) HandlerOption {
	return func(config *handlerConfig) {
		config.timeout = d
	}
}

// WithMaxBodySize sets a per-handler maximum request body size override.
// If > 0, it takes precedence over the App-level Config.MaxBodySize.
// Zero or negative means fall back to App config (default: 10 MB).
func WithMaxBodySize(n int64) HandlerOption {
	return func(config *handlerConfig) {
		config.maxBodySize = n
	}
}

// WithSuccessStatus sets the HTTP status code for successful responses
// when no explicit body is written. Default is 204 No Content.
// Common values: 200 OK, 201 Created.
func WithSuccessStatus(code int) HandlerOption {
	return func(config *handlerConfig) {
		config.successStatus = code
	}
}
