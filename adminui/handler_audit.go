package adminui

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// auditIndex renders the audit log. In tenant-admin mode the log is global
// (usermgmt's AuditLog records user events only); future per-tenant scoping can
// filter by aggregate.
func (h *Handler) auditIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	var entries []usermgmt.AuditEntry
	if al := h.cfg.Service.AuditLog(); al != nil {
		entries = al.Recent(100)
	}
	p := h.page("Audit log", "/audit", user, r)
	renderPage(w, r, auditPage(p, auditData{Entries: entries, BasePath: h.cfg.BasePath}))
}
