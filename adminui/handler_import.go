package adminui

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// maxImportUploadBytes caps the uploaded import file. A CSV/JSON row is
// ~50 bytes; 2 MiB admits tens of thousands of users while a runaway or
// hostile upload is refused before the parser reads it (413, not OOM).
const maxImportUploadBytes = 2 << 20

// maxRenderedImportErrors bounds the per-row error list the result panel
// renders. The Service's ImportResult already carries every error; showing
// all of them for a 10k-row file would drown the one fact the admin needs
// ("how much landed"). The full list lives in the response only up to this
// window — the summary counts stay exact either way.
const maxRenderedImportErrors = 20

// userImportPage renders the bulk-import form (super-admin): a file input,
// a format note, and the passwordless caveat the create-user form carries.
func (h *Handler) userImportPage(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Import users", "/users", user, r)
	renderPage(w, r, userImportPage(p, h.config.BasePath))
}

// userImport processes the uploaded CSV/JSON through the Service's public
// importer. Existing emails are SKIPPED by the importer (idempotent
// re-runs), per-row failures are summarized honestly, and an oversized or
// unparseable upload is refused before anything is dispatched.
func (h *Handler) userImport(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportUploadBytes)
	//nolint:gosec // G120: body capped by MaxBytesReader above
	if err := r.ParseMultipartForm(maxImportUploadBytes); err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusRequestEntityTooLarge,
			"Import file too large",
			fmt.Sprintf(
				"The uploaded file exceeds the %d MiB import limit. Split the file and import in batches.",
				maxImportUploadBytes>>(10*2),
			),
		)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "No file", "Choose a CSV or JSON file to import.")
		return
	}
	defer func() { _ = file.Close() }()

	format, ferr := importFormat(header)
	if ferr != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Unknown format", ferr.Error())
		return
	}

	// The upload is already size-bounded by the MaxBytesReader, so reading
	// it whole is safe and lets the CSV guard peek the header.
	data, err := io.ReadAll(file)
	if err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Unreadable file", "The uploaded file could not be read.")
		return
	}
	if format == usermgmt.UserDataFormatCSV {
		if line, _, found := strings.Cut(string(data), "\n"); found && !csvHeaderHasEmail(line) {
			h.writeErrorPage(w, r, http.StatusBadRequest, "Missing email column",
				"The CSV header must include an \"email\" column — the importer maps columns by name.")
			return
		}
	}

	var result *usermgmt.ImportResult
	var ierr error
	switch format {
	case usermgmt.UserDataFormatJSON:
		result, ierr = h.config.Service.ImportUsersFromJSON(r.Context(), bytes.NewReader(data))
	case usermgmt.UserDataFormatCSV:
		result, ierr = h.config.Service.ImportUsersFromCSV(r.Context(), bytes.NewReader(data))
	}
	if ierr != nil {
		// A malformed file (bad JSON, wrong CSV header) is a client error:
		// nothing was imported, the admin fixes the file and retries.
		h.writeErrorPage(w, r, http.StatusBadRequest, "Import failed", actionErrorMessage(ierr))
		return
	}

	back := h.config.BasePath + "/users"
	if htmxRequest(r) {
		renderPartial(w, r, importResultPanel(*result, back))
		return
	}
	triggerToast(w, "ok", fmt.Sprintf("Imported %d, skipped %d", result.Imported, result.Skipped))
	redirect(w, r, back)
}

// userExportCSV / userExportJSON stream the directory in the declared
// format. Literal routes (not a wildcard segment — ServeMux forbids partial
// ones) so the URL is the format. The Service writes straight to the
// response writer — no whole-directory buffering — and Content-Disposition
// names the file a browser saves.
func (h *Handler) userExportCSV(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.export(w, r, usermgmt.UserDataFormatCSV)
}

func (h *Handler) userExportJSON(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.export(w, r, usermgmt.UserDataFormatJSON)
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request, format usermgmt.UserDataFormat) {
	name := "users-export." + string(format)
	w.Header().Set("Content-Type", exportContentType(format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	var err error
	switch format {
	case usermgmt.UserDataFormatJSON:
		err = h.config.Service.ExportUsersToJSON(r.Context(), w)
	case usermgmt.UserDataFormatCSV:
		err = h.config.Service.ExportUsersToCSV(r.Context(), w)
	}
	if err != nil {
		// Headers are already committed by the streaming write; the partial
		// output is honest about failing rather than silently truncated.
		triggerToast(w, "err", "Export failed: "+actionErrorMessage(err))
	}
}

// csvHeaderHasEmail reports whether the first CSV line declares an email
// column. The upstream importer maps columns BY NAME and silently yields an
// empty result on a foreign header — the panel refuses that up front so the
// admin hears why nothing landed.
func csvHeaderHasEmail(header string) bool {
	for _, field := range strings.Split(strings.ToLower(header), ",") {
		if strings.TrimSpace(field) == "email" {
			return true
		}
	}
	return false
}

// importFormat decides JSON vs CSV from the uploaded file's name. The field
// is the honest signal (content sniffing is guesswork); a file without a
// known extension is refused with the two accepted spellings.
func importFormat(header *multipart.FileHeader) (usermgmt.UserDataFormat, error) {
	name := strings.ToLower(header.Filename)
	switch {
	case strings.HasSuffix(name, ".json"):
		return usermgmt.UserDataFormatJSON, nil
	case strings.HasSuffix(name, ".csv"):
		return usermgmt.UserDataFormatCSV, nil
	default:
		return "", errorfamily.NewRejection(
			"adminui.import.format",
			"the file name must end in .csv or .json",
		)
	}
}

// exportContentType maps the wire format to its media type.
func exportContentType(format usermgmt.UserDataFormat) string {
	if format == usermgmt.UserDataFormatCSV {
		return "text/csv; charset=utf-8"
	}
	return "application/json"
}

// htmxRequest reports whether the POST came from htmx (partial swap wanted).
func htmxRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// visibleImportErrors bounds the rendered error list; remainingImportErrors
// reports the honest remainder (templ renders each with a single-value call).
func visibleImportErrors(errs []string) []string {
	if len(errs) <= maxRenderedImportErrors {
		return errs
	}
	return errs[:maxRenderedImportErrors]
}

func remainingImportErrors(errs []string) int {
	if len(errs) <= maxRenderedImportErrors {
		return 0
	}
	return len(errs) - maxRenderedImportErrors
}
