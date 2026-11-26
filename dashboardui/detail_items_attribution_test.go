package dashboardui

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/templ-components/display"
)

// The command and query detail pages name who the audit record is attributed
// to: the user when the transport stamped one, and the device key when the
// browser sent its X-Client-Id header.

// itemTerms lists the terms of a definition list in render order.
func itemTerms(items []display.DefinitionItem) []string {
	terms := make([]string, 0, len(items))
	for _, item := range items {
		terms = append(terms, item.Term)
	}

	return terms
}

// attributedQuery builds a persisted query carrying the given identity under
// the metadata keys the cqrs-htmx transport stamps.
func attributedQuery(userID id.UserID, deviceKey string) *query.PersistedQuery {
	GinkgoHelper()

	options := []query.Option{query.WithRequestID(id.NewRequestID())}
	if !userID.IsZero() {
		options = append(options, query.WithUserID(userID))
	}

	if deviceKey != "" {
		options = append(options, query.WithCustomMetadata(string(event.MetadataKeyClientID), deviceKey))
	}

	basic, err := query.New("Catalog.Search", options...)
	Expect(err).NotTo(HaveOccurred())

	persisted, err := query.NewPersistedQuery(
		basic.Type(), nil, query.WithQueryMetadata(basic.Metadata()),
	)
	Expect(err).NotTo(HaveOccurred())

	return persisted
}

// attributedCommand is attributedQuery's command-side twin.
func attributedCommand(userID id.UserID, deviceKey string) *command.PersistedCommand {
	GinkgoHelper()

	options := []command.Option{command.WithRequestID(id.NewRequestID())}
	if !userID.IsZero() {
		options = append(options, command.WithUserID(userID))
	}

	if deviceKey != "" {
		options = append(options, command.WithCustomMetadata(string(event.MetadataKeyClientID), deviceKey))
	}

	streamID := id.NewStreamID()
	basic, err := command.New("Catalog.Reindex", streamID, options...)
	Expect(err).NotTo(HaveOccurred())

	persisted, err := command.NewPersistedCommand(
		basic.Type(), id.NewStreamRef(id.StreamType("catalog"), streamID), nil,
		command.WithPersistedCommandID(basic.ID()),
		command.WithCommandMetadata(basic.Metadata()),
	)
	Expect(err).NotTo(HaveOccurred())

	return persisted
}

var _ = Describe("audit record attribution items", func() {
	userID := id.NewUserID()
	const deviceKey = "01HZDEVICEDEVICEDEVICEDEVICE0"

	Describe("on the query detail page", func() {
		It("shows the user and the device when the query carries both", func() {
			items := queryMetaItems(attributedQuery(userID, deviceKey), "nonce")

			Expect(itemTerms(items)[baseQueryMetaItems : baseQueryMetaItems+2]).
				To(Equal([]string{"User ID", "Device ID"}))
			Expect(renderComponentString(items[baseQueryMetaItems].DetailComponent)).
				To(ContainSubstring(userID.String()))
			Expect(renderComponentString(items[baseQueryMetaItems+1].DetailComponent)).
				To(ContainSubstring(deviceKey))
		})

		It("omits the device when the browser sent no device key", func() {
			terms := itemTerms(queryMetaItems(attributedQuery(userID, ""), "nonce"))

			Expect(terms).To(ContainElement("User ID"))
			Expect(terms).NotTo(ContainElement("Device ID"))
		})

		It("omits both when the record is anonymous", func() {
			terms := itemTerms(queryMetaItems(attributedQuery(id.UserID{}, ""), "nonce"))

			Expect(terms).NotTo(ContainElement("User ID"))
			Expect(terms).NotTo(ContainElement("Device ID"))
		})
	})

	Describe("on the command detail page", func() {
		It("shows the user and the device when the command carries both", func() {
			items := commandMetaItems(attributedCommand(userID, deviceKey), "nonce")

			Expect(itemTerms(items)[baseCommandMetaItems : baseCommandMetaItems+2]).
				To(Equal([]string{"User ID", "Device ID"}))
			Expect(renderComponentString(items[baseCommandMetaItems].DetailComponent)).
				To(ContainSubstring(userID.String()))
		})
	})
})
