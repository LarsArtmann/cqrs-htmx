package usermgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// verificationVerifyRequest is the body for POST /auth/email/verify.
type verificationVerifyRequest struct {
	Token string `json:"token"`
}

// totpCodeRequest is the body for TOTP verify endpoints.
type totpCodeRequest struct {
	Code string `json:"code"`
}

// RegisterVerificationTOTPRoutes registers email verification, TOTP, and
// user import/export endpoints on the given ServeMux. These endpoints assume
// the mux is wrapped with NewSessionMiddleware so that UserFromContext works.
//
//	POST /auth/email/verify/send    — send verification email to current user
//	POST /auth/email/verify         — verify email with token
//	POST /auth/totp/setup           — begin TOTP setup (returns secret + QR URI)
//	POST /auth/totp/setup/verify    — confirm TOTP setup with authenticator code
//	POST /auth/totp/verify          — verify a TOTP code (second factor)
//	POST /auth/totp/disable         — disable TOTP for current user
//	GET  /auth/export?format=json|csv — export all users
//	POST /auth/import?format=json|csv — import users from JSON or CSV
func (h *AuthHandler) RegisterVerificationTOTPRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/email/verify/send", h.handleSendVerificationEmail)
	mux.HandleFunc("POST /auth/email/verify", h.handleVerifyEmail)
	mux.HandleFunc("POST /auth/totp/setup", h.handleTOTPSetup)
	mux.HandleFunc("POST /auth/totp/setup/verify", h.handleTOTPSetupVerify)
	mux.HandleFunc("POST /auth/totp/verify", h.handleTOTPVerify)
	mux.HandleFunc("POST /auth/totp/disable", h.handleTOTPDisable)
	mux.HandleFunc("GET /auth/export", h.handleExportUsers)
	mux.HandleFunc("POST /auth/import", h.handleImportUsers)
}

func (h *AuthHandler) currentUser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return nil, false
	}
	return user, true
}

func (h *AuthHandler) checkRateLimit(
	w http.ResponseWriter, r *http.Request, rl *registrationRateLimiter, msg string,
) bool {
	if rl != nil && !rl.allow(r.RemoteAddr) {
		writeError(w, http.StatusTooManyRequests, msg)
		return false
	}
	return true
}

// withTimeout returns a context with the handler's configured timeout.
// If timeout is zero, returns the request context unchanged.
func (h *AuthHandler) withTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	if h.timeout > 0 {
		return context.WithTimeout(r.Context(), h.timeout)
	}
	return r.Context(), func() {}
}

func (h *AuthHandler) handleSendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.verificationLimiter, "too many verification requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	token, err := h.service.SendVerificationEmail(ctx, user.ID)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.verificationLimiter, "too many verification requests") {
		return
	}
	var req verificationVerifyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("%w: invalid request body: %w", ErrValidation, err).Error())
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	if err := h.service.VerifyEmail(ctx, req.Token); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusVerified})
}

func (h *AuthHandler) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.totpLimiter, "too many TOTP requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	resp, err := h.service.EnableTOTP(ctx, user.ID)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleTOTPSetupVerify(w http.ResponseWriter, r *http.Request) {
	h.handleTOTPCode(w, r, func(ctx context.Context, userID UserID, code string) error {
		return h.service.VerifyTOTPSetup(ctx, userID, code)
	}, statusTOTPSetupVerified)
}

func (h *AuthHandler) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	h.handleTOTPCode(w, r, func(ctx context.Context, userID UserID, code string) error {
		return h.service.VerifyTOTP(ctx, userID, code)
	}, statusTOTPVerified)
}

func (h *AuthHandler) handleTOTPCode(
	w http.ResponseWriter,
	r *http.Request,
	verify func(context.Context, UserID, string) error,
	status string,
) {
	if !h.checkRateLimit(w, r, h.totpLimiter, "too many TOTP requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req totpCodeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("%w: invalid request body: %w", ErrValidation, err).Error())
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	if err := verify(ctx, user.ID, req.Code); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: status})
}

func (h *AuthHandler) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.totpLimiter, "too many TOTP requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req totpCodeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("%w: invalid request body: %w", ErrValidation, err).Error())
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	if err := h.service.DisableTOTP(ctx, user.ID, req.Code); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusTOTPDisabled})
}

func (h *AuthHandler) handleExportUsers(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.importLimiter, "too many export requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.importExportAuthorizer(user); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	format := parseExportFormat(r)
	switch format {
	case ExportFormatCSV:
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=users.csv")
		if err := h.service.ExportUsersToCSV(ctx, w); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	case ExportFormatJSON:
		w.Header().Set("Content-Type", contentTypeJSON)
		w.Header().Set("Content-Disposition", "attachment; filename=users.json")
		if err := h.service.ExportUsersToJSON(ctx, w); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func (h *AuthHandler) handleImportUsers(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.importLimiter, "too many import requests") {
		return
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.importExportAuthorizer(user); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	ctx, cancel := h.withTimeout(r)
	defer cancel()

	format := parseImportFormat(r)
	var result *ImportResult
	var err error
	switch format {
	case ExportFormatCSV:
		result, err = h.service.ImportUsersFromCSV(ctx, io.LimitReader(r.Body, maxAuthBodySize))
	case ExportFormatJSON:
		result, err = h.service.ImportUsersFromJSON(ctx, io.LimitReader(r.Body, maxAuthBodySize))
	}
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseExportFormat(r *http.Request) ExportFormat {
	v := strings.ToLower(r.URL.Query().Get("format"))
	if v == "csv" {
		return ExportFormatCSV
	}
	return ExportFormatJSON
}

func parseImportFormat(r *http.Request) ExportFormat {
	v := strings.ToLower(r.URL.Query().Get("format"))
	if v == "csv" {
		return ExportFormatCSV
	}
	if v == "json" {
		return ExportFormatJSON
	}
	ct := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if ct == "text/csv" {
		return ExportFormatCSV
	}
	return ExportFormatJSON
}
