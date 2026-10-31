package cqrshtmx_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StructuredError", func() {
	Describe("NewStructuredError", func() {
		It("maps a Rejection error to 400 with correct fields", func() {
			err := cqrshtmx.ErrValidationFailed
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusBadRequest))
			Expect(se.Type).To(Equal("rejection"))
			Expect(se.Detail).To(ContainSubstring("validation failed"))
			Expect(se.Instance).To(BeEmpty())
		})

		It("maps a generic error to Transient (503)", func() {
			err := errors.New("something broke")
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusServiceUnavailable))
			// 5xx detail is redacted to the family's public-safe default message.
			Expect(se.Detail).To(Equal("A temporary error occurred. Please try again in a few moments."))
			Expect(se.Detail).NotTo(ContainSubstring("something broke"))
			Expect(se.Title).To(Equal("Service Unavailable"))
			Expect(se.Type).To(Equal("transient"))
			// Family metadata (RFC 7807 extensions) is exposed.
			Expect(se.Message).To(Equal(se.Detail))
			Expect(se.Fix).NotTo(BeEmpty())
		})

		It("extracts request ID from the request context", func() {
			rid := cqrshtmx.NewRequestID()

			ctx := cqrshtmx.WithRequestID(context.Background(), rid)
			r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil).WithContext(ctx)

			se := cqrshtmx.NewStructuredError(errors.New("fail"), r)
			Expect(se.Instance).To(Equal(rid.String()))
		})

		It("returns zero value when err is nil", func() {
			se := cqrshtmx.NewStructuredError(nil, nil)
			Expect(se.Status).To(Equal(0))
			Expect(se.Detail).To(BeEmpty())
		})

		It("maps Conflict family to 409", func() {
			err := errorfamily.NewConflict("email_taken", "email already registered")
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusConflict))
			Expect(se.Type).To(Equal("conflict"))
			Expect(se.Title).To(Equal("Conflict"))
		})

		It("populates Code from the error's Code() method", func() {
			err := cqrshtmx.ErrValidationFailed
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Code).To(Equal("validation_failed"))
		})

		It("includes code in JSON output", func() {
			err := cqrshtmx.ErrValidationFailed
			se := cqrshtmx.NewStructuredError(err, nil)

			var decoded map[string]any
			Expect(json.Unmarshal([]byte(se.JSON()), &decoded)).To(Succeed())
			Expect(decoded["code"]).To(Equal("validation_failed"))
		})

		It("maps Transient family to 503", func() {
			err := cqrshtmx.ErrDispatchFailed
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusServiceUnavailable))
			Expect(se.Type).To(Equal("transient"))
		})
	})

	Describe("NewStructuredErrorWithContext", func() {
		It("extracts request ID from context", func() {
			rid := cqrshtmx.NewRequestID()

			ctx := cqrshtmx.WithRequestID(context.Background(), rid)
			se := cqrshtmx.NewStructuredErrorWithContext(errors.New("fail"), ctx)

			Expect(se.Instance).To(Equal(rid.String()))
		})

		It("returns zero value when err is nil", func() {
			se := cqrshtmx.NewStructuredErrorWithContext(nil, context.Background())
			Expect(se.Status).To(Equal(0))
		})
	})

	Describe("JSON", func() {
		It("serializes to valid JSON with all fields", func() {
			se := cqrshtmx.StructuredError{
				Type:     "rejection",
				Title:    "Bad Request",
				Status:   400,
				Detail:   "invalid email",
				Instance: "req-123",
			}

			jsonStr := se.JSON()

			var decoded map[string]any
			Expect(json.Unmarshal([]byte(jsonStr), &decoded)).To(Succeed())

			Expect(decoded["type"]).To(Equal("rejection"))
			Expect(decoded["title"]).To(Equal("Bad Request"))
			Expect(decoded["status"]).To(Equal(float64(400)))
			Expect(decoded["detail"]).To(Equal("invalid email"))
			Expect(decoded["instance"]).To(Equal("req-123"))
		})

		It("omits empty instance field", func() {
			se := cqrshtmx.StructuredError{
				Type:     "rejection",
				Title:    "Bad Request",
				Status:   400,
				Detail:   "invalid email",
				Instance: "",
			}

			var decoded map[string]any
			Expect(json.Unmarshal([]byte(se.JSON()), &decoded)).To(Succeed())
			_, hasInstance := decoded["instance"]
			Expect(hasInstance).To(BeFalse())
		})
	})

	Describe("JSON round-trip", func() {
		It("marshals and unmarshals correctly", func() {
			original := cqrshtmx.StructuredError{
				Type:     "conflict",
				Title:    "Conflict",
				Status:   409,
				Detail:   "email already exists",
				Instance: "req-456",
			}

			data, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var decoded cqrshtmx.StructuredError
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())

			Expect(decoded).To(Equal(original))
		})
	})

	Describe("Context (RFC 7807 extension, round-45 M13)", func() {
		It("carries machine context to API clients for client-fault statuses", func() {
			base := errorfamily.Wrapf(cqrshtmx.ErrRequestTooLarge, errorfamily.Rejection,
				"cqrshtmx.decode.body.too_large", "body of 200 bytes exceeds limit")
			err := base.WithContextAny("max_body_size", 100)
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(se.Context).To(HaveKeyWithValue("max_body_size", "100"))
		})

		It("carries form_key context for ambiguous form keys", func() {
			err := errorfamily.NewRejection("cqrshtmx.decode.form.key_collision",
				"ambiguous form key").WithContext("form_key", "Name")
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Context).To(HaveKeyWithValue("form_key", "Name"))
		})

		It("redacts context on server faults (5xx), mirroring the Detail rule", func() {
			err := errorfamily.NewInfrastructure("cqrshtmx.test.internal",
				"internal failure").WithContext("user_id", "u-123")
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(BeNumerically(">=", 500))
			Expect(se.Context).To(BeEmpty(), "5xx context entries are internal diagnostics and must not reach the wire")
		})

		It("omits the context field entirely when the error carries none", func() {
			data, err := json.Marshal(cqrshtmx.NewStructuredError(cqrshtmx.ErrValidationFailed, nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(
				string(data),
			).NotTo(ContainSubstring("context"), "omitempty must keep the wire shape unchanged for context-less errors")
		})
	})
})
