package cqrshtmx_test

import (
	"context"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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

		It("propagates context deadline via event.FromContext", func() {
			deadline := time.Now().Add(30 * time.Second)
			ctx, cancel := context.WithDeadline(context.Background(), deadline)
			defer cancel()

			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())

			aggID := id.NewAggregateID()
			evt, err := event.NewEvent(
				"TestEvent",
				aggID,
				"Test",
				1,
				[]byte(`{}`),
				opts...,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(evt).NotTo(BeNil())
		})

		It("returns nil when no IDs and no deadline", func() {
			ctx := context.Background()
			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).To(BeNil())
		})

		It("propagates timeout deadline via event.FromContext", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			opts := cqrshtmx.EventOptionsFromContext(ctx)
			Expect(opts).NotTo(BeNil())

			aggID := id.NewAggregateID()
			evt, err := event.NewEvent(
				"TimeoutTestEvent",
				aggID,
				"Test",
				1,
				[]byte(`{}`),
				opts...,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(evt).NotTo(BeNil())
		})
	})

	Describe("EventOptionsFromContextWithSource", func() {
		It("returns nil options for empty context and empty service name", func() {
			opts := cqrshtmx.EventOptionsFromContextWithSource(context.Background(), "")
			Expect(opts).To(BeNil())
		})

		It("returns base options for empty service name", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)

			opts := cqrshtmx.EventOptionsFromContextWithSource(ctx, "")
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("appends source option when service name is valid", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)

			opts := cqrshtmx.EventOptionsFromContextWithSource(ctx, "my-service")
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(2))
		})

		It("accepts any non-empty service name (ParseSource only checks empty)", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)

			opts := cqrshtmx.EventOptionsFromContextWithSource(ctx, "anything-non-empty")
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(2))
		})
	})

	Describe("App.EventOptions", func() {
		It("returns nil options when no IDs and no service name", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: command.NewDispatcher(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(app.EventOptions(context.Background())).To(BeNil())
		})

		It("appends source option when ServiceName is set", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:    command.NewDispatcher(),
				ServiceName: "my-service",
			})
			Expect(err).NotTo(HaveOccurred())

			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)

			opts := app.EventOptions(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(2))
		})

		It("returns base options when ServiceName is empty", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: command.NewDispatcher(),
			})
			Expect(err).NotTo(HaveOccurred())

			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)

			opts := app.EventOptions(ctx)
			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("exposes ServiceName via getter", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:    command.NewDispatcher(),
				ServiceName: "exposed-name",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(app.ServiceName()).To(Equal("exposed-name"))
		})

		It("returns empty string from ServiceName when unset", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Queries: query.NewDispatcher(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(app.ServiceName()).To(BeEmpty())
		})
	})
})
