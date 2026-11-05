package hx

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// RequestID is the per-request identifier type stored in the context. It is the
// same branded type the root package re-exports as cqrshtmx.RequestID, so a
// request ID written by the root middleware is readable here and vice versa.
type RequestID = id.RequestID

type requestIDContextKey struct{}

// WithRequestID stores a strongly-typed request ID in the context.
func WithRequestID(ctx context.Context, requestID RequestID) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext retrieves the request ID stored by WithRequestID.
// Returns the zero value of RequestID if no request ID is present.
func RequestIDFromContext(ctx context.Context) RequestID {
	v, _ := ctx.Value(requestIDContextKey{}).(RequestID)

	return v
}

// NewRequestID generates a new random RequestID.
func NewRequestID() RequestID {
	return id.NewRequestID()
}

// ParseRequestID converts a string to a RequestID.
func ParseRequestID(s string) (RequestID, error) {
	return id.ParseRequestID(s)
}

// MustParseRequestID converts a string to a RequestID, panicking on error.
func MustParseRequestID(s string) RequestID {
	v, err := ParseRequestID(s)
	if err != nil {
		panic(err)
	}

	return v
}
