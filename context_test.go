package cqrshtmx_test

import (
	"context"

	"github.com/larsartmann/cqrs-htmx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context", func() {
	Describe("WithUserID / UserIDFromContext", func() {
		It("stores and retrieves a user ID", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithUserID(ctx, "user-123")
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(Equal("user-123"))
		})

		It("returns empty string when no user ID is set", func() {
			ctx := context.Background()
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(BeEmpty())
		})

		It("overwrites a previously set user ID", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithUserID(ctx, "user-1")
			ctx = cqrshtmx.WithUserID(ctx, "user-2")
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(Equal("user-2"))
		})
	})

	Describe("EventOptionsFromContext", func() {
		It("returns nil options when no user ID is set", func() {
			ctx := context.Background()
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).To(BeNil())
		})

		It("returns options with user ID when set", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithUserID(ctx, "01HQAAAAAAAAAAAAAAAAAAAAAAAAAAA")
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("returns options with empty UserID for invalid IDs", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithUserID(ctx, "invalid-id-format")
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})
	})
})
