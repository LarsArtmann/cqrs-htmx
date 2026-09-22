#!/usr/bin/env bash
# tag-message-guard.sh — post-creation tag verification (docs-health D2.1).
#
# A release tag must be an ANNOTATED tag object whose subject names the
# module and version ("<module-path> <version>"). Lightweight tags and
# mislabeled subjects break tooling that reads tag metadata (module-proxy
# checksums, release scripts, ls-remote filtering).
#
# Usage (sourced):
#   verify_tag_message <tagname> <expected-mod> <expected-ver>
#     Prints "verify-tag: message guard OK ..." on success.
#     Prints a "verify-tag:" error to stderr and returns 1 on failure.
# shellcheck shell=bash

verify_tag_message() {
  if [ "$#" -ne 3 ]; then
    echo "verify_tag_message: usage: verify_tag_message <tag> <mod> <ver>" >&2
    return 2
  fi
  local tag="$1" mod="$2" ver="$3" tagtype tagsubj
  tagtype="$(git cat-file -t "$tag" 2>/dev/null || true)"
  if [ "$tagtype" != "tag" ]; then
    echo "verify-tag: $tag is a '${tagtype:-missing}' object, not an annotated tag — refusing to continue" >&2
    return 1
  fi
  tagsubj="$(git tag -l --format='%(contents:subject)' "$tag")"
  if [ "$tagsubj" != "$mod $ver" ]; then
    echo "verify-tag: $tag subject is '$tagsubj', expected '$mod $ver'" >&2
    return 1
  fi
  echo "verify-tag: message guard OK — $tag is annotated with subject '$tagsubj'"
  return 0
}
