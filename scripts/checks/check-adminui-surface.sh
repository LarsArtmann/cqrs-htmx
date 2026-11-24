#!/usr/bin/env bash
# check-adminui-surface.sh — surface-parity gate for the adminui panel (T27/F197).
#
# adminui/routes.go routeSpec rows are the single source of truth: the mux
# registration and the [Handler.Routes] manifest are both generated from the
# table. This gate pins the three textual parities the compiler cannot see:
#
#   (a) table ↔ mux: every routeSpec row-defining function in routes.go is
#       assembled into the table (reachable via an h.<fn>() call in the
#       buildRouteTable chain), and the mux registers ONLY from the table —
#       every mux.Handle/HandleFunc call must carry spec.registration().
#       An orphaned row function is a dead route; a raw registration is a
#       route outside the manifest.
#   (b) nav ↔ table: every sidebar href literal layout.templ renders (the
#       navItem{Href: ...} literals in adminui/handler.go, the sidebar's
#       data source) resolves to a declared route pattern — literal or
#       pattern* const. Verbatim links (external DashboardURL etc.) and
#       dynamic values are skipped. A renamed pattern that leaves a nav
#       label pointing at a dead route goes red here.
#   (c) templ ↔ table: every path literal inside templ href/hx-*/action
#       attributes (adminui/*.templ) points at a declared route: exact
#       pattern, segment-aligned fragment ("/users/" of "/users/{id}"), or
#       route-family suffix ("/delete" of d.BasePath + id + "/delete");
#       "#..." anchors pass. A typo'd href fails here with file:line
#       instead of 404ing in the demo.
#
# The Go goldens (routes_manifest_test.go, check-route-postures.sh) pin
# behavior; this gate pins the parity text so a drift fails without a build.
#
# Gotcha (this host): literal backticks inside awk programs break under
# process substitution (mvdan/sh feeding EOF) — this script avoids them
# entirely; the house escape if one is ever needed is octal "\140".
#
# Env: ADMINUI_SURFACE_ROOT — repo root override (fixture self-tests).
# Exit: 0 parity holds · 1 a check failed · 2 setup.

set -uo pipefail

ROOT="${ADMINUI_SURFACE_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
PKG="$ROOT/adminui"
ROUTES="$PKG/routes.go"
NAV_SRC="$PKG/handler.go"

fail=0
nav_checked=0
templ_checked=0

err() { # <message> — record a red
  echo "FAIL: $*"
  fail=1
}

if [ ! -f "$ROUTES" ]; then
  echo "FAIL: route table missing: $ROUTES"
  exit 2
fi
if [ ! -f "$NAV_SRC" ]; then
  echo "FAIL: sidebar nav source missing: $NAV_SRC"
  exit 2
fi
if ! ls "$PKG"/*.templ >/dev/null 2>&1; then
  echo "FAIL: no templ templates under $PKG"
  exit 2
fi

tmp_patterns="$(mktemp)"
tmp_consts="$(mktemp)"
trap 'rm -f "$tmp_patterns" "$tmp_consts"' EXIT

# pattern* constants (patternUsers = "/users" ...) declared in routes.go.
sed -n '/^const (/,/^)/p' "$ROUTES" |
  grep -E '^\s*pattern[A-Za-z]+\s*=' |
  sed -E 's/^\s*(pattern[A-Za-z]+)\s*=\s*"([^"]*)".*/\1 \2/' >"$tmp_consts"

# Declared patterns: every pattern: row value in the table, consts expanded
# into the pattern set; a value that is neither literal nor known const is
# drift the scan cannot verify. Pattern syntax: single-line rows inline the
# field, multi-line rows start the line with it — match both.
pattern_values="$(mktemp)"
trap 'rm -f "$tmp_patterns" "$tmp_consts" "$pattern_values"' EXIT
grep -oE 'pattern:[[:space:]]*("[^"]*"|pattern[A-Za-z]+)' "$ROUTES" |
  sed -E 's/^pattern:[[:space:]]*//' >"$pattern_values"
rows="$(wc -l <"$pattern_values")"
if [ "$rows" -eq 0 ]; then
  echo "FAIL: no pattern rows found in $ROUTES — scanner broken?"
  exit 2
fi
while IFS= read -r val; do
  case "$val" in
    '"'*)
      printf '%s\n' "${val//\"/}"
      ;;
    pattern*)
      resolved="$(awk -v name="$val" '$1 == name { print $2 }' "$tmp_consts")"
      if [ -n "$resolved" ]; then
        printf '%s\n' "$resolved"
      else
        err "route row references pattern const $val which routes.go does not define"
      fi
      ;;
  esac
done <"$pattern_values" >"$tmp_patterns"
sort -u -o "$tmp_patterns" "$tmp_patterns"

# --- (a) table ↔ mux ---------------------------------------------------------

# Row-defining functions: any *Handler method whose body holds routeSpec rows.
row_fns="$(awk '
  /^func \(h \*Handler\) [A-Za-z]/ {
    cur = $0
    sub(/^func \(h \*Handler\) /, "", cur)
    sub(/\(.*/, "", cur)
    infn = 1
    has = 0
    next
  }
  infn && /routeSpec\{/ { has = 1 }
  infn && /^\}/ {
    if (has) print cur
    infn = 0
  }
' "$ROUTES")"

if [ -z "$row_fns" ]; then
  err "no routeSpec row-defining functions found in routes.go — scanner or table broken?"
fi

for fn in $row_fns; do
  # The definer must be called from the assembly chain (h.<fn>() on a line
  # that is not its own definition), or its rows are served by nothing.
  if ! grep -nF "h.$fn()" "$ROUTES" | grep -qv '^[0-9]*:func '; then
    err "route-table source $fn() is defined but never assembled into the table — its rows are dead routes"
  fi
done

# The mux may only register from the table: every registration expression
# must be spec.registration() (the table's canonical mux form).
bad_mux="$(grep -nE 'mux\.(Handle|HandleFunc)\(' "$ROUTES" | grep -v 'spec.registration()' || true)"
if [ -n "$bad_mux" ]; then
  err "routes.go registers a route OUTSIDE the route table (no spec.registration()):"
  printf '%s\n' "$bad_mux" | sed 's/^/      /'
fi
if ! grep -q 'mux.Handle(spec.registration()' "$ROUTES"; then
  err "registerRoutes no longer serves the table (mux.Handle(spec.registration()) missing) — the manifest and the mux can drift apart"
fi

# --- (b) nav ↔ table ---------------------------------------------------------

nav_lits="$(grep -c '{Href:' "$NAV_SRC" || true)"
if [ "$nav_lits" -eq 0 ]; then
  err "no navItem{Href: ...} literals found in $NAV_SRC — the sidebar scan verified nothing"
fi
while IFS= read -r line; do
  lineno="${line%%:*}"
  body="${line#*:}"
  case "$body" in *'Verbatim: true'*) continue ;; esac
  href="$(printf '%s' "$body" | sed -E 's/.*\{Href: ([^,]*),.*/\1/')"
  target=""
  case "$href" in
    '"'*)
      target="${href//\"/}"
      ;;
    pattern*)
      target="$(awk -v name="$href" '$1 == name { print $2 }' "$tmp_consts")"
      if [ -z "$target" ]; then
        err "nav item references const $href which routes.go does not define (handler.go:$lineno)"
        continue
      fi
      ;;
    *) continue ;; # dynamic (entry.Href, config.DashboardURL)
  esac
  nav_checked=$((nav_checked + 1))
  if ! grep -qxF "$target" "$tmp_patterns"; then
    err "sidebar nav href \"$target\" (handler.go:$lineno) matches NO route table row — nav pointing at a dead route"
  fi
done < <(grep -n '{Href:' "$NAV_SRC")

# --- (c) templ ↔ table -------------------------------------------------------

is_declared_route() { # <literal> — exact, segment-aligned fragment, or family suffix
  lit="$1"
  lit="${lit%%\?*}" # strip query strings: "/audit?q=" filters as "/audit"
  case "$lit" in
    '#'*) return 0 ;; # in-page anchor
  esac
  if grep -qxF "$lit" "$tmp_patterns"; then
    return 0
  fi
  # Segment-aligned fragment: the literal occurs in some pattern with
  # route-boundary neighbors ("/", "{", "}", string edges).
  if awk -v lit="$lit" '
    {
      i = index($0, lit)
      if (i > 0) {
        before = (i == 1) ? "" : substr($0, i - 1, 1)
        j = i + length(lit)
        after = (j > length($0)) ? "" : substr($0, j, 1)
        if ((before == "" || before == "/" || before == "{" || before == "}") &&
          (after == "" || after == "/" || after == "{" || after == "}")) {
          found = 1
          exit
        }
      }
    }
    END { exit(found ? 0 : 1) }
  ' "$tmp_patterns"; then
    return 0
  fi
  # Route-family suffix: the trailing segment(s) of a fluent URL build
  # (d.BasePath + id + "/delete" → "/delete" of "/users/{id}/delete").
  [ -n "$(grep -E "${lit}\$" "$tmp_patterns" | head -1)" ]
}

# Attribute forms: href={ expr }, hx-post={ expr }, and the templ attribute
# map style "hx-post": expr,
attr_re='(href|hx-get|hx-post|hx-put|hx-patch|hx-delete|action)='
attr_map_re='"(href|hx-get|hx-post|hx-put|hx-patch|hx-delete|action)":'
for f in "$PKG"/*.templ; do
  while IFS= read -r line; do
    lineno="${line%%:*}"
    body="${line#*:}"
    if ! printf '%s' "$body" | grep -qE "$attr_re|$attr_map_re"; then
      continue
    fi
    while IFS= read -r lit; do
      [ -n "$lit" ] || continue
      case "$lit" in
        /* | '#'*) : ;;
        *) continue ;;
      esac
      templ_checked=$((templ_checked + 1))
      if ! is_declared_route "$lit"; then
        err "templ href \"$lit\" ($f:$lineno) points at no declared route — dead link on the panel surface"
      fi
    done < <(printf '%s' "$body" | grep -oE '"[^"]*"' | tr -d '"' | sort -u)
  done < <(grep -nE "$attr_re|$attr_map_re" "$f")
done

# --- false-green guards (a sweep that verifies nothing must fail) ------------

if [ "$rows" -lt 30 ]; then
  echo "FAIL: expected at least 30 route table rows, found $rows — scanner broken?"
  exit 1
fi
if [ "$(wc -l <"$tmp_patterns")" -lt 30 ]; then
  echo "FAIL: expected at least 30 distinct route patterns, found $(wc -l <"$tmp_patterns") — scanner broken?"
  exit 1
fi
if [ "$nav_checked" -lt 5 ]; then
  echo "FAIL: expected at least 5 nav href literals checked, found $nav_checked — scanner broken?"
  exit 1
fi
if [ "$templ_checked" -lt 25 ]; then
  echo "FAIL: expected at least 25 templ href literals checked, found $templ_checked — scanner broken?"
  exit 1
fi

summary="surface: rows=$rows patterns=$(wc -l <"$tmp_patterns") nav=$nav_checked templ-literals=$templ_checked"
if [ "$fail" -ne 0 ]; then
  echo "$summary"
  exit 1
fi
echo "$summary"
echo "OK: adminui surface parity holds — route table, sidebar nav, and templ hrefs all agree"
