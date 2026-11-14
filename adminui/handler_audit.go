package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// auditIndex renders the audit log. In tenant-admin mode the log is global
// (usermgmt's AuditLog records user events only); future per-tenant scoping can
// filter by aggregate.
//
// The q omnibox filters across action, resolved actor email, and aggregate id
// (see filterAuditEntries) before pagination, so the page window is honest
// about the filtered set. HTMX search requests swap the table partial only.
func (h *Handler) auditIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var entries []usermgmt.AuditEntry
	total := 0
	pg := listPage{Page: 1, TotalPages: 1}
	if al := h.config.Service.AuditLog(); al != nil {
		all := al.Recent(al.Count())
		resolveAuditEmails(h.config.Service, all)
		matched := filterAuditEntries(all, q)
		total = len(matched)
		offset, limit, totalPages, page := pageBounds(parsePageQuery(r), total, listPageSize)
		entries = matched[offset : offset+limit]
		pg = listPage{Page: page, TotalPages: totalPages}
	}
	d := auditData{
		Entries: entries, Total: total, Search: q, BasePath: h.config.BasePath,
		listPage: pg,
	}

	if htmx.RenderPartial(r) {
		renderPartial(w, r, auditTableContent(d))
		return
	}
	p := h.page("Audit log", "/audit", user, r)
	renderPage(w, r, auditPage(p, d))
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
