package dashboardui

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/a-h/templ"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
)

// Page-level golden + behavior specs for the overview page. The component
// goldens in golden_test.go pin individual adopted components; these pin the
// COMPOSITION — the stat grid container, the polled projection-health
// region, and the recent-events table — which the 2026-09-22 full-templ
// migration left unpinned (it deleted the differential test).
//
// Golden files regenerate with: go test ./... -run TestDashboardUISuite -update
// Review the diff before committing — the page baseline feeds the compiled
// CSS class coverage and the e2e selectors.

// renderComponentString renders a templ component the same way the golden
// helper does (plain Render into a strings.Builder).
func renderComponentString(c templ.Component) string {
	GinkgoHelper()

	var b strings.Builder
	Expect(c.Render(context.Background(), &b)).To(Succeed())

	return b.String()
}

// expectGoldenFile compares got against testdata/golden/<name>.golden,
// rewriting the file when the -update flag is set.
func expectGoldenFile(name, got string) {
	GinkgoHelper()

	path := filepath.Join("testdata", "golden", name+".golden")

	if *updateGolden {
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(got), 0o644)).To(Succeed())

		return
	}

	want, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred(), "golden file missing — run go test ./... -run TestDashboardUISuite -update")
	Expect(
		got,
	).To(Equal(string(want)), "page markup drifted from the pinned baseline (regenerate with -update after an INTENTIONAL change)")
}

func overviewFixtures() (pageData, overviewStats) {
	p := pageData{
		Title:    "Overview",
		BasePath: "/dashboard",
		Nav:      []navItem{{Label: "Overview", Href: "/dashboard/", Active: true}},
		Nonce:    "test-nonce",
	}
	stats := overviewStats(core.Overview{
		TotalEvents:     "1,234",
		TotalAggregates: "42",
		HealthStatus:    "healthy",
		HealthKind:      core.StatusGood,
		DLQCount:        "2",
		Projections: []core.ProjectionStat{
			{
				Name: "user-read-model", Status: "running", StatusKind: core.StatusGood,
				Lag: "0s", Processed: 500, Errors: 0,
			},
		},
		RecentEvents: []core.RecentEvent{
			{
				Time: "2026-09-26 12:00:00", Type: "UserRegistered",
				StreamID: "01HW8EXAMPLE", StreamType: "user", Version: "1",
				EventID: "01HWEVENTID01",
			},
		},
	})

	return p, stats
}

var _ = Describe("the overview page", func() {
	var (
		p     pageData
		stats overviewStats
		html  string
	)

	BeforeEach(func() {
		p, stats = overviewFixtures()
		html = renderComponentString(overviewContent(p, stats))
	})

	It("pins the full page baseline in a golden", func() {
		expectGoldenFile("overview_page", html)
	})

	It("renders the stat grid through display.Grid's bundle-covered auto-fit utility", func() {
		Expect(html).To(ContainSubstring(
			`[grid-template-columns:repeat(auto-fit,minmax(190px,1fr))]`))
		Expect(html).NotTo(ContainSubstring(`class="stat-grid"`),
			"the hand-rolled .stat-grid container must stay retired")
	})

	It("renders every stat card value", func() {
		for _, value := range []string{"1,234", "42", "healthy", "2"} {
			Expect(html).To(ContainSubstring(value))
		}
	})

	It("renders the projection-health region as a library PolledRegion", func() {
		Expect(html).To(ContainSubstring(`id="projection-health"`))
		Expect(html).To(ContainSubstring(`hx-get="/dashboard/-/partials/projection-health"`))
		Expect(html).To(ContainSubstring(`hx-trigger="every 10s, refresh"`),
			"the custom refresh event must survive alongside the poll")
		Expect(html).To(ContainSubstring(`aria-live="polite"`),
			"the library region announces content changes politely")
		Expect(html).To(ContainSubstring(`hx-swap="outerHTML"`),
			"the partial must re-render the region itself or polling dies after one refresh")
	})

	It("links recent events to their detail pages", func() {
		Expect(html).To(ContainSubstring(`href="/dashboard/events/01HWEVENTID01"`))
		Expect(html).To(ContainSubstring("UserRegistered"))
	})
})

var _ = Describe("the overview page without projections", func() {
	It("omits the projection-health panel entirely", func() {
		p, stats := overviewFixtures()
		stats.Projections = nil

		html := renderComponentString(overviewContent(p, stats))

		Expect(html).NotTo(ContainSubstring("projection-health"))
		Expect(html).NotTo(ContainSubstring("Projection Health"))
	})
})
