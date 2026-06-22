package cqrshtmx_test

import (
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

func BenchmarkMapError(b *testing.B) {
	b.Run("Unauthorized", func(b *testing.B) {
		for range b.N {
			cqrshtmx.MapError(cqrshtmx.ErrUnauthorized)
		}
	})
	b.Run("Forbidden", func(b *testing.B) {
		for range b.N {
			cqrshtmx.MapError(cqrshtmx.ErrForbidden)
		}
	})
	b.Run("Rejection", func(b *testing.B) {
		err := event.NewRejection("test.rejection", "rejected")
		for range b.N {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Conflict", func(b *testing.B) {
		err := event.NewConflict("test.conflict", "conflict")
		for range b.N {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Transient", func(b *testing.B) {
		err := event.NewTransient("test.transient", "transient")
		for range b.N {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Nil", func(b *testing.B) {
		for range b.N {
			cqrshtmx.MapError(nil)
		}
	})
}
