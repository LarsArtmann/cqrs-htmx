package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coverage Gaps - Notifications and Dispatch", func() {
	Describe("Notification HandlerOptions", func() {
		DescribeTable(
			"notification triggers on command success",
			func(opt cqrshtmx.HandlerOption, expectedLevel string) {
				testNotificationTrigger(opt, expectedLevel)
			},
			Entry("NotifySuccess", cqrshtmx.NotifySuccess("User created"), "success"),
			Entry("NotifyError", cqrshtmx.NotifyError("Something went wrong"), "error"),
			Entry("NotifyWarning", cqrshtmx.NotifyWarning("Check your input"), "warning"),
			Entry("NotifyInfo", cqrshtmx.NotifyInfo("Sync started"), "info"),
		)

		It("NotifyWithEvent uses custom event name", func() {
			app := newCommandAppWithHandler(noOpCommandHandler)
			r := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"/users",
				strings.NewReader(`{}`),
			)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.NotifyWithEvent("showToast").Success("User created"),
			), r)
			trigger := w.Header().Get("Hx-Trigger")
			Expect(trigger).To(ContainSubstring("showToast"))
			Expect(trigger).To(ContainSubstring("success"))
		})
	})

	Describe("Command with redirect and HTMX", func() {
		It("sets HTMX redirect for HTMX requests", func() {
			app := newCommandApp()
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			), newPostRequest("/users", `{}`, withHTMX))
			Expect(w.Header().Get("Hx-Redirect")).To(Equal("/users"))
		})
	})

	Describe("NotifyEventBuilder methods", func() {
		DescribeTable(
			"NotifyEventBuilder triggers notification",
			func(opt cqrshtmx.HandlerOption, level string) {
				testNotificationTrigger(opt, level)
			},
			Entry("Error", cqrshtmx.NotifyWithEvent("showToast").Error("Failed"), "error"),
			Entry("Warning", cqrshtmx.NotifyWithEvent("showToast").Warning("Careful"), "warning"),
			Entry("Info", cqrshtmx.NotifyWithEvent("showToast").Info("FYI"), "info"),
		)
	})

	Describe("DefaultErrorHandlerWithRedirect empty loginRedirect", func() {
		It("uses default /login when empty string is passed", func() {
			assertHTMXErrorRedirect(cqrshtmx.ErrUnauthorized, "", "/login")
		})
	})

	Describe("setTriggerWithDetail fallback merge", func() {
		It("falls back to comma when existing header is not JSON", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := cqrshtmx.NewResponse(w, r)
			w.Header().Set("Hx-Trigger", "not-json")
			resp.TriggerWithDetail("newEvent", map[string]string{"x": "1"})
			resp.Apply()
			Expect(w.Header().Get("Hx-Trigger")).To(ContainSubstring("newEvent"))
		})
	})

	Describe("NewUserID", func() {
		It("generates a non-zero user ID", func() {
			Expect(cqrshtmx.NewUserID().IsZero()).To(BeFalse())
		})

		It("generates unique IDs", func() {
			Expect(cqrshtmx.NewUserID()).NotTo(Equal(cqrshtmx.NewUserID()))
		})
	})

	Describe("WithTimeout", func() {
		It("overrides app-level timeout for a specific handler", func() {
			app := newCommandAppWithHandler(ctxCancelCommandHandler)
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(50*time.Millisecond),
			), newPostRequest("/slow", `{}`))
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})

		It("falls back to app timeout when zero", func() {
			var dispatched bool

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp, Timeout: 5 * time.Second})
			Expect(err).NotTo(HaveOccurred())
			serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(0),
			), newPostRequest("/fast", `{}`))
			Expect(dispatched).To(BeTrue())
		})
	})

	DescribeTable(
		"sanitizeRedirectURL",
		func(url string, expectRedirect bool) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect(url).Apply()

			if expectRedirect {
				Expect(w.Code).To(Equal(http.StatusSeeOther))
			} else {
				Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
			}
		},
		Entry("blocks javascript: URLs", "javascript:alert(1)", false),
		Entry("blocks absolute URLs with host", "https://evil.com", false),
		Entry("allows valid relative path", "/dashboard", true),
		Entry("allows root path", "/", true),
		Entry("blocks empty path", "", false),
		Entry("normalizes path with .. segments", "/a/../b/c", true),
		Entry("blocks data: URLs", "data:text/html,<script>alert(1)</script>", false),
		Entry("blocks scheme-relative URLs", "//evil.com", false),
		Entry("blocks unparseable URLs", "://\x00bad", false),
		Entry("blocks fragment-only URLs", "#section", false),
		Entry("blocks userinfo URLs", "http://user@host", false),
		Entry("blocks query-only URLs", "?foo=bar", false),
		Entry("blocks escape above root", "/../../../etc/passwd", false),
		Entry("allows legitimate normalization", "/a/../b/c", true),
		Entry("blocks deep traversal", "/../../../../../etc/passwd", false),
	)

	Describe("decodeFormValues multi-value fields", func() {
		It("decodes form with multi-value fields into slice", func() {
			var receivedEmail string

			app := newCommandAppWithHandler(func(_ context.Context, _ command.Command) error {
				receivedEmail = "dispatched"

				return nil
			})
			form := url.Values{}
			form.Set("Tags", "go")
			form.Add("Tags", "htmx")
			r := httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(_ struct {
					Tags []string
				},
				) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			), r)
			Expect(receivedEmail).To(Equal("dispatched"))
		})

		It("returns error for form that cannot unmarshal to target type", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			form := url.Values{}
			form.Set("Count", "not-a-number")
			r := httptest.NewRequest(http.MethodPost, "/bad", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(_ struct{ Count int }) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusNoContent))
		})
	})

	Describe("Command dispatch with AfterDispatchHook error", func() {
		It("still returns success when AfterDispatchHook fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, _ error) {},
			})
			Expect(err).NotTo(HaveOccurred())

			w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Query dispatch with PushURL", func() {
		It("sets HX-Push-URL on query success", func() {
			app := newQueryAppWithResult(testResultQueryHandler())
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
				cqrshtmx.PushURL("/users/1"),
			), r)
			Expect(w.Header().Get("Hx-Push-Url")).To(Equal("/users/1"))
		})
	})
})
