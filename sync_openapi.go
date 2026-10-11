package cqrshtmx

import (
	"net/http"
	"strconv"

	"github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// SyncPullOpenAPIOp returns OpenAPI 3.1 metadata describing the
// [SyncPullHandler] wire contract, ready to attach wherever the consumer
// mounted the handler:
//
//	mux.Handle("GET /sync/pull", cqrshtmx.SyncPullHandler(journal, ...))
//	spec.PathOp("/sync/pull", cqrshtmx.SyncPullOpenAPIOp())
//
// The schemas are inline (self-contained): the operation drops into any spec
// without component registration. See [AddSyncOpenAPI] for the common
// one-liner covering both sync endpoints.
func SyncPullOpenAPIOp() openapi.Operation {
	return openapi.Get("syncPull").
		Tag("sync").
		Summary("Pull domain events after a cursor").
		Description(
			"Read half of the frontend sync protocol (ADR-0056): a cursor-based, " +
				"permission-filtered batch pull of events WITH payloads. Responses are " +
				"ETagged over (backendId, nextCursor, event count, hasMore); send " +
				"If-None-Match to make re-polls cheap. Events hidden by the server-side " +
				"WithSyncPullFilter still advance the cursor — they are never re-read.",
		).
		QueryParam("after", openapi.String(),
			"Event ID (ULID) cursor; pull events strictly after it. Empty bootstraps from the journal start.").
		QueryParam("limit",
			openapi.Integer().WithMin(1).WithMax(MaxSyncPullLimit),
			"Batch size; default "+strconv.Itoa(DefaultSyncPullLimit)+", hard ceiling "+strconv.Itoa(MaxSyncPullLimit)+".").
		HeaderParam("If-None-Match", openapi.String(),
			"ETag from a previous pull; a match answers 304 without a body.").
		Response(http.StatusOK,
			"The visible events after the cursor, in journal order.",
			openapi.JSON(syncPullResponseSchema()),
		).
		RespHeader("ETag", openapi.String(),
			"Weak-change tag over backendId, nextCursor, event count, and hasMore; reuse as If-None-Match.").
		RespHeader("Cache-Control", openapi.String(),
			"private, no-cache — always revalidate; the ETag makes that cheap.").
		Response(http.StatusNotModified, "Unchanged since the If-None-Match tag (no body).").
		Response(http.StatusBadRequest, "Invalid cursor or limit.", openapi.JSON(syncErrorSchema())).
		Op()
}

// SyncPushOpenAPIOp returns OpenAPI 3.1 metadata describing the
// [App.SyncPushHandler] wire contract, ready to attach wherever the consumer
// mounted the handler:
//
//	mux.Handle("POST /sync/push", app.SyncPushHandler())
//	spec.PathOp("/sync/push", cqrshtmx.SyncPushOpenAPIOp())
func SyncPushOpenAPIOp() openapi.Operation {
	return openapi.Post("syncPush").
		Tag("sync").
		Summary("Push a batch of queued commands").
		Description(
			"Write half of the frontend sync protocol (ADR-0056): each envelope replays " +
				"the SAME pipeline its single-command endpoint uses (authorization, " +
				"decoding, request guards, context enrichment, dispatch) against the outer " +
				"request's headers and context. Commands answer with per-command outcomes, " +
				"never domain state — state arrives via pull or SSE. Batch failures are " +
				"isolated; only malformed requests (wrong method, undecodable body, empty " +
				"or oversized batch) fail whole.",
		).
		JSONBody(openapi.Object(
			openapi.PropReq("commands",
				openapi.Array(syncPushEnvelopeSchema()).WithMinItems(1).WithMaxItems(MaxSyncPushBatch)),
		)).
		Response(http.StatusOK,
			"The batch was processable; per-command outcomes live in the results.",
			openapi.JSON(syncPushResponseSchema()),
		).
		Response(http.StatusBadRequest,
			"Malformed batch: undecodable body, empty batch, or more than "+
				strconv.Itoa(MaxSyncPushBatch)+" envelopes.",
			openapi.JSON(syncErrorSchema()),
		).
		Response(http.StatusServiceUnavailable,
			"No command dispatcher configured on the App.",
			openapi.JSON(syncErrorSchema()),
		).
		Op()
}

// AddSyncOpenAPI registers the frontend-sync pull and push operations on the
// spec at the given paths — wherever the consumer mounted [SyncPullHandler]
// and [App.SyncPushHandler]:
//
//	spec := cqrshtmx.AddSyncOpenAPI(spec, "/sync/pull", "/sync/push")
//
// Equivalent to calling [openapi.Spec.PathOp] with [SyncPullOpenAPIOp] and
// [SyncPushOpenAPIOp].
func AddSyncOpenAPI(spec *openapi.Spec, pullPath, pushPath string) *openapi.Spec {
	return spec.
		PathOp(pullPath, SyncPullOpenAPIOp()).
		PathOp(pushPath, SyncPushOpenAPIOp())
}

// syncPullResponseSchema mirrors [SyncPullResponse].
func syncPullResponseSchema() *openapi.Schema {
	return openapi.Object(
		openapi.Prop("backendId", openapi.String().WithDescription(
			"Stable identity of the backing journal; clients reset their local cache when it changes. Omitted when no backend ID is configured.")),
		openapi.PropReq("events", openapi.Array(syncEventSchema())),
		openapi.Prop("nextCursor", openapi.String().WithDescription(
			"Event ID to pass as ?after= on the next pull; the last READ event (visible or filtered-out), so invisible events are never re-read. Empty when no events were read — keep the previous cursor.")),
		openapi.PropReq("hasMore", openapi.Boolean()),
	)
}

// syncEventSchema mirrors [SyncEvent].
func syncEventSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("eventId", openapi.String()),
		openapi.PropReq("type", openapi.String()),
		openapi.PropReq("streamType", openapi.String()),
		openapi.PropReq("streamId", openapi.String()),
		openapi.PropReq("version", openapi.Integer()),
		openapi.Prop("schemaVersion", openapi.String()),
		openapi.PropReq("occurredAt", openapi.String().WithFormat("date-time")),
		openapi.PropReq("payloadEncoding", openapi.String().WithDescription(
			"Framing of the delivered payload bytes: \"json\" — payload holds verbatim JSON; "+
				"a codec name (e.g. \"cbor\") — payloadB64 holds that codec's bytes; "+
				"\"opaque\" — the event's stamp promised JSON but the bytes are not valid JSON, "+
				"so payloadB64 carries them without a decodable-framing promise. Exactly one "+
				"of payload/payloadB64 is ever set.")),
		openapi.Prop("payload", openapi.FreeForm().WithDescription("Present only when payloadEncoding is json.")),
		openapi.Prop("payloadB64", openapi.String().WithFormat("byte").
			WithDescription("Present only when payloadEncoding is not json.")),
	)
}

// syncPushEnvelopeSchema mirrors [SyncPushEnvelope].
func syncPushEnvelopeSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("commandId", openapi.String().WithDescription(
			"Client-minted ULID; used for ACK correlation and idempotent retries.")),
		openapi.PropReq("type", openapi.String().WithDescription(
			"Command type registered via App.Command / CommandTyped.")),
		openapi.PropReq("body", openapi.String().WithDescription(
			"Exact request body the single-command endpoint's decoder expects (JSON or form-encoded; match contentType).")),
		openapi.Prop("contentType", openapi.String().WithDescription(
			"Body's content type, e.g. application/json (default) or application/x-www-form-urlencoded.")),
	)
}

// syncPushResponseSchema mirrors [SyncPushResponse].
func syncPushResponseSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("results", openapi.Array(syncPushResultSchema())),
	)
}

// syncPushResultSchema mirrors [SyncPushResult].
func syncPushResultSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("commandId", openapi.String()),
		openapi.PropReq("status", openapi.String().
			WithEnum(SyncPushStatusConfirmed, SyncPushStatusRejected)),
		openapi.Prop("error", syncPushErrorSchema()),
	)
}

// syncPushErrorSchema mirrors [SyncPushError].
func syncPushErrorSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("message", openapi.String()),
		openapi.Prop("code", openapi.String()),
		openapi.PropReq("family", openapi.String().WithDescription(
			"The client's retry contract: rejection/conflict are permanent (surface, drop); "+
				"transient is retryable (re-queue); corruption/infrastructure are permanent failures (drop, log).").
			WithEnum("rejection", "conflict", "transient", "corruption", "infrastructure")),
	)
}

// syncErrorSchema mirrors the standalone error body written by the sync
// endpoints ({"error", "status", "code"} — the JSONErrorHandler shape).
func syncErrorSchema() *openapi.Schema {
	return openapi.Object(
		openapi.PropReq("error", openapi.String()),
		openapi.PropReq("status", openapi.Integer()),
		openapi.Prop("code", openapi.String()),
	)
}
