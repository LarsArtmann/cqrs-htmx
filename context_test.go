package cqrshtmx_test

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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

	Describe("WithCorrelationID / CorrelationIDFromContext", func() {
		It("stores and retrieves a correlation ID", func() {
			ctx := context.Background()
			want := cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithCorrelationID(ctx, want)
			Expect(cqrshtmx.CorrelationIDFromContext(ctx)).To(Equal(want))
		})

		It("returns zero value when no correlation ID is set", func() {
			Expect(cqrshtmx.CorrelationIDFromContext(context.Background())).To(BeZero())
		})
	})

	Describe("CorrelationID types", func() {
		It("re-exports the CorrelationID type", func() {
			_ = cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")
		})

		It("parses a valid correlation ID", func() {
			cid, err := cqrshtmx.ParseCorrelationID("01HK1549P84T9XF8R94E960633")
			Expect(err).NotTo(HaveOccurred())
			Expect(cid.String()).To(Equal("01HK1549P84T9XF8R94E960633"))
		})

		It("returns error for invalid correlation ID", func() {
			_, err := cqrshtmx.ParseCorrelationID("not-a-ulid")
			Expect(err).To(HaveOccurred())
		})

		It("panics for invalid MustParseCorrelationID", func() {
			Expect(func() {
				cqrshtmx.MustParseCorrelationID("not-a-ulid")
			}).To(Panic())
		})

		It("generates a new correlation ID", func() {
			cid := cqrshtmx.NewCorrelationID()
			Expect(cid.IsZero()).To(BeFalse())
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

		It("propagates correlation ID into event options", func() {
			ctx := context.Background()
			cid := cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithCorrelationID(ctx, cid)
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("propagates both user ID and correlation ID", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)
			cid := cqrshtmx.MustParseCorrelationID("01HK154ANGZHV2ZW0X3SKSNEN2")
			ctx = cqrshtmx.WithCorrelationID(ctx, cid)
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(2))
		})

		It("ignores zero correlation ID silently", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)
			ctx = cqrshtmx.WithCorrelationID(ctx, cqrshtmx.CorrelationID{})
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("returns nil when only zero correlation ID and no user ID", func() {
			ctx := context.Background()
			ctx = cqrshtmx.WithCorrelationID(ctx, cqrshtmx.CorrelationID{})
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).To(BeNil())
		})

		It("produces valid event.Option that NewEvent accepts", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			cid := cqrshtmx.MustParseCorrelationID("01HK154ANGZHV2ZW0X3SKSNEN2")
			aggID := id.NewAggregateID()
			ctx = cqrshtmx.WithUserID(ctx, userID)
			ctx = cqrshtmx.WithCorrelationID(ctx, cid)
			opts := cqrshtmx.EventOptionsFromContext(ctx)

			evt, err := event.NewEvent(
				"UserCreated",
				aggID,
				"User",
				1,
				[]byte(`{"Name":"Alice"}`),
				opts...,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(evt).NotTo(BeNil())
			Expect(evt.Metadata()).NotTo(BeNil())
		})
	})
})
