package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX Core", func() {
	DescribeTable(
		"bool accessors",
		func(header string, setHeader bool, accessor func(*http.Request) bool, expected bool) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if setHeader {
				r.Header.Set(header, cqrshtmx.HeaderTrue)
			}
			Expect(accessor(r)).To(Equal(expected))
		},
		Entry("IsHTMXRequest true", "HX-Request", true, cqrshtmx.IsHTMXRequest, true),
		Entry("IsHTMXRequest absent", "HX-Request", false, cqrshtmx.IsHTMXRequest, false),
		Entry("IsBoosted true", "HX-Boosted", true, cqrshtmx.IsBoosted, true),
		Entry("IsBoosted absent", "HX-Boosted", false, cqrshtmx.IsBoosted, false),
		Entry(
			"IsHistoryRestore true",
			"HX-History-Restore-Request",
			true,
			cqrshtmx.IsHistoryRestore,
			true,
		),
	)

	DescribeTable(
		"string accessors",
		func(header, value string, accessor func(*http.Request) string, expected string) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if value != "" {
				r.Header.Set(header, value)
			}
			Expect(accessor(r)).To(Equal(expected))
		},
		Entry("HTMXTarget set", "HX-Target", "user-list", cqrshtmx.HTMXTarget, "user-list"),
		Entry("HTMXTarget empty", "HX-Target", "", cqrshtmx.HTMXTarget, ""),
		Entry("HTMXTrigger set", "HX-Trigger", "submit-btn", cqrshtmx.HTMXTrigger, "submit-btn"),
		Entry("HTMXPrompt set", "HX-Prompt", "yes", cqrshtmx.HTMXPrompt, "yes"),
		Entry(
			"HTMXCurrentURL set",
			"HX-Current-URL",
			"https://example.com/users",
			cqrshtmx.HTMXCurrentURL,
			"https://example.com/users",
		),
	)

	It("returns false when HX-Request header is not 'true'", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "false")
		Expect(cqrshtmx.IsHTMXRequest(r)).To(BeFalse())
	})
})
