Shipped as `huma/v4` — separate module, own `go.mod`, pins published root `v4.16.0`, so the root package gains nothing (huma is in no root dependency graph).

`RegisterCommand`/`RegisterQuery` run the same stages a typed operation cannot reach today: authorize → build → validate → guard → dispatch. Failures come back as the same RFC 9457 body the route pipeline writes — 401 anonymous, 403 denied, 503 engine down, 400 validator with `cqrshtmx.validate.failed`, 409 conflict — and Huma keeps decode, status codes and the schema (`TestOpenAPI_IsStillGeneratedByHuma` asserts error responses still point at `ErrorModel`).

```
GOWORK=off go build ./...   rc 0
GOWORK=off go vet ./...     rc 0
go test ./... -race         ok, 46 cases
coverage                    95.6% (gate 90)
golangci-lint               0 issues
```

Option shapes, stage-as-callable escape hatches and the deliberate gaps are in `huma/README.md`; the wiring recipe sits next to the datastar section in `docs/guides/fullstack-wiring.md`.

Still open on my side: no tag yet, so `go get github.com/larsartmann/cqrs-htmx/huma/v4` fails until `huma/v4.0.0` is pushed.
