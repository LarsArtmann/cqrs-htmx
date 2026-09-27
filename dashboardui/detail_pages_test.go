package dashboardui

import (
	"regexp"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Page-level specs for the event detail and snapshot detail pages: the
// detail header (badges + neighbor nav) and the RelativeTime adoption on
// the snapshot "Created" line.

func detailEvent() event.Event {
	streamID, err := id.ParseStreamID("01HW8STREAMID000000000000000")
	Expect(err).NotTo(HaveOccurred())
	eventID, err := id.ParseEventID("01HWEVENT1D000000000000000")
	Expect(err).NotTo(HaveOccurred())
	evt, err := event.NewEvent(
		event.Type("UserRegistered"),
		streamID,
		id.StreamType("user"),
		event.Version(3),
		[]byte(`{"email":"user@example.com"}`),
		event.WithEventID(eventID),
		event.WithOccurredAt(time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)),
	)
	Expect(err).NotTo(HaveOccurred())

	return evt
}

var _ = Describe("the event detail page", func() {
	var html string

	BeforeEach(func() {
		p := pageData{Title: "Event", BasePath: "/dashboard", Nonce: "test-nonce"}
		html = renderComponentString(
			eventDetailContent(
				p,
				detailEvent(),
				"01HWPREV000000000000000000",
				"01HWNEXT000000000000000000",
				[]byte(`{"email":"user@example.com"}`),
			),
		)
	})

	It("pins the full page baseline in a golden", func() {
		expectGoldenFile("event_detail_page", html)
	})

	It("shows the event type as a code element with schema and encoding badges", func() {
		Expect(html).To(ContainSubstring("<code>UserRegistered</code>"))
		Expect(html).To(ContainSubstring("schema v"))
	})

	It("links to the previous and next events when neighbors exist", func() {
		Expect(html).To(ContainSubstring(`href="/dashboard/events/01HWPREV000000000000000000"`))
		Expect(html).To(ContainSubstring(`href="/dashboard/events/01HWNEXT000000000000000000"`))
	})
})

var _ = Describe("the event detail page without neighbors", func() {
	It("omits the prev/next navigation bar", func() {
		p := pageData{Title: "Event", BasePath: "/dashboard", Nonce: "test-nonce"}

		html := renderComponentString(eventDetailContent(p, detailEvent(), "", "", nil))

		Expect(html).NotTo(ContainSubstring("Previous"))
		Expect(html).NotTo(ContainSubstring("Next"))
	})
})

var _ = Describe("the snapshot detail page", func() {
	var html string

	BeforeEach(func() {
		p := pageData{Title: "Snapshot", BasePath: "/dashboard", Nonce: "test-nonce"}
		streamID, err := id.ParseStreamID("01HW8STREAMID000000000000000")
		Expect(err).NotTo(HaveOccurred())

		created := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
		snap := &snapshot.Snapshot{
			StreamID:   streamID,
			StreamType: id.StreamType("user"),
			Version:    event.Version(7),
			State:      []byte(`{"email":"user@example.com"}`),
			CreatedAt:  created,
		}
		html = renderComponentString(snapshotDetailContent(
			p, id.NewStreamRef(id.StreamType("user"), streamID), snap, "{\n  \"email\": \"user@example.com\"\n}"))
	})

	It("pins the full page baseline in a golden", func() {
		// The RelativeTime label ("3 hours ago") is wall-clock dependent —
		// normalize its text so the golden pins structure, not the hour.
		normalized := regexp.MustCompile(`(<time[^>]*>)[^<]*(</time>)`).ReplaceAllString(html, "$1REL$2")
		expectGoldenFile("snapshot_detail_page", normalized)
	})

	It("renders the Created line through display.RelativeTime (time element with machine datetime)", func() {
		Expect(html).To(ContainSubstring("<time"))
		Expect(html).To(ContainSubstring(`datetime="2026-09-26T12:00:00Z"`))
		Expect(html).To(ContainSubstring("Version 7 · Created 2026-09-26T12:00:00Z"))
	})
})
