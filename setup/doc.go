// Package setup provides a one-call composition root for cqrs-htmx applications.
//
// It eliminates the boilerplate of manually wiring usermgmt.Service, adminui.Handler,
// dashboardui.Dashboard, and loginpage.Handler — plus the middleware chain that connects them.
//
// # Quick start
//
//	bundle, err := setup.New(setup.Config{
//	    Title:   "My App",
//	    TOTP:    totp.New(totp.Config{Issuer: "MyApp"}),
//	})
//	if err != nil { log.Fatal(err) }
//	defer bundle.Close()
//
//	mux := http.NewServeMux()
//	bundle.Mount(mux)
//	http.ListenAndServe(":8080", bundle.Middleware()(mux))
//
// # What you get
//
//   - /auth/* — registration, login (WebAuthn/TOTP/OAuth2), logout, me
//   - /admin/* — admin dashboard (user/tenant/membership management)
//   - /dashboard/* — CQRS/ES observability (events, projections, DLQ, time-travel)
//   - / — login page
//
// # Customization
//
// Every sub-component is exposed on the [Bundle] struct. Override panels, add custom
// routes, or swap middleware after construction:
//
//	bundle.Admin.SetAccentColor("#ff0000")
//	mux.Handle("POST /orders", bundle.SessionMiddleware(app.Command("CreateOrder", ...)))
//
// # Persistence
//
// By default, everything runs in-memory (lost on restart). Provide your own [event.Store]
// or SQL database for production:
//
//	bundle, err := setup.New(setup.Config{
//	    EventStore:  mySQLStore,
//	    ReadModelDB: db,
//	})
package setup
