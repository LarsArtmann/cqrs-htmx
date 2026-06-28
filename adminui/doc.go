// Package adminui provides a ready-made, good-looking Admin Dashboard for
// applications built on [github.com/larsartmann/cqrs-htmx/usermgmt/v3].
//
// It renders a complete HTMX-driven management UI — dashboard, users, tenants,
// tenant members, and an audit log — backed by a [*usermgmt.Service].
// Consumers mount it with a single call:
//
//	svc, _ := usermgmt.NewService(cfg)
//	panel := adminui.New(adminui.Config{Service: svc})
//	panel.Mount(mux, "/admin")
//
// The panel is intended to sit behind the consumer's session middleware
// (e.g. [usermgmt.NewSessionMiddleware]) so that [*usermgmt.User] is present in
// the request context. Access is gated by [Config.Authorizer].
//
// # Two scopes
//
//   - Super Admin (default): a global view of every user, tenant, and audit
//     event. Best for platform operators.
//   - Tenant Admin: a scoped view limited to a single tenant
//     ([Config.TenantID]). Best for per-customer admin sub-panels. Only the
//     dashboard, members, and audit sections are shown.
//
// # Design
//
// All markup is authored in templ and compiled to Go (the generated _templ.go
// files are committed, so consumers never run the templ generator). A modern
// embedded stylesheet ([assets/admin-tw.css]) provides the look, with automatic
// light/dark theming. No JavaScript framework — just HTMX, Tailwind v4, and a binary.
//
//go:generate templ generate
package adminui
