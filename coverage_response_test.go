package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Root Coverage Gaps - Response Builder", func() {
		Describe("Response builder enhancements", func() {
			Describe("Status", func() {
				It("defers the status code to Apply()", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					resp := cqrshtmx.NewResponse(w, r)
					result := resp.Status(http.StatusCreated)
					Expect(result).To(Equal(resp))
					Expect(w.Code).To(Equal(http.StatusOK)) // not written yet
					resp.Apply()
					Expect(w.Code).To(Equal(http.StatusCreated))
				})

				It("allows Status then Redirect without breaking the chain", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					resp := cqrshtmx.NewResponse(w, r)
					resp.Status(http.StatusCreated).Redirect("/users").Apply()
					Expect(w.Code).To(Equal(http.StatusSeeOther))
					Expect(w.Header().Get("Location")).To(Equal("/users"))
				})
			})

			Describe("Header", func() {
				It("sets a custom header", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					resp := cqrshtmx.NewResponse(w, r)
					result := resp.Header("X-Custom", "value")
					Expect(result).To(Equal(resp))
					Expect(w.Header().Get("X-Custom")).To(Equal("value"))
				})
			})

			Describe("ContentType", func() {
				It("sets Content-Type header", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					resp := cqrshtmx.NewResponse(w, r)
					result := resp.ContentType("text/xml")
					Expect(result).To(Equal(resp))
					Expect(w.Header().Get("Content-Type")).To(Equal("text/xml"))
				})
			})

			Describe("JSON", func() {
				It("encodes and writes JSON body", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
					Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
					Expect(w.Body.String()).To(ContainSubstring(`"s":"ok"`))
				})
			})

			Describe("WriteString", func() {
				It("writes string body", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).WriteString("hello world")
					Expect(w.Body.String()).To(Equal("hello world"))
				})
			})

			Describe("Body", func() {
				It("writes byte body", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					result := cqrshtmx.NewResponse(w, r).Body([]byte("raw bytes"))
					Expect(result).NotTo(BeNil())
					Expect(w.Body.String()).To(Equal("raw bytes"))
				})
			})

			Describe("WriteString", func() {
				It("writes via WriteString when StringWriter is available", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).WriteString("hello world")
					Expect(w.Body.String()).To(Equal("hello world"))
				})

				It("writes via Write when StringWriter is not available", func() {
					rec := httptest.NewRecorder()
					w := &nonStringWriter{ResponseWriter: rec, recorder: rec}
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).WriteString("fallback")
					Expect(w.recorder.Body.String()).To(Equal("fallback"))
				})
			})

			Describe("JSON", func() {
				It("encodes and writes JSON body", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
					Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
					Expect(w.Body.String()).To(ContainSubstring(`"s":"ok"`))
				})

				It("returns 500 on marshal error", func() {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					cqrshtmx.NewResponse(w, r).JSON(make(chan int))
					Expect(w.Code).To(Equal(http.StatusInternalServerError))
				})
			})
		})

		Describe("WithMaxBodySize HandlerOption", func() {
			It("allows per-handler override of max body size", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{
					Commands:    disp,
					MaxBodySize: 1,
				})
				Expect(err).NotTo(HaveOccurred())

				smallBody := `{"email":"test@test.com"}`
				handler := app.Command("CreateUser", decodeBDDCreateUserJSONWithBody(), cqrshtmx.WithMaxBodySize(1024))
				w := serve(handler, newPostRequest("/users", smallBody))
				Expect(w.code()).To(Equal(http.StatusNoContent))
			})
		})

		Describe("WithSuccessStatus HandlerOption", func() {
			It("returns custom success status code", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
				Expect(err).NotTo(HaveOccurred())

				handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.WithSuccessStatus(http.StatusCreated))
				w := serve(handler, newPostRequest("/users", `{}`))
				Expect(w.code()).To(Equal(http.StatusCreated))
			})
		})

		Describe("OnError HandlerOption", func() {
			It("calls per-handler error callback on failure", func() {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
				Expect(err).NotTo(HaveOccurred())

				var capturedErr error
				handler := app.Command(
					"CreateUser",
					decodeCreateUserJSON(),
					cqrshtmx.OnError(func(_ *http.Request, err error) { capturedErr = err }),
				)
				w := serve(handler, newPostRequest("/users", `{invalid json`))
				Expect(w.code()).To(Equal(http.StatusBadRequest))
				Expect(capturedErr).To(HaveOccurred())
			})

})
