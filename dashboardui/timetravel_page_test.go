package dashboardui

import (
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Page-level specs for the time-travel detail page: the payload column (the
// version slider alone only ever showed event types and timestamps), the
// step-row emphasis at the selected version, and the htmx-driven slider
// (hx-get + name="v" keeps ?v=N bookmarkable while swapping #main-content
// instead of a full page reload).

type timetravelTestEvent struct {
	version event.Version
	email   string
}

func timetravelEvent(tt timetravelTestEvent) event.Event {
	streamID, err := id.ParseStreamID("01HW8STREAMID000000000000000")
	Expect(err).NotTo(HaveOccurred())
	eventID, err := id.ParseEventID("01HWEVENT" + tt.version.String() + "D000000000000000")
	Expect(err).NotTo(HaveOccurred())
	evt, err := event.NewEvent(
		event.Type("UserRegistered"),
		streamID,
		id.StreamType("user"),
		tt.version,
		[]byte(`{"email":"`+tt.email+`"}`),
		event.WithEventID(eventID),
		event.WithOccurredAt(time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)),
	)
	Expect(err).NotTo(HaveOccurred())

	return evt
}

var _ = Describe("the time-travel detail page", func() {
	var html string

	BeforeEach(func() {
		p := pageData{Title: "Time Travel", BasePath: "/dashboard", Nonce: "test-nonce"}
		ref := id.NewStreamRef(id.StreamType("user"), mustParseStreamID("01HW8STREAMID000000000000000"))
		rows := buildTimeTravelRows(
			Config{}.withPayloadRenderingDefaults(),
			[]event.Event{
				timetravelEvent(timetravelTestEvent{version: event.Version(1), email: "v1@example.com"}),
				timetravelEvent(timetravelTestEvent{version: event.Version(2), email: "v2@example.com"}),
			},
			event.Version(2),
		)
		html = renderComponentString(
			timeTravelDetailContent(p, ref, rows, event.Version(2), event.Version(3)),
		)
	})

	It("pins the full page baseline in a golden", func() {
		expectGoldenFile("time_travel_detail_page", html)
	})

	It("renders the payload column with each event's data inline", func() {
		Expect(html).To(ContainSubstring(">Payload</th>"))
		Expect(html).To(ContainSubstring("&#34;email&#34;: &#34;v1@example.com&#34;"))
		Expect(html).To(ContainSubstring("&#34;email&#34;: &#34;v2@example.com&#34;"))
	})

	It("highlights and expands the step row at the selected version only", func() {
		Expect(html).To(ContainSubstring(`class="step-row"`))
		Expect(html).To(ContainSubstring(`<details class="payload-details" open>`))
		Expect(html).To(ContainSubstring(`<details class="payload-details">`))
		Expect(html).To(ContainSubstring("this step"))
		// Exactly one step row and one expanded payload.
		Expect(strings.Count(html, `class="step-row"`)).To(Equal(1))
		Expect(strings.Count(html, `<details class="payload-details" open>`)).To(Equal(1))
	})

	It("drives the slider through htmx so a version change is a partial swap, not a reload", func() {
		// The ID segment is StreamID display-form (branded-kind prefix — gotcha 25).
		Expect(
			html,
		).To(ContainSubstring(`hx-get="/dashboard/time-travel/user/StreamMarker:01HW8STREAMID000000000000000"`))
		Expect(html).To(ContainSubstring(`hx-trigger="change"`))
		Expect(html).To(ContainSubstring(`hx-target="#main-content"`))
		Expect(html).To(ContainSubstring(`hx-select="#main-content"`))
		Expect(html).To(ContainSubstring(`hx-push-url="true"`))
		Expect(html).To(ContainSubstring(`hx-sync="this:replace"`))
		Expect(html).To(ContainSubstring(`name="v"`))
		Expect(html).NotTo(ContainSubstring("data-nav-base"))
	})
})

func mustParseStreamID(s string) id.StreamID {
	sid, err := id.ParseStreamID(s)
	Expect(err).NotTo(HaveOccurred())

	return sid
}
