package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable(
	"notification methods",
	func(notify func(*cqrshtmx.Response) *cqrshtmx.Response, level, message string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		notify(cqrshtmx.NewResponse(w, r)).Apply()
		trigger := w.Header().Get("HX-Trigger")
		Expect(trigger).To(ContainSubstring(level))
		Expect(trigger).To(ContainSubstring(message))
	},
	Entry(
		"NotifySuccess",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifySuccess("Created") },
		"success",
		"Created",
	),
	Entry(
		"NotifyError",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyError("Failed") },
		"error",
		"Failed",
	),
	Entry(
		"NotifyWarning",
		func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyWarning("Warn") },
		"warning",
		"Warn",
	),
	Entry("NotifyInfo", func(r *cqrshtmx.Response) *cqrshtmx.Response { return r.NotifyInfo("FYI") }, "info", "FYI"),
)
