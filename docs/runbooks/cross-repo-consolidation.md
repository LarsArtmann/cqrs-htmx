# Cross-Repo Consolidation Runbook

> Step-by-step procedure for consolidating a Go module from cqrs-htmx into
> go-cqrs-lite (or vice versa). Based on the catalog/ merge (ADR 0020).

## When to Use This

When a module exists in cqrs-htmx but would be better placed upstream in
go-cqrs-lite (or the reverse). This happens when the code has no cqrs-htmx-
specific dependencies and serves a broader audience.

## Procedure

### Phase 1: Port Upstream (1% that delivers 51%)

1. **Audit unique code** — identify what exists only in the source module
   (not already upstream). Delete anything that's a strict subset of upstream.
2. **Create upstream packages** — port unique code into the target repo.
   Use upstream utilities (e.g., `internal/caseutil.ToKebab`) instead of
   hand-rolled equivalents.
3. **Port tests** — adapt imports, fix lint (tagliatelle JSON casing, etc.).
4. **Lint new packages** — `golangci-lint run ./newpkg/`. Add lint exclusions
   if needed (e.g., `wrapcheck` for thin wrappers around upstream calls).
5. **Update upstream README** — add new package rows to the package table.
6. **Update upstream CHANGELOG** — add `[Unreleased]` entry.
7. **Commit + tag upstream** — `git tag -a <module>/v<X.Y.Z> -m "..."`.
   **Use annotated tags** (`-a -m`), not lightweight tags.
8. **Push upstream** — `git push && git push <module>/v<X.Y.Z>`.

### Phase 2: Migrate Consumers (4% that delivers 64%)

9. **Update go.mod** — remove old module dep, add new upstream dep at the
   tagged version. Use `go get ...@v<X.Y.Z>` (not `go mod tidy` — it can
   fail on transitive test deps).
10. **Update imports** — swap all import paths.
11. **Update handler/API calls** — map old function names to new ones.
12. **Remove `replace` directives** — if the old module had local replace
    directives, remove them from go.mod.
13. **Update go.work** — remove the old module from the `use` list.
14. **Update build infra** — remove old module references from `flake.nix`
    (test/lint/build/coverage/coverage-gate apps).
15. **Delete old module** — `rm -rf oldmodule/`.
16. **Verify build + tests** — `go build ./... && go test ./... -race` across
    all workspace modules.

### Phase 3: Documentation Honesty (20% that delivers 80%)

17. **Run the reference sweep** (CRITICAL — do not skip):
    ```bash
    grep -rn "<old-module>" \
      --include="*.go" --include="*.md" --include="*.nix" \
      --include="go.mod" --include="go.work" . \
      | grep -v "docs/status/" | grep -v "docs/planning/" | grep -v "docs/reviews/"
    ```
    Fix EVERY hit in living docs (AGENTS.md, README.md, FEATURES.md,
    CHANGELOG.md, VERSIONING.md, CONTRIBUTING.md).
18. **Update CHANGELOG** — add `[Unreleased]` breaking-change entry with
    import mapping table.
19. **Flip ADR status** — mark the original ADR as SUPERSEDED with a pointer
    to the new reversal ADR.
20. **Write reversal ADR** — document the decision and rationale.
21. **Check ADR numbering** — `ls docs/adr/ | sort` before creating a new ADR
    to avoid number collisions.
22. **Update architecture tree** — in AGENTS.md, remove deleted module.
23. **Update FEATURES.md** — replace the feature section with a pointer to
    upstream. Never leave a FULLY_FUNCTIONAL entry for deleted code.
24. **Verify** — `go build ./... && go test ./... -race && golangci-lint run`.
25. **Commit + push** — one commit for the whole consumer migration.

## Common Pitfalls

| Pitfall | Prevention |
|---------|------------|
| Stale docs (code green, docs lying) | Step 17 — always run the reference sweep |
| ADR number collision | Step 21 — check `ls docs/adr/ \| sort` |
| `go mod tidy` fails on transitive test deps | Use `go get @version` instead |
| Unpushed upstream tag | Verify `git ls-remote --tags origin \| grep <tag>` |
| Breaking change without CHANGELOG | Step 18 — mandatory, not optional |

## See Also

- [ADR 0020: Merge catalog into go-cqrs-lite](../adr/0020-merge-catalog-into-go-cqrs-lite.md)
- [AGENTS.md Migration Gotchas (#22-26)](../../AGENTS.md)
