package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMXExtensionHandler", func() {
	DescribeTable("serves each embedded extension",
		func(name, wantVersion string) {
			handler := cqrshtmx.HTMXExtensionHandler(name)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ext/"+name+".js", nil)
			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
			Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
			Expect(w.Header().Get("ETag")).To(Equal(`"htmx-ext-` + name + `-` + wantVersion + `"`))
			Expect(w.Body.Len()).To(BeNumerically(">", 0))
		},
		Entry("sse", cqrshtmx.HTMXExtSSE, "2.2.4"),
		Entry("ws", cqrshtmx.HTMXExtWS, "2.0.4"),
		Entry("idiomorph", cqrshtmx.HTMXExtIdiomorph, "0.7.4"),
	)

	It("returns 304 for matching If-None-Match", func() {
		handler := cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/sse.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-sse-2.2.4"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(Equal(0))
	})

	It("returns 200 for non-matching If-None-Match", func() {
		handler := cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/sse.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-sse-2.2.3"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns 405 for POST", func() {
		handler := cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtWS)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/ext/ws.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	})

	It("panics for unknown extension name", func() {
		Expect(func() { cqrshtmx.HTMXExtensionHandler("unknown") }).To(Panic())
	})
})

var _ = Describe("HTMXExtensionsHandler", func() {
	It("serves a concatenated bundle of multiple extensions", func() {
		handler := cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, cqrshtmx.HTMXExtWS, cqrshtmx.HTMXExtIdiomorph)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/bundle.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
		Expect(w.Header().Get("ETag")).To(Equal(`"htmx-ext-bundle-sse-2.2.4,ws-2.0.4,idiomorph-0.7.4"`))
		Expect(w.Body.Len()).To(BeNumerically(">", 0))

		body := w.Body.String()
		Expect(body).To(ContainSubstring("htmx-ext-sse 2.2.4"))
		Expect(body).To(ContainSubstring("htmx-ext-ws 2.0.4"))
		Expect(body).To(ContainSubstring("htmx-ext-idiomorph 0.7.4"))
	})

	It("returns 304 for matching If-None-Match", func() {
		handler := cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, cqrshtmx.HTMXExtIdiomorph)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ext/bundle.js", nil)
		req.Header.Set("If-None-Match", `"htmx-ext-bundle-sse-2.2.4,idiomorph-0.7.4"`)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotModified))
	})

	It("returns 405 for POST", func() {
		handler := cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/ext/bundle.js", nil)
		handler.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	})

	It("panics for unknown extension name in bundle", func() {
		Expect(func() {
			cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, "bogus")
		}).To(Panic())
	})

	It("panics when no names are given", func() {
		Expect(func() { cqrshtmx.HTMXExtensionsHandler() }).To(Panic())
	})
})

var _ = Describe("HTMXExtensionVersion", func() {
	DescribeTable("returns the version for known extensions",
		func(name, want string) {
			Expect(cqrshtmx.HTMXExtensionVersion(name)).To(Equal(want))
		},
		Entry("sse", cqrshtmx.HTMXExtSSE, "2.2.4"),
		Entry("ws", cqrshtmx.HTMXExtWS, "2.0.4"),
		Entry("idiomorph", cqrshtmx.HTMXExtIdiomorph, "0.7.4"),
	)

	It("returns empty string for unknown extension", func() {
		Expect(cqrshtmx.HTMXExtensionVersion("bogus")).To(Equal(""))
	})
})

var _ = Describe("HTMXExtensionNames", func() {
	It("returns all extension names sorted alphabetically", func() {
		names := cqrshtmx.HTMXExtensionNames()
		Expect(names).To(Equal([]string{"idiomorph", "sse", "ws"}))
	})

	It("returns names that join cleanly for error messages", func() {
		names := cqrshtmx.HTMXExtensionNames()
		Expect(strings.Join(names, ", ")).To(Equal("idiomorph, sse, ws"))
	})
})

var _ = Describe("HTMXExtensionCDNScriptTag", func() {
	DescribeTable("returns a CDN script tag with the embedded version",
		func(name, wantURL string) {
			Expect(cqrshtmx.HTMXExtensionCDNScriptTag(name)).To(Equal(
				`<script src="` + wantURL + `"></script>`,
			))
		},
		Entry("sse", cqrshtmx.HTMXExtSSE, "https://unpkg.com/htmx-ext-sse@2.2.4/dist/sse.min.js"),
		Entry("ws", cqrshtmx.HTMXExtWS, "https://unpkg.com/htmx-ext-ws@2.0.4/dist/ws.min.js"),
		Entry("idiomorph", cqrshtmx.HTMXExtIdiomorph, "https://unpkg.com/idiomorph@0.7.4/dist/idiomorph-ext.min.js"),
	)

	It("returns empty string for unknown extension", func() {
		Expect(cqrshtmx.HTMXExtensionCDNScriptTag("bogus")).To(Equal(""))
	})
})
