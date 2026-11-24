package adminui

import (
	"encoding/csv"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	errorfamily "github.com/larsartmann/go-error-family"
)

// auditDateLayout is the date-only wire format of the from/to filters. Dates
// are interpreted in the server's local zone: the from bound includes its
// whole day, the to bound includes its whole day (inclusive range).
const auditDateLayout = "2006-01-02"

// userActivityPageSize is how many timeline rows one activity page carries —
// the same page the user detail card opens with, so "Load more" continues the
// card's list seamlessly.
const userActivityPageSize = 8

// auditView carries everything the audit page renders: the shared list
// skeleton plus the filter inputs as submitted, the honest validation issues
// they produced, and the audit window's retention facts. (auditData stays the
// shared list struct; the audit page's extra state lives here.)
type auditView struct {
	auditData
	// From/To are the raw date-filter inputs as submitted ("" = unset); the
	// inputs re-render with them so the filter state survives the swap.
	From string
	To   string
	// Actor is the active actor-kind chip ("" = all kinds).
	Actor string
	// Issues holds one message per filter input that could not be applied
	// (invalid date, reversed range). Empty when the filters parsed clean —
	// a refused filter is announced, never silently dropped.
	Issues []string
	// Window is the audit projection's retention picture for the honesty
	// caption. Zero facts render no caption.
	Window auditWindowFacts
}

// auditWindowFacts is what the in-memory audit window can honestly claim:
// how many entries it currently holds, its capacity, and how many older
// entries have rolled off. The journal remains the durable record; the
// caption says so in the audit-window vocabulary.
type auditWindowFacts struct {
	Holds   int `json:"holds"`   // entries currently retained
	Limit   int `json:"limit"`   // window capacity
	Dropped int `json:"dropped"` // entries evicted since construction
}

// hasFacts reports whether real window facts were captured (the panel may run
// without an audit log at all).
func (w auditWindowFacts) hasFacts() bool { return w.Limit > 0 }

// auditWindowCaptionText builds the retention-honesty caption in the audit
// window's own vocabulary: what the window holds, what has rolled off, and
// where the durable record lives.
func auditWindowCaptionText(w auditWindowFacts) string {
	var b strings.Builder
	b.WriteString("Audit window holds ")
	b.WriteString(strconv.Itoa(w.Holds))
	b.WriteString(" entries")
	if w.Dropped > 0 {
		b.WriteString(" · ")
		b.WriteString(strconv.Itoa(w.Dropped))
		b.WriteString(" older entries have rolled off")
	}
	b.WriteString(" · the journal keeps the full history.")
	return b.String()
}

// auditFiltersActive reports whether any filter narrows the view — the empty
// state distinguishes "nothing recorded at all" from "nothing matches the
// active filters" instead of pretending the window is empty.
func auditFiltersActive(d auditView) bool {
	return d.Search != "" || d.From != "" || d.To != "" || d.Actor != ""
}

// auditDateInputClass is the class string for the from/to date inputs — the
// same input styling the omnibox uses, minus the search glyph's end padding.
func auditDateInputClass() string {
	return "block w-full max-w-[10rem] rounded-md border-0 py-1.5 ps-3 pe-3 text-sm text-gray-900 dark:text-white bg-white dark:bg-gray-800 ring-1 ring-inset ring-gray-300 dark:ring-gray-600 focus:ring-2 focus:ring-inset focus:ring-blue-600 dark:focus:ring-blue-500"
}

// auditIndex renders the audit log. In tenant-admin mode the log is global
// (usermgmt's AuditLog records user events only); future per-tenant scoping
// can filter by aggregate.
//
// The filters narrow the matched set BEFORE pagination, so the page window is
// honest about the filtered set: the q omnibox matches across action, resolved
// actor email, and aggregate id (see filterAuditEntries); from/to narrow the
// inclusive date range; the actor chip narrows the actor kind. A filter input
// that cannot be parsed renders an explicit issue (never a silent drop), and
// every active filter is carried through the pager, the chips, and the HTMX
// panel swaps.
func (h *Handler) auditIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))

	from, fromIssue := parseAuditDateFilter(fromRaw, "From")
	to, toIssue := parseAuditDateFilter(toRaw, "To")
	issues := make([]string, 0, 3)
	for _, issue := range []string{fromIssue, toIssue} {
		if issue != "" {
			issues = append(issues, issue)
		}
	}
	if from != nil && to != nil && from.After(*to) {
		issues = append(issues, "From date is after To date — the date range was ignored.")
		from, to = nil, nil
	}
	actor := parseAuditActorChip(r.URL.Query().Get("actor"))

	var entries []usermgmt.AuditEntry
	total := 0
	pg := listPage{Page: 1, TotalPages: 1}
	var window auditWindowFacts
	if al := h.config.Service.AuditLog(); al != nil {
		all := al.Recent(al.Count())
		resolveAuditEmails(h.config.Service, all)
		matched := filterAuditActor(filterAuditRange(filterAuditEntries(all, q), from, to), actor)
		total = len(matched)
		offset, limit, totalPages, page := pageBounds(parsePageQuery(r), total, h.config.PageSize)
		entries = matched[offset : offset+limit]
		pg = listPage{Page: page, TotalPages: totalPages}
		window = auditWindowFacts{Holds: al.Count(), Limit: al.Limit(), Dropped: al.Dropped()}
	}
	d := auditView{
		Entries: entries, Total: total, Search: q, BasePath: h.config.BasePath,
		listPage: pg,
		From:     fromRaw,
		To:       toRaw,
		Actor:    actor,
		Issues:   issues,
		Window:   window,
	}

	p := h.page("Audit log", "/audit", user, r)
	if htmx.RenderPartial(r) {
		renderPartial(w, r, auditPanelContent(p, d))
		return
	}
	renderPage(w, r, auditPage(p, d))
}

// parseAuditDateFilter parses one date-only filter bound. An empty input
// means the bound is unset; a malformed date yields an honest issue and NO
// bound — the filter half is ignored loudly (the issue names which one), not
// silently.
func parseAuditDateFilter(raw, label string) (*time.Time, string) {
	if raw == "" {
		return nil, ""
	}
	// time.Now().Location() IS time.Local — spelled this way because the
	// audit window is deliberately server-local (the server's own clock
	// defines its days), never a request's or a user's zone.
	day, err := time.ParseInLocation(auditDateLayout, raw, time.Now().Location())
	if err != nil {
		return nil, label + " date is not a valid date — use YYYY-MM-DD; that bound was ignored."
	}
	return &day, ""
}

// auditActorKinds is the actor-kind chip vocabulary — exactly the spelling
// of the audit entries' own actor kinds (id.ActorKind.String(): user, bot,
// system, service).
func auditActorKinds() []string {
	return []string{"user", "bot", "system", "service"}
}

// parseAuditActorChip maps the chip query param to the vocabulary; an unknown
// value degrades to "" (every kind) — the same unknown-chip degradation the
// users status filter uses. The degrade is observable: no chip renders active
// and every row shows.
func parseAuditActorChip(raw string) string {
	for _, chip := range auditActorKinds() {
		if raw == chip {
			return chip
		}
	}
	return ""
}

// auditChip is one rendered filter chip on the audit page (the users index
// chip shape).
type auditChip struct {
	Label  string
	Href   string
	Active bool
}

// auditActorChipList renders the actor-kind chip list: All + one chip per
// kind. The active chip toggles OFF (keeping q and the date range), the
// others toggle their own kind ON.
func auditActorChipList(d auditView) []auditChip {
	kinds := auditActorKinds()
	chips := make([]auditChip, 0, len(kinds)+1)
	chips = append(chips, auditChip{
		Label:  "All actors",
		Href:   auditFilterURL(d.BasePath+"/audit", d.Search, d.From, d.To, ""),
		Active: d.Actor == "",
	})
	for _, kind := range auditActorKinds() {
		chips = append(chips, auditChip{
			Label:  auditActorChipLabel(kind),
			Href:   auditFilterURL(d.BasePath+"/audit", d.Search, d.From, d.To, kind),
			Active: d.Actor == kind,
		})
	}
	return chips
}

// auditActorChipLabel capitalizes a kind for display. The wire value stays
// the entries' own spelling; only the chip label reads human.
func auditActorChipLabel(kind string) string {
	switch kind {
	case "user":
		return "Users"
	case "bot":
		return "Bots"
	case "system":
		return "System"
	case "service":
		return "Services"
	}
	return kind
}

// auditListURL builds the audit page URL carrying every active filter, so the
// pager links keep the narrowed view (the pagination component appends or
// replaces only the page parameter).
func auditListURL(base string, d auditView) string {
	return auditFilterURL(base, d.Search, d.From, d.To, d.Actor)
}

// auditFilterURL is the one query-string builder behind the pager, the chips,
// and the omnibox wiring: q plus from/to/actor, omitting empties. The omnibox
// typing path bakes the non-q filters into the wired URL (its element value
// is the only field it serializes), so a keystroke search cannot silently
// drop the date range or the actor chip.
func auditFilterURL(base, search, from, to, actor string) string {
	params := url.Values{}
	if search != "" {
		params.Set("q", search)
	}
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}
	if actor != "" {
		params.Set("actor", actor)
	}
	if len(params) == 0 {
		return base
	}
	return base + "?" + params.Encode()
}

// auditExportCSV / auditExportJSON stream the audit WINDOW's filtered
// entries in the declared format — the same q/from/to/actor narrowing the
// page applies (one filter order, filterAuditEntries → range → actor), over
// the whole retained window, not just the visible page. The response leads
// with an honest caption: the window's retention facts and the filter state
// the file was narrowed by, so an exported file can be read without the
// page next to it.
func (h *Handler) auditExportCSV(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.auditExport(w, r, usermgmt.UserDataFormatCSV)
}

func (h *Handler) auditExportJSON(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.auditExport(w, r, usermgmt.UserDataFormatJSON)
}

func (h *Handler) auditExport(w http.ResponseWriter, r *http.Request, format usermgmt.UserDataFormat) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	from, _ := parseAuditDateFilter(fromRaw, "From")
	to, _ := parseAuditDateFilter(toRaw, "To")
	// A reversed range narrows nothing (the page announces the issue; an
	// export has no issue strip, so it simply exports un-date-narrowed).
	if from != nil && to != nil && from.After(*to) {
		from, to = nil, nil
	}
	actor := parseAuditActorChip(r.URL.Query().Get("actor"))

	var window auditWindowFacts
	var matched []usermgmt.AuditEntry
	if al := h.config.Service.AuditLog(); al != nil {
		all := al.Recent(al.Count())
		resolveAuditEmails(h.config.Service, all)
		matched = filterAuditActor(filterAuditRange(filterAuditEntries(all, q), from, to), actor)
		window = auditWindowFacts{Holds: al.Count(), Limit: al.Limit(), Dropped: al.Dropped()}
	}

	filters := auditExportFilters{Q: q, From: fromRaw, To: toRaw, Actor: actor}
	name := "audit-export." + string(format)
	w.Header().Set("Content-Type", exportContentType(format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	var err error
	switch format {
	case usermgmt.UserDataFormatCSV:
		err = writeAuditCSV(w, matched, filters, window)
	case usermgmt.UserDataFormatJSON:
		err = writeAuditJSON(w, matched, filters, window)
	}
	if err != nil {
		// Headers are already committed by the streaming write; the partial
		// output is honest about failing rather than silently truncated.
		triggerToast(w, "err", "Export failed: "+actionErrorMessage(err))
	}
}

// auditExportFilters is the filter state an export was narrowed by, echoed
// into the file so its scope is self-describing.
type auditExportFilters struct {
	Q     string `json:"q,omitzero"`
	From  string `json:"from,omitzero"`
	To    string `json:"to,omitzero"`
	Actor string `json:"actor,omitzero"`
}

// active reports whether any filter narrowed the export (mirrors the page's
// auditFiltersActive).
func (f auditExportFilters) active() bool {
	return f.Q != "" || f.From != "" || f.To != "" || f.Actor != ""
}

// describe renders the filter state as one human caption line; an unfiltered
// export says so instead of staying quiet.
func (f auditExportFilters) describe() string {
	if !f.active() {
		return "Filters: none — every entry the window retains."
	}
	var b strings.Builder
	b.WriteString("Filters:")
	if f.Q != "" {
		b.WriteString(" q=" + f.Q)
	}
	if f.From != "" {
		b.WriteString(" from=" + f.From)
	}
	if f.To != "" {
		b.WriteString(" to=" + f.To)
	}
	if f.Actor != "" {
		b.WriteString(" actor=" + f.Actor)
	}
	return b.String()
}

// writeAuditCSV streams the filtered entries as CSV led by "# " comment
// lines: the window caption, the filter state, and the row count — a file
// that says what it covers before the first data row.
func writeAuditCSV(
	w io.Writer,
	entries []usermgmt.AuditEntry,
	filters auditExportFilters,
	window auditWindowFacts,
) error {
	cw := csv.NewWriter(w)
	caption := []string{
		"Audit export — filtered view of the in-memory audit window.",
		filters.describe(),
	}
	if window.hasFacts() {
		caption = append(caption, auditWindowCaptionText(window))
	}
	caption = append(caption, strconv.Itoa(len(entries))+" entries exported.")
	for _, line := range caption {
		if err := cw.Write([]string{"# " + line}); err != nil {
			return errorfamily.WrapTransient(
				err,
				"adminui.audit_export.csv_caption_failed",
				"write audit export caption",
			)
		}
	}
	if err := cw.Write(
		[]string{"action", "actor_email", "aggregate_id", "occurred_at", "actor_kind", "detail"},
	); err != nil {
		return errorfamily.WrapTransient(err, "adminui.audit_export.csv_header_failed", "write audit export header")
	}
	for _, e := range entries {
		if err := cw.Write([]string{
			e.Action,
			e.Email,
			e.AggregateID.Get(),
			e.OccurredAt.Format(time.RFC3339),
			e.ActorID.Kind().String(),
			e.Detail,
		}); err != nil {
			return errorfamily.WrapTransient(err, "adminui.audit_export.csv_row_failed", "write audit export row")
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return errorfamily.WrapTransient(err, "adminui.audit_export.csv_flush_failed", "flush audit export CSV")
	}
	return nil
}

// writeAuditJSON streams the filtered entries as one envelope object: the
// window facts, the filter state, and the count travel WITH the entries, so
// a reader can tell what the file covers without the page.
func writeAuditJSON(
	w io.Writer,
	entries []usermgmt.AuditEntry,
	filters auditExportFilters,
	window auditWindowFacts,
) error {
	doc := auditExportDoc{
		Window:  window,
		Filters: filters,
		Count:   len(entries),
		Entries: entries,
	}
	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))
	if err := json.MarshalEncode(enc, doc); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"adminui.audit_export.json_encode_failed",
			"encode audit export JSON",
		)
	}
	return nil
}

// auditExportDoc is the JSON export envelope: what the window can honestly
// claim, what narrowed the file, and the entries themselves.
type auditExportDoc struct {
	Window  auditWindowFacts      `json:"window,omitzero"`
	Filters auditExportFilters    `json:"filters"`
	Count   int                   `json:"count"`
	Entries []usermgmt.AuditEntry `json:"entries"`
}

// userActivity serves the user detail activity card's load-more tail
// (GET /users/{id}/activity?skip=N): the NEXT chunk of the card's timeline as
// a fragment that replaces the card's load-more row, so the appended rows
// land directly under the existing ones. Entries run in the card's order (the
// audit window's chronological order), so a page continues the visible list;
// the fragment carries either the next load-more row or the honest
// end-of-window note. Same posture as the user detail route it extends:
// session+role gated, read-only, super-admin only.
func (h *Handler) userActivity(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return
	}
	shown, err := h.config.Service.GetUser(r.Context(), target)
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusNotFound,
			"User not found",
			"No user with this id exists (it may have been deleted).",
		)
		return
	}
	// The fragment carries the label table + locale so an override-configured
	// deployment re-words the swapped-in rows too (T16).
	renderPartial(w, r, userActivityFragment(h.page("", "/users", user, r),
		h.userActivityData(shown, parseActivitySkip(r.URL.Query().Get("skip")))))
}

// userActivityPage is one load-more fragment's data: the next rows, the skip
// cursor for the page AFTER them, and whether the audit window is exhausted.
type userActivityPage struct {
	UserID   string // the user's id, for the next load-more URL
	BasePath string
	Entries  []usermgmt.AuditEntry // the rows this fragment appends
	NextSkip int                   // cursor the next load-more row fetches
	Done     bool                  // true when the window holds nothing further
	Total    int                   // events the window retains for this user
}

// userActivityData pages one user's audit stream in the card's order (the
// window's chronological order, matching what userTimeline rendered first)
// and resolves actor emails like every audit surface.
func (h *Handler) userActivityData(shown *identitymodel.User, skip int) userActivityPage {
	d := userActivityPage{
		UserID:   shown.ID.Get().String(),
		BasePath: h.config.BasePath,
		Entries:  nil,
		NextSkip: 0,
		Done:     false,
		Total:    0,
	}
	al := h.config.Service.AuditLog()
	if al == nil {
		d.Done = true
		return d
	}
	all := al.Recent(al.Count())
	entries := make([]usermgmt.AuditEntry, 0, userActivityPageSize)
	for _, e := range all {
		if e.AggregateID.Get() != d.UserID {
			continue
		}
		entries = append(entries, e)
	}
	d.Total = len(entries)
	resolveAuditEmails(h.config.Service, entries)
	if skip >= len(entries) {
		d.Done = true
		return d
	}
	end := min(skip+userActivityPageSize, len(entries))
	d.Entries = entries[skip:end]
	d.NextSkip = end
	d.Done = end >= len(entries)
	return d
}

// parseActivitySkip reads the "skip" cursor: missing, malformed, or negative
// values start at the beginning of the tail — the same degrade
// parsePageQuery applies to page numbers (our own links emit the cursor, so
// a malformed value is tampering, not intent).
func parseActivitySkip(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// userActivityURL builds the load-more endpoint URL for one user's timeline
// at a skip cursor.
func userActivityURL(basePath, userID string, skip int) string {
	return basePath + "/users/" + userID + "/activity?skip=" + strconv.Itoa(skip)
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
