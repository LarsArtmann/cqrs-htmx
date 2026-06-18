package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// htmxScriptTestCase describes one HTTP exchange to be exercised against
// either the embedded HTMXScriptHandler or a custom HTMXScriptHandlerWith.
type htmxScriptTestCase struct {
	name             string
	method           string
	js               []byte
	version          string
	etag             string
	ifNoneMatch      string
	wantStatus       int
	wantBodyNonEmpty bool
}

// htmxCaseBuilder provides a fluent, table-friendly way to build
// htmxScriptTestCase entries. The default values match the embedded
// HTMXScriptHandler; callers override the knobs that vary between
// cases (method, version, ifNoneMatch, wantStatus, name).
type htmxCaseBuilder struct {
	name             string
	method           string
	js               []byte
	version          string
	etag             string
	ifNoneMatch      string
	wantStatus       int
	wantBodyNonEmpty bool
}

func newCase() htmxCaseBuilder {
	return htmxCaseBuilder{
		name:             "",
		method:           "",
		js:               nil,
		version:          "2.0.9",
		etag:             `"htmx-2.0.9"`,
		ifNoneMatch:      "",
		wantStatus:       0,
		wantBodyNonEmpty: false,
	}
}

func (b htmxCaseBuilder) custom() htmxCaseBuilder {
	b.js = []byte("// custom htmx build")
	b.version = "4.0.0-beta4"
	b.etag = `"htmx-4.0.0-beta4"`
	return b
}

func (b htmxCaseBuilder) withMethod(method string) htmxCaseBuilder {
	b.method = method
	return b
}

func (b htmxCaseBuilder) withIfNoneMatch(header string) htmxCaseBuilder {
	b.ifNoneMatch = header
	return b
}

func (b htmxCaseBuilder) withName(name string) htmxCaseBuilder {
	b.name = name
	return b
}

func (b htmxCaseBuilder) withWantStatus(status int) htmxCaseBuilder {
	b.wantStatus = status
	return b
}

func (b htmxCaseBuilder) withBody() htmxCaseBuilder {
	b.wantBodyNonEmpty = true
	return b
}

func (b htmxCaseBuilder) build() htmxScriptTestCase {
	return htmxScriptTestCase(b)
}

func htmxScriptTestCases() []htmxScriptTestCase {
	custom := newCase().custom()
	return []htmxScriptTestCase{
		newCase().
			withName("embedded GET serves body with correct content type and ETag").
			withMethod(http.MethodGet).
			withWantStatus(http.StatusOK).
			withBody().
			build(),
		newCase().
			withName("embedded GET returns 304 for matching If-None-Match").
			withMethod(http.MethodGet).
			withIfNoneMatch(`"htmx-2.0.9"`).
			withWantStatus(http.StatusNotModified).
			build(),
		custom.
			withName("custom GET serves body with correct content type and ETag").
			withMethod(http.MethodGet).
			withWantStatus(http.StatusOK).
			build(),
		custom.
			withName("custom GET returns 304 for matching If-None-Match").
			withMethod(http.MethodGet).
			withIfNoneMatch(`"htmx-4.0.0-beta4"`).
			withWantStatus(http.StatusNotModified).
			build(),
		custom.
			withName("custom GET does not return 304 for non-matching If-None-Match").
			withMethod(http.MethodGet).
			withIfNoneMatch(`"htmx-2.0.9"`).
			withWantStatus(http.StatusOK).
			build(),
		newCase().
			withName("embedded HEAD returns 200").
			withMethod(http.MethodHead).
			withWantStatus(http.StatusOK).
			build(),
		custom.
			withName("custom HEAD returns 200").
			withMethod(http.MethodHead).
			withWantStatus(http.StatusOK).
			build(),
		newCase().
			withName("embedded POST returns 405").
			withMethod(http.MethodPost).
			withWantStatus(http.StatusMethodNotAllowed).
			build(),
		custom.
			withName("custom POST returns 405").
			withMethod(http.MethodPost).
			withWantStatus(http.StatusMethodNotAllowed).
			build(),
		newCase().
			withName("embedded PUT returns 405").
			withMethod(http.MethodPut).
			withWantStatus(http.StatusMethodNotAllowed).
			build(),
	}
}

func runHTMXScriptCase(tc htmxScriptTestCase) {
	var handler http.Handler
	if tc.js == nil {
		handler = cqrshtmx.HTMXScriptHandler()
	} else {
		handler = cqrshtmx.HTMXScriptHandlerWith(tc.js, tc.version)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(tc.method, "/htmx.js", nil)
	if tc.ifNoneMatch != "" {
		req.Header.Set("If-None-Match", tc.ifNoneMatch)
	}
	handler.ServeHTTP(w, req)

	Expect(w.Code).To(Equal(tc.wantStatus))
	if tc.wantStatus == http.StatusMethodNotAllowed {
		return
	}
	Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
	Expect(w.Header().Get("Content-Type")).To(Equal("text/javascript; charset=utf-8"))
	Expect(w.Header().Get("ETag")).To(Equal(tc.etag))
	if tc.wantBodyNonEmpty {
		Expect(w.Body.Len()).To(BeNumerically(">", 0))
	} else if tc.wantStatus == http.StatusNotModified {
		Expect(w.Body.Len()).To(Equal(0))
	}
}

var _ = Describe("HTMXScriptHandler", func() {
	for _, tc := range htmxScriptTestCases() {
		It(tc.name, func() {
			runHTMXScriptCase(tc)
		})
	}
})

var _ = Describe("HTMXVersion", func() {
	It("returns the version string", func() {
		Expect(cqrshtmx.HTMXVersion()).To(Equal("2.0.9"))
	})
})

var _ = Describe("HTMXScriptTag", func() {
	It("returns a script tag with the given path", func() {
		Expect(cqrshtmx.HTMXScriptTag("/static/htmx.js")).To(
			Equal(`<script src="/static/htmx.js"></script>`),
		)
	})

	It("returns a script tag with a different path", func() {
		Expect(cqrshtmx.HTMXScriptTag("/assets/js/htmx.js")).To(
			Equal(`<script src="/assets/js/htmx.js"></script>`),
		)
	})
})

var _ = Describe("HTMXCDNScriptTag", func() {
	It("returns a CDN script tag with a specific version", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("4.0.0")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@4.0.0"></script>`,
		))
	})

	It("uses the embedded version when version is empty", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@2.0.9"></script>`,
		))
	})

	It("supports pre-release versions", func() {
		Expect(cqrshtmx.HTMXCDNScriptTag("4.0.0-beta4")).To(Equal(
			`<script src="https://unpkg.com/htmx.org@4.0.0-beta4"></script>`,
		))
	})
})
