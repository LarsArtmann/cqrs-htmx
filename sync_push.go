package cqrshtmx

import (
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// MaxSyncPushBatch is the maximum number of command envelopes accepted in one
// batch push. Larger batches are rejected whole (400) before any dispatch —
// the guard exists to protect the server, not to rate-limit the client.
const MaxSyncPushBatch = 100

// Sync push outcome statuses (see [SyncPushResult]).
const (
	SyncPushStatusConfirmed = "confirmed"
	SyncPushStatusRejected  = "rejected"
)

// SyncPushEnvelope is one queued command as sent by the offline sync client.
// Body is the exact request body the single-command endpoint's decoder
// expects (JSON or form-encoded — match ContentType accordingly); commandId is
// the client-minted ULID used for ACK correlation and idempotent retries.
type SyncPushEnvelope struct {
	CommandID   string `json:"commandId"`
	Type        string `json:"type"`
	Body        string `json:"body"`
	ContentType string `json:"contentType,omitempty"`
}

// SyncPushRequest is the batch push request body.
type SyncPushRequest struct {
	Commands []SyncPushEnvelope `json:"commands"`
}

// SyncPushError is the error detail attached to a rejected outcome. Family is
// the client's retry contract: "rejection"/"conflict" are permanent (surface
// to the user, drop from the queue), "transient" is retryable (re-queue),
// "corruption"/"infrastructure" are permanent-failures (drop + log).
type SyncPushError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Family  string `json:"family"`
}

// SyncPushResult is the outcome of one envelope. Commands answer with
// outcomes, never domain state (Axon CommandGateway semantics) — state arrives
// via pull or SSE.
type SyncPushResult struct {
	CommandID string         `json:"commandId"`
	Status    string         `json:"status"`
	Error     *SyncPushError `json:"error,omitempty"`
}

// SyncPushResponse is the batch push response body. The endpoint answers 200
// whenever the batch itself was processable; per-command failures live in the
// results.
type SyncPushResponse struct {
	Results []SyncPushResult `json:"results"`
}

// rememberCommandConfig retains a command registration for SyncPushHandler.
// Later registrations for the same type win (mirrors route registration
// semantics: the last wiring is the live one).
func (a *App) rememberCommandConfig(cmdType command.Type, config *handlerConfig) {
	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	if a.commandConfigs == nil {
		a.commandConfigs = make(map[command.Type]*handlerConfig)
	}

	a.commandConfigs[cmdType] = config
}

// commandConfig looks up a retained command registration.
func (a *App) commandConfig(cmdType command.Type) (*handlerConfig, bool) {
	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	config, ok := a.commandConfigs[cmdType]

	return config, ok
}

// SyncPushHandler returns an http.HandlerFunc implementing the write half of
// the frontend sync protocol (ADR-0056): a batched, per-command-outcome push
// of queued commands.
//
//	POST {commands: [{commandId, type, body, contentType}]}
//	  → {results: [{commandId, status: "confirmed"|"rejected", error?}]}
//
// Each envelope replays the SAME pipeline its single-command endpoint uses —
// authorization, decoding, request guards, context enrichment, dispatch —
// against a synthesized request carrying the envelope body plus the outer
// request's headers and context. No new per-command registration is needed:
// the types are the ones already registered via [App.Command] /
// CommandTyped.
//
// The envelope's commandId is stamped as X-Command-Id on the synthesized
// request, so AfterDispatch hooks like Broadcaster.BroadcastOnAck fire per
// command and consumer idempotency middleware can dedup retries.
//
// Batch failures are isolated: unknown types, decoding failures, and rejections
// answer per-command without aborting the batch. Only malformed requests
// (wrong method, undecodable body, empty/oversized batch) fail whole with 4xx.
//
// CSRF is validated once on the outer POST by the surrounding middleware;
// inner envelopes are not independently reachable and do not re-validate.
func (a *App) SyncPushHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeSyncPullError(w, http.StatusMethodNotAllowed, "cqrshtmx.sync.push.method_not_allowed",
				"sync push is a POST endpoint")

			return
		}

		if a.commands == nil {
			writeSyncPullError(w, http.StatusServiceUnavailable, "cqrshtmx.sync.push.no_commands",
				"no command dispatcher configured")

			return
		}

		var body io.Reader = r.Body
		if a.maxBodySize > 0 {
			body = http.MaxBytesReader(w, r.Body, a.maxBodySize)
		}

		var req SyncPushRequest
		if err := json.UnmarshalRead(body, &req); err != nil {
			writeSyncPullError(w, http.StatusBadRequest, "cqrshtmx.sync.push.decode",
				"request body must be a JSON batch of commands")

			return
		}

		if len(req.Commands) == 0 {
			writeSyncPullError(w, http.StatusBadRequest, "cqrshtmx.sync.push.empty",
				"commands must not be empty")

			return
		}

		if len(req.Commands) > MaxSyncPushBatch {
			writeSyncPullError(w, http.StatusBadRequest, "cqrshtmx.sync.push.too_large",
				"batch exceeds MaxSyncPushBatch envelopes")

			return
		}

		results := make([]SyncPushResult, 0, len(req.Commands))
		for _, envelope := range req.Commands {
			results = append(results, a.dispatchSyncPushEnvelope(r, envelope))
		}

		_ = WriteJSON(
			w,
			http.StatusOK,
			SyncPushResponse{Results: results},
		) //nolint:errcheck // WriteJSON fails only on encode bugs; the empty-body class is guarded by tests
	}
}

// dispatchSyncPushEnvelope replays the endpoint pipeline for one envelope and
// returns its outcome. It never panics upward on domain errors: every failure
// path becomes a rejected result with a classified family.
func (a *App) dispatchSyncPushEnvelope(r *http.Request, envelope SyncPushEnvelope) SyncPushResult {
	if envelope.CommandID == "" {
		return rejectedSyncPushResult("", "cqrshtmx.sync.push.command_id_missing",
			"commandId is required", event.Rejection)
	}

	cmdType := command.Type(envelope.Type)

	config, ok := a.commandConfig(cmdType)
	if !ok || config.commandDecoder == nil {
		return rejectedSyncPushResult(envelope.CommandID, "cqrshtmx.sync.push.unknown_type",
			"unknown command type: "+envelope.Type, event.Rejection)
	}

	inner := synthesizeSyncPushRequest(r, envelope)

	if err := a.executeAuthorization(inner, config); err != nil {
		return rejectedSyncPushResult(envelope.CommandID, ErrorCode(err), err.Error(),
			errorfamily.Classify(err))
	}

	cmd, err := config.commandDecoder(inner)
	if err != nil || cmd == nil {
		if err == nil {
			err = errDecoderReturnedNil
		}

		return rejectedSyncPushResult(envelope.CommandID, "cqrshtmx.sync.push.decode_failed",
			"decode "+envelope.Type+" failed: "+err.Error(), event.Rejection)
	}

	if config.requestGuard != nil {
		if err := config.requestGuard(inner, cmd); err != nil {
			return rejectedSyncPushResult(envelope.CommandID, ErrorCode(err), err.Error(),
				errorfamily.Classify(err))
		}
	}

	ctx, cancel := a.timeoutCtx(inner.Context(), config)
	defer cancel()

	enrichCommandFromContext(ctx, cmd)

	dispatchErr := a.commands.Dispatch(ctx, cmd)

	// The afterDispatch hook runs per envelope with the synthesized request,
	// so ACK-style hooks (BroadcastOnAck reads X-Command-Id) fire per command.
	a.afterDispatchHook(ctx, inner, dispatchErr)

	if dispatchErr != nil {
		return rejectedSyncPushResult(envelope.CommandID, ErrorCode(dispatchErr),
			SafeDetail(dispatchErr, MapError(dispatchErr), false),
			errorfamily.Classify(dispatchErr))
	}

	slog.DebugContext(ctx, "cqrs-htmx: sync push confirmed",
		slog.String("commandId", envelope.CommandID),
		slog.String("commandType", envelope.Type),
	)

	return SyncPushResult{CommandID: envelope.CommandID, Status: SyncPushStatusConfirmed}
}

// synthesizeSyncPushRequest clones the outer request with the envelope's body.
// Headers (session cookies, X-Client-Id, CSRF-free content headers) and
// context (user ID, actor, correlation) carry over unchanged, so decoders and
// authz behave exactly as on the single-command endpoint. X-Command-Id is
// (re)stamped from the envelope for ACK correlation and idempotent retries.
func synthesizeSyncPushRequest(r *http.Request, envelope SyncPushEnvelope) *http.Request {
	inner := r.Clone(r.Context())

	contentType := envelope.ContentType
	if contentType == "" {
		contentType = ContentTypeJSON
	}

	inner.Header = inner.Header.Clone()
	inner.Header.Set("Content-Type", contentType)
	inner.Header.Set(CommandIDHeader, envelope.CommandID)

	inner.Body = io.NopCloser(strings.NewReader(envelope.Body))
	inner.ContentLength = int64(len(envelope.Body))

	return inner
}

// rejectedSyncPushResult builds a rejected outcome. The unknown family (a
// zero-value classification) defaults to Rejection so clients always see a
// decidable retry contract.
func rejectedSyncPushResult(commandID, code, message string, family errorfamily.Family) SyncPushResult {
	if family.String() == "unknown" {
		family = errorfamily.Rejection
	}

	return SyncPushResult{
		CommandID: commandID,
		Status:    SyncPushStatusRejected,
		Error: &SyncPushError{
			Message: message,
			Code:    code,
			Family:  family.String(),
		},
	}
}
