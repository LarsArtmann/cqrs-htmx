package cataloghtmx

import (
	"strings"
	"unicode"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

// Builder accumulates messages and produces an immutable catalog.Catalog.
// It wraps catalog.Builder with a streamlined single-service API.
//
// Create one with New, then register messages using the generic
// Command[T], Query[T], or Event[T] package-level functions, or via
// AddMessage with catalog.Command[T] etc. Call Build to get the final catalog.
//
// Example:
//
//	b := cataloghtmx.New("User Service", "1.0.0")
//	cataloghtmx.Command[RegisterUserCmd](b, "register-user",
//	    cataloghtmx.WithOperation("POST", "/api/users"))
//	cataloghtmx.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)
//	cat := b.Build()
type Builder struct {
	inner      *catalog.Builder
	serviceID  catalog.ServiceID
	serviceCfg serviceConfig
	msgs       []catalog.MessageConfig
}

type serviceConfig struct {
	name    string
	version string
	summary string
}

// Option configures a Builder.
type Option func(*Builder)

// WithServiceName overrides the service display name (defaults to the title).
func WithServiceName(name string) Option {
	return func(b *Builder) {
		b.serviceCfg.name = name
	}
}

// WithServiceSummary sets a human-readable summary for the service.
func WithServiceSummary(summary string) Option {
	return func(b *Builder) {
		b.serviceCfg.summary = summary
	}
}

// WithServiceID overrides the service ID (defaults to kebab-case of title).
func WithServiceID(id string) Option {
	return func(b *Builder) {
		b.serviceID = catalog.ServiceID(id)
	}
}

// New creates a Builder for a single-service catalog.
// The title becomes both the catalog title and the default service name.
// The version applies to both the catalog and the service.
func New(title, version string, opts ...Option) *Builder {
	b := &Builder{
		inner: catalog.NewBuilder(title, version),
		serviceCfg: serviceConfig{
			name:    title,
			version: version,
		},
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.serviceID == "" {
		b.serviceID = catalog.ServiceID(toKebab(title))
	}

	return b
}

// Command registers a command message on the builder.
// The schema is auto-derived from T via reflection on its struct fields
// and tags. The name is auto-derived from T's type name
// (e.g., RegisterUserCmd → "Register User").
//
// Returns the builder for potential chaining.
func Command[T any](b *Builder, id string, opts ...catalog.MessageOption) *Builder {
	b.msgs = append(b.msgs, catalog.Command[T](catalog.MessageID(id), opts...))
	return b
}

// Query registers a query message on the builder.
// The schema is auto-derived from T. Direction defaults to Receives.
//
// Returns the builder for potential chaining.
func Query[T any](b *Builder, id string, opts ...catalog.MessageOption) *Builder {
	b.msgs = append(b.msgs, catalog.Query[T](catalog.MessageID(id), opts...))
	return b
}

// Event registers an event message on the builder.
// The schema is auto-derived from T. The direction must be explicit
// (catalog.Sends or catalog.Receives).
//
// Returns the builder for potential chaining.
func Event[T any](
	b *Builder,
	id string,
	direction catalog.Direction,
	opts ...catalog.MessageOption,
) *Builder {
	b.msgs = append(b.msgs, catalog.Event[T](catalog.MessageID(id), direction, opts...))
	return b
}

// AddMessage adds a pre-built MessageConfig (from catalog.Command[T],
// catalog.Event[T], or catalog.Query[T]). This is an alternative to the
// generic Command[T]/Query[T]/Event[T] package-level functions.
func (b *Builder) AddMessage(msg catalog.MessageConfig) *Builder {
	b.msgs = append(b.msgs, msg)
	return b
}

// Build creates the immutable catalog with all registered messages.
func (b *Builder) Build() *catalog.Catalog {
	b.inner.AddService(
		b.serviceID,
		b.serviceCfg.name,
		b.serviceCfg.version,
		b.serviceCfg.summary,
		b.msgs...,
	)

	return b.inner.Build()
}

// Registry returns the underlying catalog.Registry for advanced use cases.
func (b *Builder) Registry() *catalog.Registry {
	return b.inner.Registry()
}

// InnerBuilder returns the underlying catalog.Builder.
// Use this for multi-service catalogs, domains, channels, or other
// catalog features not exposed by this wrapper.
func (b *Builder) InnerBuilder() *catalog.Builder {
	return b.inner
}

// WithOperation attaches HTTP endpoint metadata to a message.
// The OpenAPI exporter uses this to generate accurate paths.
//
// This is a re-export of catalog.MsgOperation for convenience.
func WithOperation(method, path string) catalog.MessageOption {
	return catalog.MsgOperation(method, path)
}

// toKebab converts a title-cased or space-separated string to kebab-case.
// e.g., "User Service" → "user-service", "MyAPIService" → "my-api-service".
func toKebab(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	var buf strings.Builder

	prevDash := false

	for i, r := range s {
		if r == '-' {
			if !prevDash {
				buf.WriteRune('-')
			}

			prevDash = true
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevDash {
				buf.WriteRune('-')
			}

			buf.WriteRune(unicode.ToLower(r))
		} else {
			buf.WriteRune(r)
		}

		prevDash = false
	}

	result := buf.String()
	result = strings.Trim(result, "-")

	return result
}
