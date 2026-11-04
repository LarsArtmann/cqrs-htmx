package htmx_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX Core", func() {
	DescribeTable(
		"bool accessors",
		func(header string, setHeader bool, accessor func(*http.Request) bool, expected bool) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if setHeader {
				r.Header.Set(header, htmx.HeaderTrue)
			}

			Expect(accessor(r)).To(Equal(expected))
		},
		Entry("IsRequest true", "HX-Request", true, htmx.IsRequest, true),
		Entry("IsRequest absent", "HX-Request", false, htmx.IsRequest, false),
		Entry("IsBoosted true", "HX-Boosted", true, htmx.IsBoosted, true),
		Entry("IsBoosted absent", "HX-Boosted", false, htmx.IsBoosted, false),
		Entry(
			"IsHistoryRestore true",
			"HX-History-Restore-Request",
			true,
			htmx.IsHistoryRestore,
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
		Entry("Target set", "HX-Target", "user-list", htmx.Target, "user-list"),
		Entry("Target empty", "HX-Target", "", htmx.Target, ""),
		Entry("Trigger set", "HX-Trigger", "submit-btn", htmx.Trigger, "submit-btn"),
		Entry("Prompt set", "HX-Prompt", "yes", htmx.Prompt, "yes"),
		Entry(
			"CurrentURL set",
			"HX-Current-URL",
			"https://example.com/users",
			htmx.CurrentURL,
			"https://example.com/users",
		),
	)

	It("returns false when HX-Request header is not 'true'", func() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Hx-Request", "false")
		Expect(htmx.IsRequest(r)).To(BeFalse())
	})

	It("classifies a nil request as non-HTMX instead of panicking", func() {
		Expect(htmx.IsRequest(nil)).To(BeFalse())
		Expect(htmx.IsBoosted(nil)).To(BeFalse())
		Expect(htmx.Target(nil)).To(BeEmpty())
	})
})
