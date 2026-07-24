# Comprehensive TODO Execution Plan — 2026-07-20

**Goal:** Complete ALL 5 open TODO_LIST items, verified, one at a time.

## Findings from research (truth)

| TODO                                | Current state                                                                                                     | Real work remaining                                                                                                                                                               |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Property-based tests for event fold | 20 single-event property tests exist (`property_test.go`, `property_extras_test.go`, `property_identity_test.go`) | **Gap:** no random-SEQUENCE properties (the rapid/Hypothesis killer feature: shrinking, associativity, idempotent folds, monotonic versions)                                      |
| SSE broadcaster load benchmarks     | **None exist**                                                                                                    | Add fan-out throughput benchmark across subscriber counts (1/10/100/1000) + slow-consumer drop                                                                                    |
| OpenAPI spec generation             | **None exists**                                                                                                   | New opt-in builder (`openapi/` submodule-free package in root): declarative Spec + PathItem + HandlerOption metadata hook                                                         |
| Admin UI OAuth2 link/unlink         | **None exists**                                                                                                   | External-accounts card on user detail + `Unlink` endpoint (public `UnlinkExternalAccount` exists). Link documented as user-side (OAuth handshake cannot be impersonated by admin) |
| dedup.Ring vs map benchmark         | **Exists but missing 100K** (only 100/1K/10K)                                                                     | Add 100K tier + `b.ReportAllocs`                                                                                                                                                  |

## Execution order (Pareto: impact × 1/effort)

| #   | Task                            | Impact | Effort | Why this order                                                                                           |
| --- | ------------------------------- | ------ | ------ | -------------------------------------------------------------------------------------------------------- |
| 1   | dedup 100K                      | Med    | XS     | Trivial completion of existing benchmark; fast morale win                                                |
| 2   | SSE broadcaster load benchmarks | High   | S      | Entirely missing; the broadcaster is core realtime infra, load data is high customer value               |
| 3   | Property sequence tests         | High   | M      | Fills the REAL gap (single-event tests already pass); catches fold associativity/idempotency regressions |
| 4   | Admin UI OAuth2 unlink views    | High   | M      | User-facing feature; closes a visible admin gap                                                          |
| 5   | OpenAPI spec generation         | High   | L      | Architectural; new opt-in API surface, biggest scope, last                                               |

## Subtask breakdown (each ≤ ~12 min)

### 1. dedup.Ring 100K (XS)

1a. Add 100000 to sizes slice + ReportAllocs on both ring & map.
1b. Run benchmark, confirm it completes, record numbers in CHANGELOG.

### 2. SSE broadcaster load benchmarks (S)

2a. Create `sse_broadcaster_bench_test.go`: BenchmarkBroadcasterFanOut over {1,10,100,1000} subscribers; Broadcast to all; ReportAllocs + SetBytes.
2b. Add BenchmarkBroadcasterSlowConsumerDrop: subscriber that never drains, verify non-blocking.
2c. Add BenchmarkSubscribeUnsubscribe (churn). Run + verify.

### 3. Property-based sequence tests (M)

3a. Create `property_sequences_test.go`: rapid generator for a stream of user events; invariant: fold-left over concat == fold of prefix then suffix (associativity).
3b. Invariant: tombstone idempotency (two UserDeleted events == one).
3c. Invariant: unknown event halts (fold returns error, state unchanged semantics).
3d. Same for tenant/membership/bot sequences where sensible. Run + verify.

### 4. Admin UI OAuth2 unlink views (M)

4a. Extend `userDetailData` with `ConfiguredProviders []string` + `UnlinkBase string`.
4b. Add templ block: external-accounts card (list provider/email/date + Unlink button each).
4c. Add route `POST /users/{id}/external/{provider}/unlink` + handler calling `svc.UnlinkExternalAccount`.
4d. Add tests (handler + render). Regenerate `_templ.go`. Build + test.

### 5. OpenAPI spec generation (L)

5a. New package `openapi/` in root module: types (Spec, PathItem, Operation, Response, Schema) + fluent Builder.
5b. Marshal to OpenAPI 3.1 JSON (encoding/json/v2). Validate against a golden test.
5c. Add `HandlerOption` metadata hook (`WithOpenAPI(op)`) on the App so consumers can annotate endpoints, plus `App.OpenAPI()` collector — opt-in only.
5d. Tests + example snippet in doc.go. Build + test.

## Verification gates (after EACH task)

- `GOEXPERIMENT=jsonv2 go build ./...` (root + workspace)
- `GOEXPERIMENT=jsonv2 go test <affected module> ./... -count=1 -race`
- `nix run .#lint` scope where touched
- Append CHANGELOG entry; mark TODO item resolved (remove from TODO_LIST per convention).

## Final deliverable

Table view of all 5 TODOs: status, files touched, verification result.

---

> **Resolution (2026-07-20, v4.5.0):** All 5 items executed. dedup.Ring 100K benchmark, SSE
> broadcaster load benchmarks, and property-based sequence tests shipped. AdminUI OAuth2 unlink
> views and OpenAPI spec generation also shipped. The OpenAPI race condition found during execution
> was fixed in `2026-07-20_23-04`. See CHANGELOG [v4.5.0].
