> [!NOTE]
> This filing was drafted by GLM-5.3 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

## TL;DR

`isUnsafeCSRFMethod` (csrf.go:787) safe-lists GET/HEAD/OPTIONS/TRACE, so a native HTTP QUERY request (RFC 10008 — safe, idempotent, request-body-carrying) is CSRF-challenged like a mutation. QUERY is spec-safe exactly like GET; the challenge is conservative but wrong-headed friction: token-less QUERY clients (server-to-server readers, new browsers) get rejected on a read. Ask: add `QUERY` to the safe list behind the next minor, with a release note for anyone who deliberately wanted the challenge.

## Problem

RFC 10008 (published 2026) defines QUERY with GET's safety and idempotence: servers MUST NOT derive state from it. CSRF attestation-consistency checking exists for state-changing requests — the method's own contract places QUERY outside that class. The current switch:

```go
func isUnsafeCSRFMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	}
	return true // <- "QUERY" lands here
}
```

Concrete consumer impact (cqrs-htmx, ADR-0064): mounting `QUERY /contacts/search` behind `CSRFMiddleware` today answers token-less native QUERY with the challenge rejection; consumers work around it by shipping only the POST+`X-HTTP-Method-Override` fallback, which keeps the WIRE method a POST forever — the exact inversion of what the fallback ladder wants long-term.

## Suggested fix

```go
case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, "QUERY":
	return false
```

(stdlib has no `http.MethodQuery` constant — the literal rides with a comment, or a shared constant if one is ever added upstream.) Release-note the behavior change: a consumer who WANTED QUERY challenged can re-add it via a custom safe-method list if the config exposes one; if it does not, that knob is the only ask beyond the literal.

Repro: `curl -X QUERY -d '{"filter":"x"}' http://localhost:8080/query-route` behind default `CSRFMiddleware` → challenge rejection; same request as GET-equivalent → passes.

💘 Generated with Crush
