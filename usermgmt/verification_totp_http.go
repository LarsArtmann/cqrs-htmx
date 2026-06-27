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
	w http.ResponseWriter, r *http.Request, rl *perIPRateLimiter, msg string,
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

// decodeAuthJSON decodes a JSON request body into target, applying
// maxAuthBodySize and returning a 400 Bad Request with ErrValidation
// on failure. Returns true on success; the handler should return on false.
func (h *AuthHandler) decodeAuthJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("%s: invalid request body: %s", ErrValidation, err))
		return false
	}
	return true
}

func (h *AuthHandler) handleSendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	user, ctx, cancel, ok := h.authContext(w, r, h.verificationLimiter, "too many verification requests")
	if !ok {
		return
	}
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
	if !h.decodeAuthJSON(w, r, &req) {
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
	user, ctx, cancel, ok := h.authContext(w, r, h.totpLimiter, "too many TOTP requests")
	if !ok {
		return
	}
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
	user, ctx, cancel, ok := h.authContext(w, r, h.totpLimiter, "too many TOTP requests")
	if !ok {
		return
	}
	defer cancel()

	var req totpCodeRequest
	if !h.decodeAuthJSON(w, r, &req) {
		return
	}

	if err := verify(ctx, user.ID, req.Code); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: status})
}

func (h *AuthHandler) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	h.handleTOTPCode(w, r, func(ctx context.Context, userID UserID, code string) error {
		return h.service.DisableTOTP(ctx, userID, code)
	}, statusTOTPDisabled)
}

// importExportContext runs the standard preflight for import/export endpoints:
// rate-limit, current user, import/export authorizer, and a timeout-bound
// context. On any failure it writes the response and returns ok=false.
func (h *AuthHandler) importExportContext(
	w http.ResponseWriter,
	r *http.Request,
	limiter *perIPRateLimiter,
	limitMsg string,
) (context.Context, context.CancelFunc, bool) {
	if !h.checkRateLimit(w, r, limiter, limitMsg) {
		return nil, nil, false
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return nil, nil, false
	}
	if err := h.importExportAuthorizer(user); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return nil, nil, false
	}
	ctx, cancel := h.withTimeout(r)
	return ctx, cancel, true
}

// authContext runs the common preflight for authenticated endpoints:
// rate-limit, current user, and a timeout-bound context. Returns the
// authenticated user alongside the context so handlers don't have to
// re-derive it. On any failure it writes the response and returns ok=false.
func (h *AuthHandler) authContext(
	w http.ResponseWriter,
	r *http.Request,
	limiter *perIPRateLimiter,
	limitMsg string,
) (*User, context.Context, context.CancelFunc, bool) {
	if !h.checkRateLimit(w, r, limiter, limitMsg) {
		return nil, nil, nil, false
	}
	user, ok := h.currentUser(w, r)
	if !ok {
		return nil, nil, nil, false
	}
	ctx, cancel := h.withTimeout(r)
	return user, ctx, cancel, true
}

func (h *AuthHandler) handleExportUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, ok := h.importExportContext(w, r, h.importLimiter, "too many export requests")
	if !ok {
		return
	}
	defer cancel()

	format := parseUserDataFormat(r)
	switch format {
	case UserDataFormatCSV:
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=users.csv")
		if err := h.service.ExportUsersToCSV(ctx, w); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	case UserDataFormatJSON:
		w.Header().Set("Content-Type", contentTypeJSON)
		w.Header().Set("Content-Disposition", "attachment; filename=users.json")
		if err := h.service.ExportUsersToJSON(ctx, w); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func (h *AuthHandler) handleImportUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, ok := h.importExportContext(w, r, h.importLimiter, "too many import requests")
	if !ok {
		return
	}
	defer cancel()

	format := parseImportFormat(r)
	var result *ImportResult
	var err error
	switch format {
	case UserDataFormatCSV:
		result, err = h.service.ImportUsersFromCSV(ctx, io.LimitReader(r.Body, maxAuthBodySize))
	case UserDataFormatJSON:
		result, err = h.service.ImportUsersFromJSON(ctx, io.LimitReader(r.Body, maxAuthBodySize))
	}
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseUserDataFormat(r *http.Request) UserDataFormat {
	v := strings.ToLower(r.URL.Query().Get("format"))
	if v == string(UserDataFormatCSV) {
		return UserDataFormatCSV
	}
	return UserDataFormatJSON
}

func parseImportFormat(r *http.Request) UserDataFormat {
	v := strings.ToLower(r.URL.Query().Get("format"))
	if v == string(UserDataFormatCSV) {
		return UserDataFormatCSV
	}
	if v == string(UserDataFormatJSON) {
		return UserDataFormatJSON
	}
	ct := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if ct == "text/csv" {
		return UserDataFormatCSV
	}
	return UserDataFormatJSON
}
