package cqrshtmx_test

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handler Options", func() {
	Describe("RequireAuth", func() {
		It("rejects requests without user ID", func() {
			app := newCommandApp()
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.RequireAuth(),
				decodeCreateUserJSON(),
			), newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})
	})
})
