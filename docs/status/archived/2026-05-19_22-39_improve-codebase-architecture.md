# Architecture Deepening Opportunities — cqrs-htmx

**Date:** 2026-05-19_22-39

## Method

Walked all 15 production modules. Applied the deletion test to each: "If I delete this module, does complexity concentrate (deep) or scatter (shallow)?"

## Candidates

### 1. HTMX Request Module — Shallow accessor seam

**Files:** `htmx.go:96-170`
**Problem:** 8 exported accessor functions are mechanically identical — check context then fall back to header. Each is a thin wrapper (~5 lines) with zero depth. The interface is as complex as the implementation. This is the definition of a shallow module.
**Solution:** Introduce a generic `htmxField[T]` helper. The 8 functions become one-liners delegating to it. Depth increases: callers don't change, but maintainers edit one function instead of 8.
**Benefits:** Locality — any future change to context-then-header fallback logic is in one place. One adapter at the seam.

### 2. Decode Module — Repeated decoder pattern

**Files:** `options.go:66-107`
**Problem:** `DecodeJSON`, `DecodeJSONQuery`, `DecodeForm`, `DecodeFormQuery` are structurally identical. Each creates a decoder closure that calls `decodeRequest` with a body decoder. The only axis of variation is (JSON|Form) × (Command|Query). Four functions where one generic could suffice.
**Solution:** A single `decode[T, R any](bodyDecoder, mapper, setter)` function parameterized by the body decoder and the handlerConfig setter. The four exported functions become thin wrappers.
**Benefits:** Leverage — the body decoding → mapping → error handling pipeline is written once. Locality — a bug in decode-then-map logic is fixed once.

### 3. Validation Module — Duplicate wrapping

**Files:** `options.go:201-250`
**Problem:** `ValidateCommand` and `ValidateQuery` have identical bodies. They check nil decoder, save original, wrap with validation, and wrap errors with `ErrValidationFailed`. The only difference is the decoder field (`commandDecoder` vs `queryDecoder`) and the validator signature (`func(command.Command) error` vs `func(query.Query) error`).
**Solution:** Since `command.Command` and `query.Query` are both interfaces, a generic `validate[T any](decoder *func(r *http.Request) (T, error), validator func(T) error)` could collapse both.
**Benefits:** Depth — one validation wrapper to test and maintain. Leverage — future validation features (e.g., multi-error accumulation) are added once.

### 4. Notification Module — Triplicated notification surface

**Files:** `notify.go:17-78`, `response.go:121-146`
**Problem:** The notification pattern `{level, message}` → `TriggerWithDetail` is implemented in three places:

1. Package-level: `NotifySuccess/Error/Warning/Info` (4 functions)
2. `NotifyEventBuilder`: `.Success/.Error/.Warning/.Info` (4 methods)
3. `Response`: `.NotifySuccess/.NotifyError/.NotifyWarning/.NotifyInfo` (4 methods)

That's 12 functions doing the same thing: map a level+message to a TriggerWithDetail call. Two separate but identical implementations: `notifyOption` (package-level) and `triggerNotification` (Response method).
**Solution:** Keep the three surfaces (they serve different use cases) but have ALL of them delegate to a single internal `notify(level, message, event)` function. `notifyOption` and `triggerNotification` become calls to the same function.
**Benefits:** Locality — notification format changes in one place. Currently if you change the `{level, message}` key format, you must change two implementations.

### 5. ID Type Module — Mechanical Parse/New/Must/Context patterns

**Files:** `context.go:1-152`
**Problem:** Three ID types (UserID, CorrelationID, RequestID) each have 6 wrapper functions: `New`, `Parse`, `MustParse`, `With`, `FromContext`, plus the context key type. That's 18 mechanically-generated functions plus 3 context key types, all following identical patterns.
**Solution:** Go generics cannot easily abstract over context keys (they're distinct types), but a `parseID[T any](s string, parseFunc func(string) (T, error))` helper would collapse the Parse wrappers. The context functions could use a generic `withValue[T any](ctx, key, val)` + `fromContext[T any](ctx, key)`.
**Benefits:** Locality — if the error wrapping format changes, one edit. Leverage — adding a 4th ID type is near-zero work.

### 6. Logging Module — Repeated context extraction

**Files:** `logging.go:25-167`
**Problem:** `DefaultLogFormatter`, `JSONLogFormatter`, and `RequestLoggingSlog` all independently extract CorrelationID and UserID from context with identical logic. Adding a new context value (e.g., RequestID) requires editing three places.
**Solution:** A shared `contextFields(r) map[string]string` helper that extracts all known context IDs. Each formatter calls it and formats the output.
**Benefits:** Locality — adding a new context field is one edit. The formatters become purely about output format, not data extraction.

### 7. Error Handler Module — Shared redirect core

**Files:** `errors.go:127-183`
**Problem:** `DefaultErrorHandlerWithRedirect` and `JSONErrorHandlerWithRedirect` share the same skeleton: default loginRedirect → writeHTMXAuthRedirect → MapError → write response. Only the response format (plain text vs JSON) differs.
**Solution:** Extract `handleErrorWithRedirect(w, r, err, loginRedirect, writeBody func(status int, err error))`. Both handlers delegate to it with different `writeBody` closures.
**Benefits:** Depth — the redirect/Auth-error logic is in one place. Locality — changes to HTMX auth redirect behavior are made once.

## Summary Table

| # | Module               | Depth Problem           | Impact | Effort |
| - | -------------------- | ----------------------- | ------ | ------ |
| 1 | HTMX accessors       | 8 shallow wrappers      | Medium | Low    |
| 2 | Decoder pattern      | 4 identical structures  | High   | Low    |
| 3 | Validation wrapping  | 2 identical bodies      | Medium | Low    |
| 4 | Notification surface | 12 duplicated functions | Medium | Low    |
| 5 | ID type pattern      | 18 mechanical wrappers  | Medium | Medium |
| 6 | Logging extraction   | 3 copy-paste blocks     | Low    | Low    |
| 7 | Error handler core   | 2 shared skeletons      | Low    | Low    |

## Recommendation

Candidates 1-4 are high-leverage, low-effort. They should be done first. Candidate 5 requires Go generics expertise and is medium effort. Candidates 6-7 are polish.
