package cqrshtmx_test

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context", func() {
	Describe("WithUserID / UserIDFromContext", func() {
		It("stores and retrieves a user ID", func() {
			ctx := context.Background()
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, want)
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(Equal(want))
		})

		It("returns zero value when no user ID is set", func() {
			ctx := context.Background()
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(BeZero())
		})

		It("overwrites a previously set user ID", func() {
			ctx := context.Background()
			id1 := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			id2 := cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
			ctx = cqrshtmx.WithUserID(ctx, id1)
			ctx = cqrshtmx.WithUserID(ctx, id2)
			Expect(cqrshtmx.UserIDFromContext(ctx)).To(Equal(id2))
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
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("returns nil for zero UserID", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithUserID(ctx, cqrshtmx.UserID{})
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).To(BeNil())
		})
	})
})
