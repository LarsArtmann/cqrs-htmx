package adminui

import (
	"net/http"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// auditIndex renders the audit log. In tenant-admin mode the log is global
// (usermgmt's AuditLog records user events only); future per-tenant scoping can
// filter by aggregate.
func (h *Handler) auditIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	var entries []usermgmt.AuditEntry
	if al := h.config.Service.AuditLog(); al != nil {
		entries = al.Recent(100)
		resolveAuditEmails(h.config.Service, entries)
	}
	p := h.page("Audit log", "/audit", user, r)
	renderPage(w, r, auditPage(p, auditData{Entries: entries, BasePath: h.config.BasePath}))
}

// resolveAuditEmails fills in the human email for audit entries recorded
// without one: the audit projection cannot see user emails, so the view
// resolves them from the user read model (events about deleted users keep
// the raw aggregate ID).
func resolveAuditEmails(svc *usermgmt.Service, entries []usermgmt.AuditEntry) {
	for i := range entries {
		if entries[i].Email != "" {
			continue
		}
		if u, ok := svc.ReadModel().FindByID(entries[i].AggregateID); ok {
			entries[i].Email = u.Email
		}
	}
}
