# Local eventtest module

This directory contains a local copy of `github.com/larsartmann/go-cqrs-lite/event/v3/eventtest`
to work around a tag naming bug in go-cqrs-lite (the tag `event/v3/eventtest/v3.5.0` is
misnamed — Go expects `event/eventtest/v3.5.0`).

The `go.work` file has a `replace` directive pointing here. This only affects workspace
development — consumers of cqrs-htmx are unaffected.

**When upgrading go-cqrs-lite**: re-extract this directory from the new tag:

```bash
CACHE=$(go env GOMODCACHE)/cache/vcs/<go-cqrs-lite-hash>
cd "$CACHE" && git archive <new-tag> event/eventtest/ | tar -x -C /tmp/et
cp /tmp/et/event/eventtest/* <repo>/.vendor-local/eventtest/
# Then update go.mod inside to use the new versions (remove replace directives)
```
