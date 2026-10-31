#!/usr/bin/env python3
"""check-pr-secrets.py — no-permanently-red-PR gate (round-45 M16).

Invariant: a job that runs on pull_request and consumes a `secrets.*`
reference MUST be skip-gated on the secret's presence (the
architecture-gate-preflight shape: a probe job publishes `has_key`, the
consuming job's `if:` requires it). An unconditional pull_request job that
consumes an unconfigured secret reds EVERY PR until someone configures it —
a permanent red that trains everyone to ignore the checks.

Offline text scan (no YAML dependency): walks each workflow's `jobs:` block,
tracks job boundaries by 2-space indentation, and asserts the invariant.
Skips `schedule`/`push`-only workflows (secrets are configured there by
definition — repo secrets exist on the default branch).

Fixture self-test: scripts/selftests/test-check-pr-secrets.sh
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

WORKFLOWS = Path(".github/workflows")
JOB_RE = re.compile(r"^  ([A-Za-z0-9_-]+):\s*(?:#.*)?$")
SECRETS_RE = re.compile(r"secrets\.[A-Za-z0-9_]+")
SKIP_IF_RE = re.compile(r"needs\.\S+\.outputs\.\w+\s*==\s*'(true|false)'")
STEP_USES_SECRET_RE = re.compile(r"^\s+(?:- )?(?:uses|env:|run:)|^\s+\S[^\n]*secrets\.")


def workflow_triggers(text: str) -> set[str]:
    """Return the `on:` trigger names (shallow parse)."""
    triggers: set[str] = set()
    in_on = False
    on_indent = 0
    for line in text.splitlines():
        if re.match(r"^(on|\"on\"):\s*(\[|\s*$)", line):
            in_on = True
            on_indent = 0
            inline = re.search(r"\[(.*?)\]", line)
            if inline:
                triggers.update(
                    t.strip().strip("'\"") for t in inline.group(1).split(",")
                )
                in_on = False
            continue
        if in_on:
            m = re.match(r"^\s+([a-z_]+):\s*$", line)
            if m:
                if on_indent == 0:
                    on_indent = len(m.group(0)) - len(m.group(0).lstrip())
                triggers.add(m.group(1))
            elif line.strip() and not line.startswith("      "):
                in_on = False
    return triggers


def parse_jobs(text: str) -> list[dict]:
    """Shallow-parse the jobs: block into {name, if, body} dicts."""
    jobs: list[dict] = []
    lines = text.splitlines()
    in_jobs = False
    current: dict | None = None
    body_indent = None
    for line in lines:
        if re.match(r"^jobs:\s*(#.*)?$", line):
            in_jobs = True
            continue
        if not in_jobs:
            continue
        m = JOB_RE.match(line)
        if m:
            if current:
                jobs.append(current)
            current = {"name": m.group(1), "if": "", "body": []}
            body_indent = None
            continue
        if current is None:
            continue
        indent = len(line) - len(line.lstrip())
        if line.strip() and indent <= 2:
            # next top-level key ends the jobs block
            if not line.startswith("  "):
                jobs.append(current)
                current = None
                in_jobs = False
                continue
        if current is not None:
            if body_indent is None and line.strip():
                body_indent = indent
            if line.strip().startswith("if:") and indent == body_indent:
                current["if"] = line.split("if:", 1)[1].strip()
            current["body"].append(line)
    if current:
        jobs.append(current)
    return jobs


def check_workflow(path: Path) -> list[str]:
    problems: list[str] = []
    text = path.read_text()
    triggers = workflow_triggers(text)
    if "pull_request" not in triggers:
        return problems
    for job in parse_jobs(text):
        if not any(SECRETS_RE.search(ln) for ln in job["body"]):
            continue
        if SKIP_IF_RE.search(job["if"]):
            continue
        flows_to_action = False
        in_action_with = False
        for ln in job["body"]:
            stripped = ln.strip()
            if re.match(r"-?\s*uses:", stripped):
                in_action_with = True
                continue
            if re.match(r"-?\s*(run:|name:)", stripped):
                in_action_with = False
                continue
            if in_action_with and SECRETS_RE.search(ln):
                flows_to_action = True
        publishes_output = any("GITHUB_OUTPUT" in ln for ln in job["body"])
        # Secret -> action input (ssh-agent & co): fails the job outright
        # when unconfigured. Env-only reads publish empties, so they are only
        # acceptable in the PROBE shape (presence published via GITHUB_OUTPUT).
        if flows_to_action or not publishes_output:
            first = next((ln.strip() for ln in job["body"] if "secrets." in ln), "?")
            problems.append(
                f"{path.name}: job '{job['name']}' consumes secrets unconditionally "
                f"(first use: {first}) — add the probe+skip-gate shape "
                f"(see architecture-gate-preflight in ci.yml)"
            )
    return problems


def main() -> int:
    if not WORKFLOWS.is_dir():
        print("check-pr-secrets: no .github/workflows — nothing to check")
        return 0
    all_problems: list[str] = []
    for wf in sorted(WORKFLOWS.glob("*.y*ml")):
        all_problems.extend(check_workflow(wf))
    if all_problems:
        for p in all_problems:
            print(f"  ✗ {p}")
        print(f"check-pr-secrets: {len(all_problems)} permanently-red PR risk(s)")
        return 1
    print("check-pr-secrets: every pull_request secrets consumer is skip-gated")
    return 0


if __name__ == "__main__":
    sys.exit(main())
