# Runbook: GitHub queue sweep (every open PR, issue, and alert)

One pass over the whole public queue: read it, verify it, land or close it,
report with receipts. The shape below is the one that survives contact with
this repo — every numbered rule was learned by getting it wrong once.

## 0. Preconditions

- `git fetch origin --tags` **before anything else**. `git worktree add <dir>
  origin/master` resolves the LOCAL `origin/master` ref; without a fresh fetch
  the new branch silently omits the merges you just landed (this cost an extra
  merge cycle and a CHANGELOG conflict on 2026-10-27).
- One writer per worktree. The auto-commit daemon follows **HEAD**, not intent:
  whatever branch is checked out in the shared tree collects the other session's
  in-flight commits. Do queue work in a dedicated worktree
  (`git worktree add /tmp/cqrs-sweep -b <branch> origin/master`), never in the
  shared tree while a sibling session is active.
- Master first. A red base reddens every PR review (`baseRefOid`, rule 5), so
  heal master's own gates before opening a single PR.

## 1. Inventory

```bash
gh pr list --state open --json number,title,headRefName,baseRefOid,statusCheckRollup
gh issue list --state open --json number,title,labels,body
gh api repos/<owner>/<repo>/dependabot/alerts --paginate \
  --jq '.[] | "\(.number) \(.state) \(.security_advisory.severity) \(.dependency.package.name)"'
```

Dependabot alerts are part of the queue, not background noise: an open `high`
is a shipped vulnerability with a known patched version, and the `manifest_path`
row tells you exactly which `go.mod` to sweep (2026-10-28: `go-jose/v4` <=
4.1.5, CVE-2026-116587). One vulnerable pin can appear as several alerts —
verify the manifest really requires the module before trusting the row.

## 2. Read every item IN FULL, then verify AT SOURCE

Never trust the item's own prose, its title, or a prior session's summary.

- **PRs**: `gh pr diff <n>`, then check the claim against the tree (does the
  badge URL actually resolve? does the version exist? does the test fail on the
  old code?).
- **Issues**: reproduce the reported behaviour before implementing the fix, and
  make the regression test fail on the old code first. A test that passes both
  ways is documentation, not a pin.
- **Filing fixes for your own findings**: the issue comment and the CHANGELOG
  entry are receipts; each names the command or test that proves it.

## 3. Tags here are `module/vX.Y.Z`

Every tag in this repo is prefixed (`usermgmt/v4.17.0`, `health/v4.14.0`), so:

- a shields badge filter of `^v4` matches **nothing** — use `?filter=v4.*`;
- any regex over tag lists must strip the prefix before comparing versions
  (this fabricated a 37-release lag on an aligned pin during the treadmill
  bring-up);
- "is vX.Y.Z published?" means `git ls-remote --tags origin | grep '<module>/'`,
  not `git tag -l vX.Y.Z`.

## 4. Heal master's gates before reviewing anything

```bash
nix run .#check-docs-tail-budget -- --ratchet     # live report tail vs TAIL_BASELINE
nix run .#check-release-train -- --refresh-cache --strict-lag 0
nix run .#check-modules
```

Both gates fail on *growth*, so the fix is an archive pass
(`nix run .#archive-status -- <spec.tsv>`) or a dependency sweep
(`scripts/tools/bump-dep.sh '<module>$' <version> --commit --lint`), never a
baseline bump without the accompanying shrink.

## 5. Diagnosing a red PR: `baseRefOid` FIRST

CI checks out base + head. A PR carrying a months-old base inherits every red
that base had, and the checks panel says nothing about it.

```bash
gh pr view <n> --json baseRefOid,headRefOid,mergeable
gh pr update-branch <n>          # only after the base is green
gh run watch <run-id> --exit-status
```

Merge on the **head SHA you actually verified**, not on the PR number: a
concurrent push (or the daemon) can move the head between the watch and the
merge.

## 6. Supersede instead of adopting a polluted branch

When a PR branch carries foreign in-flight work (the daemon banked an unrelated
refactor on it), do not merge the branch to get its good commits:

1. branch fresh from `origin/master`,
2. cherry-pick only the commits that belong to the PR's subject,
3. fix the defects the review found,
4. open a NEW PR that says `Closes #<old>`,
5. comment on the old PR with the reason and the successor number.

Content survives, the unverified refactor does not.

## 7. Closing an issue

Close only when every acceptance box is **verifiable in-tree**:

- code landed and released (name the tag),
- the pinning test exists and fails on the old code,
- any residual is named explicitly in the closing comment (an external
  re-audit is a residual, not a box you can tick).

Demand-gated issues stay open with a comment that names the trigger and the
smallest useful first step — closing them loses the demand signal.

## 8. Report with receipts

- CHANGELOG entry per landed fix (root `Unreleased` for cross-cutting work,
  module `CHANGELOG.md` for published surfaces).
- A `docs/status/` report for the round: what was verified at source, what
  landed with hashes, what is still open and why. Archive an older report in
  the same commit if the tail would grow past `docs/status/TAIL_BASELINE`.
- Delete the sweep's own TODO rows in the same commit that finishes them.

## Failure modes seen in practice

| Symptom                                        | Cause                                                     | Move                                                                    |
| ---------------------------------------------- | --------------------------------------------------------- | ----------------------------------------------------------------------- |
| Push blocked by release-train `rc=3`           | upstream tagged a new release minutes ago                 | run the printed `bump-dep` recipe, re-verify with `--refresh-cache`     |
| `gh` returns `HTTP 401` / "token is invalid"   | keyring/API session dropped; git-over-SSH still works     | keep landing via git; retry `gh auth status` before issue/PR mutations  |
| `git push` "behind its remote counterpart"     | another session pushed during your verification           | `git fetch` + `git rebase origin/master`, re-verify, push again          |
| lint fails with `no go files to analyze`       | shared module/build cache truncated by a full disk        | check `df /mnt/buildcache`; purge the corrupted cache before triage     |
| sweep refuses: "dirty tree … 15m sweep log"    | a concurrent or interrupted `bump-dep` run                | finish or discard that sweep first; never interleave go.mod writes      |
