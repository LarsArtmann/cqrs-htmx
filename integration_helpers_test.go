package cqrshtmx_test

import (
	"net/http"
	"time"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	. "github.com/onsi/gomega"
)

const (
	testUserJSON = `{"email":"alice@example.com","name":"Alice"}`
	emailKey     = "email"
)

func integrationCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		MaxAge:       24 * time.Hour,
		Secure:       false,
		SameSite:     http.SameSiteLaxMode,
		Path:         "/",
		ErrorHandler: cqrshtmx.ForbiddenErrorHandler,
	}
}

func csrfTokenHandler(csrfMW func(http.Handler) http.Handler, tokenOut *string) http.Handler {
	return csrfMW(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*tokenOut = cqrshtmx.CSRFTokenFromContext(r.Context())
	}))
}

func csrfOKHandler(csrfMW func(http.Handler) http.Handler) http.Handler {
	return csrfMW(okHandler())
}

func newIntegrationApp(
	disp *command.Dispatcher,
	enf *casbin.Enforcer,
) (*cqrshtmx.App, *command.Dispatcher) {
	_ = disp.Register("CreateUser", noOpCommandHandler)
	_ = disp.Register("DeleteUser", rejectionHandler("user.not_found", "user does not exist"))
	cfg := cqrshtmx.Config{
		Commands:        disp,
		Enforcer:        enf,
		UserIDExtractor: headerExtractor("X-User"),
	}
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app, disp
}
