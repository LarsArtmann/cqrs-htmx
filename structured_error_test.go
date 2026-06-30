package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
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
			err := event.NewConflict("email_taken", "email already registered")
			se := cqrshtmx.NewStructuredError(err, nil)

			Expect(se.Status).To(Equal(http.StatusConflict))
			Expect(se.Type).To(Equal("conflict"))
			Expect(se.Title).To(Equal("Conflict"))
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
			se := cqrshtmx.StructuredError{ //nolint:exhaustruct // fixture: tests specific JSON fields
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
			se := cqrshtmx.StructuredError{ //nolint:exhaustruct // fixture: tests instance omitempty
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
			original := cqrshtmx.StructuredError{ //nolint:exhaustruct // fixture: round-trip
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
})
