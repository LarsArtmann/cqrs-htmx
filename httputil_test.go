package cqrshtmx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP Utilities", func() {
	const testRemoteAddr = "10.0.0.1:1234"

	Describe("WriteJSON", func() {
		It("writes JSON with the given status code", func() {
			w := httptest.NewRecorder()

			cqrshtmx.WriteJSON(w, http.StatusCreated, map[string]string{"name": "test"})

			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

			var result map[string]string
			Expect(json.Unmarshal(w.Body.Bytes(), &result)).To(Succeed())
			Expect(result["name"]).To(Equal("test"))
		})

		It("writes an empty object for nil value", func() {
			w := httptest.NewRecorder()

			cqrshtmx.WriteJSON(w, http.StatusOK, nil)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("null\n"))
		})
	})

	Describe("ClientIP", func() {
		DescribeTable(
			"extracts IP from headers",
			func(headerName, headerValue, expectedIP string) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				if headerName != "" {
					r.Header.Set(headerName, headerValue)
				}
				r.RemoteAddr = testRemoteAddr
				Expect(cqrshtmx.ClientIP(r)).To(Equal(expectedIP))
			},
			Entry("extracts first IP from X-Forwarded-For",
				"X-Forwarded-For", "1.2.3.4, 5.6.7.8", "1.2.3.4"),
			Entry("uses X-Real-IP when X-Forwarded-For is empty",
				"X-Real-IP", "9.8.7.6", "9.8.7.6"),
			Entry("falls back to RemoteAddr with SplitHostPort",
				"", "", "10.0.0.1"),
		)

		It("returns RemoteAddr as-is when SplitHostPort fails", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "no-colon-or-port"

			Expect(cqrshtmx.ClientIP(r)).To(Equal("no-colon-or-port"))
		})

		It("trims whitespace from X-Forwarded-For entries", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("X-Forwarded-For", "  1.2.3.4  , 5.6.7.8")

			Expect(cqrshtmx.ClientIP(r)).To(Equal("1.2.3.4"))
		})
	})

	Describe("StatusRecorder", func() {
		It("captures status code via WriteHeader", func() {
			w := httptest.NewRecorder()
			rec := cqrshtmx.NewStatusRecorder(w)

			rec.WriteHeader(http.StatusNotFound)

			Expect(rec.Status()).To(Equal(http.StatusNotFound))
			Expect(rec.WroteHeader()).To(BeTrue())
		})

		It("defaults to 200 when body is written without WriteHeader", func() {
			w := httptest.NewRecorder()
			rec := cqrshtmx.NewStatusRecorder(w)

			_, _ = rec.Write([]byte("hello"))

			Expect(rec.Status()).To(Equal(http.StatusOK))
			Expect(rec.WroteHeader()).To(BeTrue())
		})

		It("reports zero status when nothing written", func() {
			w := httptest.NewRecorder()
			rec := cqrshtmx.NewStatusRecorder(w)

			Expect(rec.Status()).To(Equal(0))
			Expect(rec.WroteHeader()).To(BeFalse())
		})

		It("only captures the first WriteHeader call", func() {
			w := httptest.NewRecorder()
			rec := cqrshtmx.NewStatusRecorder(w)

			rec.WriteHeader(http.StatusNotFound)
			rec.WriteHeader(http.StatusOK)

			Expect(rec.Status()).To(Equal(http.StatusNotFound))
		})
	})
})
