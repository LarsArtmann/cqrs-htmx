package cqrshtmx_test

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	errorfamily "github.com/larsartmann/go-error-family"
)

func BenchmarkMapError(b *testing.B) {
	b.Run("Unauthorized", func(b *testing.B) {
		for range b.N {
			httperr.MapError(httperr.ErrUnauthorized)
		}
	})
	b.Run("Forbidden", func(b *testing.B) {
		for range b.N {
			httperr.MapError(httperr.ErrForbidden)
		}
	})
	b.Run("Rejection", func(b *testing.B) {
		err := errorfamily.NewRejection("test.rejection", "rejected")
		for range b.N {
			httperr.MapError(err)
		}
	})
	b.Run("Conflict", func(b *testing.B) {
		err := errorfamily.NewConflict("test.conflict", "conflict")
		for range b.N {
			httperr.MapError(err)
		}
	})
	b.Run("Transient", func(b *testing.B) {
		err := errorfamily.NewTransient("test.transient", "transient")
		for range b.N {
			httperr.MapError(err)
		}
	})
	b.Run("Nil", func(b *testing.B) {
		for range b.N {
			httperr.MapError(nil)
		}
	})
}
