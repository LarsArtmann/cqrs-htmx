{
  description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;

          goEnv = ''
            export GOWORK=off
            export GOPRIVATE='github.com/larsartmann/*,github.com/LarsArtmann/*'
            export GOEXPERIMENT=jsonv2
            export GOTOOLCHAIN=local

            # forEachGoModule: iterate over all workspace modules from go.work.
            # Usage: forEachGoModule "command" [exclude_regex]
            # The root module (.) runs without cd. Optional 2nd arg skips dirs
            # matching the regex (e.g. '^(e2e/|examples/)' to skip e2e/examples).
            forEachGoModule() {
              local cmd="$1"
              local exclude="''${2:-}"
              while IFS= read -r dir; do
                dir="''${dir#./}"
                if [ -n "$exclude" ] && echo "$dir" | grep -qE "$exclude"; then
                  continue
                fi
                if [ -z "$dir" ] || [ "$dir" = "." ]; then
                  echo "==> Root module"
                  eval "$cmd"
                else
                  echo "==> $dir"
                  (cd "$dir" && eval "$cmd")
                fi
              done < <(env GOWORK= go work edit -json 2>/dev/null | jq -r '.Use[].DiskPath')
            }
          '';

          goApp =
            {
              name,
              text,
              description ? null,
              runtimeInputs ? [ ],
            }:
            {
              type = "app";
              meta = lib.optionalAttrs (description != null) { inherit description; };
              program = pkgs.writeShellApplication {
                inherit name;
                runtimeInputs = [
                  goPkg
                  pkgs.jq
                ]
                ++ runtimeInputs;
                text = goEnv + text;
              };
            };
          # benchstat (golang.org/x/perf/cmd/benchstat) is not packaged in
          # nixpkgs; build it from the canonical googlesource repo.
          # To bump: update `rev` + `version`, then fix src.hash and
          # vendorHash from the `nix build .#benchstat` error messages.
          benchstat = pkgs.buildGoModule {
            pname = "benchstat";
            version = "0.0.0-20260825160852";

            src = pkgs.fetchgit {
              url = "https://go.googlesource.com/perf";
              rev = "19be9d8e6c701dc8ccabaad34bf705f773fd398b";
              hash = "sha256-CimaQbwjQ5SMl/VTzuMeSciOp7aSomGbT/iyEsguOCg=";
            };

            subPackages = [ "cmd/benchstat" ];
            vendorHash = "sha256-AZx9tPzsPvjc5kpmiBa6eYKtrw0hczYi0sbcd/lkiiA=";
            ldflags = [
              "-s"
              "-w"
            ];
          };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              nixfmt.enable = true;
              templ.enable = true;
              gofmt.enable = true;
              # golines/shfmt/shellcheck evaluated 2026-08-29: first run
              # reformats 275 files (treefmt golines defaults disagree with
              # the golangci-lint golines config; shfmt+shellcheck flag
              # legacy gate scripts). Enabling needs a DEDICATED format-sweep
              # commit with reviewed fallout — tracked in TODO_LIST P3.
            };
          };

          packages.default = pkgs.stdenvNoCC.mkDerivation {
            pname = "cqrs-htmx";
            version = self.rev or self.dirtyRev or "dev";

            dontUnpack = true;
            dontConfigure = true;
            dontBuild = true;

            installPhase = ''
              mkdir -p $out
            '';

            meta = with lib; {
              description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";
              homepage = "https://github.com/larsartmann/cqrs-htmx";
              license = licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              platforms = platforms.unix;
            };
          };

          packages.benchstat = benchstat;

          devShells = {
            default = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.gopls
                pkgs.golangci-lint
                pkgs.govulncheck
                benchstat # benchmark comparison (golang.org/x/perf; used by nix run .#bench-spike)
                pkgs.ginkgo
                pkgs.tailwindcss_4
                pkgs.templ
                # BuildFlow pre-commit hook formatters/linters
                pkgs.shfmt # shell scripts (scripts/*.sh)
                pkgs.nixfmt # flake.nix
                pkgs.dprint # markdown-format
                pkgs.prettier # YAML/JSON/CSS/config files
                pkgs.biome # JS/TS files (admin.js, login.js, sync-*.js)
                pkgs.codespell # spell check across all files
                pkgs.treefmt # nix fmt aggregator
                # BuildFlow pre-commit steps that failed on missing binaries
                pkgs.typescript # tsc (type-check + tsconfig-check)
                pkgs.go-licenses # license-check
                pkgs.vulnix # NixOS vulnerability scan
              ];

              GOWORK = "off";
              GOPRIVATE = "github.com/larsartmann/*,github.com/LarsArtmann/*";
              GOEXPERIMENT = "jsonv2";
              GOTOOLCHAIN = "local";
            };

            ci = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.golangci-lint
                pkgs.templ
              ];

              GOWORK = "off";
              GOPRIVATE = "github.com/larsartmann/*,github.com/LarsArtmann/*";
              GOEXPERIMENT = "jsonv2";
              GOTOOLCHAIN = "local";
            };
          };

          checks = {
            build = config.packages.default;
            formatting = config.treefmt.build.check self;
          };

          apps = {
            test = goApp {
              name = "run-tests";
              description = "Run Go tests with race detector across all workspace modules (auto-discovered, excludes e2e/examples)";
              text = ''
                forEachGoModule "go test ./... -count=1 -race" '^(e2e/|examples/)'
              '';
            };

            test-race = goApp {
              name = "run-tests-race";
              description = "Run all Go tests with the race detector across all workspace modules (auto-discovered, excludes e2e/examples)";
              text = ''
                forEachGoModule "go test ./... -count=1 -race" '^(e2e/|examples/)'
              '';
            };

            test-flake = goApp {
              name = "run-tests-flake";
              description = "Run all Go tests 3x with race detector to detect flaky tests (auto-discovered, excludes e2e/examples)";
              text = ''
                runFlake() {
                  local i
                  for i in 1 2 3; do
                    echo "  (flake run $i/3)"
                    go test ./... -count=1 -race
                  done
                }
                forEachGoModule "runFlake" '^(e2e/|examples/)'
              '';
            };

            test-fuzz = goApp {
              name = "run-tests-fuzz";
              description = "Run all Go fuzz tests across all workspace modules (auto-discovered, FUZZTIME env var, default 30s)";
              text = ''
                FUZZTIME="''${FUZZTIME:-30s}"
                runModuleFuzz() {
                  local pkg fuzzList fuzz
                  for pkg in $(go list ./... 2>/dev/null || true); do
                    fuzzList=$(go test -run='^$' -list='Fuzz.*' "$pkg" 2>/dev/null | grep '^Fuzz' || true)
                    for fuzz in $fuzzList; do
                      echo "    -> $fuzz ($pkg)"
                      go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" "$pkg"
                    done
                  done
                }
                forEachGoModule "runModuleFuzz" '^(e2e/|examples/)'
              '';
            };

            lint = goApp {
              name = "run-lint";
              description = "Run golangci-lint across all workspace modules (auto-discovered, excludes e2e/examples)";
              runtimeInputs = [ pkgs.golangci-lint ];
              text = ''
                lintFail=0
                while IFS= read -r dir; do
                  dir="''${dir#./}"
                  if [ -n "$dir" ] && echo "$dir" | grep -qE '^(e2e/|examples/)'; then
                    continue
                  fi
                  if [ -z "$dir" ] || [ "$dir" = "." ]; then
                    echo "==> Root module"
                    dir="."
                  else
                    echo "==> $dir"
                  fi
                  if ! (cd "$dir" && golangci-lint run); then
                    lintFail=1
                  fi
                done < <(env GOWORK= go work edit -json 2>/dev/null | jq -r '.Use[].DiskPath')
                if [ "$lintFail" -eq 1 ]; then
                  echo "FAIL: one or more modules had lint issues"
                  exit 1
                fi
              '';
            };

            coverage = goApp {
              name = "run-coverage";
              description = "Run Go tests with coverage across all workspace modules (auto-discovered, excludes e2e/examples)";
              text = ''
                forEachGoModule "go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out" '^(e2e/|examples/)'
              '';
            };

            build = goApp {
              name = "run-build";
              description = "Build all workspace modules (auto-discovered from go.work)";
              text = ''
                forEachGoModule "go build ./..."
                echo "All modules built successfully."
              '';
            };

            bench-spike = goApp {
              name = "run-bench-spike";
              description = "Run the setup spike benchmark (default 5x2s) and fail on a >10% median ns/op regression vs docs/benchmarks/setup-baseline.raw.txt (10% default: run-to-run noise on the pinned machine measures up to ~9% on the ~1us json-roundtrip sub-bench; tighten via BENCH_SPIKE_THRESHOLD). Env: BENCH_COUNT, BENCHTIME, BENCH_SPIKE_THRESHOLD. --save-baseline [path] re-pins the baseline.";
              runtimeInputs = [
                benchstat
                pkgs.git
                pkgs.coreutils
                pkgs.gawk
                pkgs.gnugrep
              ];
              text = ''
                baseline="docs/benchmarks/setup-baseline.raw.txt"
                # 10% default: measured run-to-run median noise on the pinned
                # machine reaches ~9% on the ~1us json-roundtrip sub-bench and
                # ~5% on the ~20us HTTP sub-benches. A noisier gate is a worse
                # gate; real regressions (the 2.8x logging bug class) clear 10%
                # by an order of magnitude. Tighten with BENCH_SPIKE_THRESHOLD.
                threshold="''${BENCH_SPIKE_THRESHOLD:-10}"
                count="''${BENCH_COUNT:-5}"
                benchtime="''${BENCHTIME:-2s}"

                # Fall back to /tmp when the ambient build cache is unwritable
                # (e.g. a dead secondary disk), so the gate never fails on cache
                # init. Shared guard, same lib the isolation script sources.
                source scripts/lib/go-cache-env.sh
                bench_args=(
                  -run xxx
                  -bench "^BenchmarkSpikeBaselineVsAppkit$"
                  -benchtime="$benchtime"
                  -benchmem
                  -count="$count"
                  -timeout=120s
                )

                if [ "''${1:-}" = "--save-baseline" ]; then
                  out="''${2:-$baseline}"
                  echo "== bench-spike: pinning baseline to $out ($count x $benchtime) =="
                  (cd setup && go test "''${bench_args[@]}" .) | tee "$out"
                  echo "Saved $out — commit it so everyone gates against the same numbers."
                  exit 0
                fi

                if [ ! -f "$baseline" ]; then
                  echo "bench-spike: no baseline at $baseline." >&2
                  echo "  Pin one on this machine first: nix run .#bench-spike -- --save-baseline" >&2
                  exit 2
                fi

                current="$(mktemp /tmp/bench-spike-current.XXXXXX)"
                trap 'rm -f "$current"' EXIT

                echo "== bench-spike: $count x $benchtime vs $baseline =="
                (cd setup && go test "''${bench_args[@]}" .) | tee "$current"

                echo
                echo "== benchstat: baseline -> current =="
                benchstat "$baseline" "$current" || true

                # ns/op is only comparable within the same environment; refuse
                # to gate across machines instead of producing a false verdict.
                for key in goos goarch pkg cpu; do
                  b="$(grep -m1 "^$key:" "$baseline" | cut -d' ' -f2-)"
                  c="$(grep -m1 "^$key:" "$current" | cut -d' ' -f2-)"
                  if [ "$b" != "$c" ]; then
                    echo "bench-spike: environment mismatch for '$key': baseline='$b' current='$c'" >&2
                    echo "  ns/op is not comparable across machines. Re-pin the baseline here:" >&2
                    echo "    nix run .#bench-spike -- --save-baseline" >&2
                    exit 2
                  fi
                done

                medians() {
                  awk '/^Benchmark/ && /ns\/op/ {
                    name = $1
                    sub(/-[0-9]+$/, "", name)
                    for (i = 2; i <= NF; i++) if ($i == "ns/op") { v = $(i - 1); break }
                    vals[name] = vals[name] " " v
                  }
                  END {
                    for (n in vals) {
                      m = split(vals[n], a, " ")
                      for (i = 1; i <= m; i++)
                        for (j = i + 1; j <= m; j++)
                          if ((a[j] + 0) < (a[i] + 0)) { t = a[i]; a[i] = a[j]; a[j] = t }
                      mid = int((m + 1) / 2)
                      med = (m % 2 == 1) ? a[mid] : (a[mid] + a[mid + 1]) / 2
                      printf "%s %.6f\n", n, med
                    }
                  }' "$1"
                }

                joined="$(join -j 1 <(medians "$baseline" | sort) <(medians "$current" | sort))"
                if [ -z "$joined" ]; then
                  echo "bench-spike: no overlapping benchmark names between baseline and current run." >&2
                  echo "  The benchmark set changed; re-pin: nix run .#bench-spike -- --save-baseline" >&2
                  exit 2
                fi

                regressed=0
                while read -r name old new; do
                  delta="$(awk -v o="$old" -v n="$new" 'BEGIN { printf "%+.1f", (n - o) / o * 100 }')"
                  printf '  %-50s median %10.1f -> %10.1f ns/op (%s%%)\n' "$name" "$old" "$new" "$delta"
                  over="$(awk -v o="$old" -v n="$new" -v t="$threshold" 'BEGIN { print (((n - o) / o * 100) > t) ? 1 : 0 }')"
                  if [ "$over" = "1" ]; then
                    echo "  REGRESSION: '$name' regressed more than $threshold% vs baseline" >&2
                    regressed=1
                  fi
                done <<< "$joined"

                if [ "$regressed" -eq 1 ]; then
                  echo "bench-spike: FAIL (median ns/op regression > $threshold%)" >&2
                  exit 1
                fi
                echo "bench-spike: OK (no median ns/op regression > $threshold%)"
              '';
            };

            test-root = goApp {
              name = "test-root";
              description = "Run the root module's Go tests in isolation";
              text = ''
                go test ./... -count=1 -race "$@"
              '';
            };

            test-usermgmt = goApp {
              name = "test-usermgmt";
              description = "Run the usermgmt submodule's Go tests in isolation";
              text = ''
                cd usermgmt
                go test ./... -count=1 -race "$@"
              '';
            };

            test-adminui = goApp {
              name = "test-adminui";
              description = "Run the adminui submodule's Go tests in isolation";
              text = ''
                cd adminui
                go test ./... -count=1 -race "$@"
              '';
            };

            test-loginpage = goApp {
              name = "test-loginpage";
              description = "Run the loginpage submodule's Go tests in isolation";
              text = ''
                cd loginpage
                go test ./... -count=1 -race "$@"
              '';
            };

            test-dashboardui = goApp {
              name = "test-dashboardui";
              description = "Run the dashboardui submodule's Go tests in isolation";
              text = ''
                cd dashboardui
                go test ./... -count=1 -race "$@"
              '';
            };

            test-integration = goApp {
              name = "test-integration";
              description = "Run the integration_test module's Go tests in isolation";
              text = ''
                cd integration_test
                go test ./... -count=1 -race "$@"
              '';
            };

            test-totp = goApp {
              name = "test-totp";
              description = "Run the usermgmt/totp submodule's Go tests in isolation";
              text = ''
                cd usermgmt/totp
                go test ./... -count=1 -race "$@"
              '';
            };

            test-webauthn = goApp {
              name = "test-webauthn";
              description = "Run the usermgmt/webauthn submodule's Go tests in isolation";
              text = ''
                cd usermgmt/webauthn
                go test ./... -count=1 -race "$@"
              '';
            };

            test-oauth2 = goApp {
              name = "test-oauth2";
              description = "Run the usermgmt/oauth2 submodule's Go tests in isolation";
              text = ''
                cd usermgmt/oauth2
                go test ./... -count=1 -race "$@"
              '';
            };

            build-datastar-demo = goApp {
              name = "build-datastar-demo";
              description = "Build the datastar-demo example binary";
              text = ''
                cd examples/datastar-demo
                go build ./... "$@"
              '';
            };

            build-admin-demo = goApp {
              name = "build-admin-demo";
              description = "Build the admin-demo example binary (runnable admin panel showcase)";
              text = ''
                cd examples/admin-demo
                go build ./... "$@"
              '';
            };

            build-dashboard-demo = goApp {
              name = "build-dashboard-demo";
              description = "Build the dashboard-demo example binary";
              text = ''
                cd examples/dashboard-demo
                go build ./... "$@"
              '';
            };

            build-catalog-demo = goApp {
              name = "build-catalog-demo";
              description = "Build the catalog-demo example binary";
              text = ''
                cd examples/catalog-demo
                go build ./... "$@"
              '';
            };

            build-adminui-css = {
              type = "app";
              meta.description = "Compile adminui Tailwind v4 CSS (tailwind.css → assets/admin-tw.css)";
              program = pkgs.writeShellApplication {
                name = "build-adminui-css";
                runtimeInputs = [
                  pkgs.tailwindcss_4
                  goPkg
                ];
                text = ''
                  cd adminui
                  # Resolve templ-components module dir at build time.
                  TC_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components 2>/dev/null || true)

                  TMP_CSS=$(mktemp --suffix=.css)
                  cp tailwind.css "$TMP_CSS"

                  # Add @source for adminui itself (the temp CSS lives in
                  # /tmp so Tailwind's auto-detection won't find our .templ
                  # files without this).
                  ADMINUI_DIR=$(pwd)
                  echo "@source \"$ADMINUI_DIR\";" >> "$TMP_CSS"

                  if [ -n "$TC_DIR" ]; then
                    # Copy ONLY .templ files to a temp dir.
                    # _templ.go are generated mirrors (same class strings,
                    # 3x larger). Scanning the full module cache causes
                    # 55 GB RAM usage — this approach uses <500 MB.
                    SCAN_DIR=$(mktemp -d)
                    for pkg in display errorpage feedback forms htmx icons layout navigation; do
                      if [ -d "$TC_DIR/$pkg" ]; then
                        cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                      fi
                    done
                    echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS"
                  fi

                  tailwindcss -i "$TMP_CSS" -o assets/admin-tw.css --minify

                  rm -f "$TMP_CSS"
                  [ -n "''${SCAN_DIR:-}" ] && rm -rf "$SCAN_DIR"
                  echo "Done: adminui/assets/admin-tw.css"
                '';
              };
            };

            gen = {
              type = "app";
              meta.description = "Regenerate adminui templ components and normalize formatting";
              program = pkgs.writeShellApplication {
                name = "templ-generate";
                runtimeInputs = [
                  goPkg
                  pkgs.templ
                ];
                text = ''
                  cd adminui
                  templ generate
                  gofmt -w *_templ.go
                  echo "Done: adminui templ components regenerated and formatted"
                '';
              };
            };

            render-diagrams = {
              type = "app";
              meta.description = "Render all .d2 source files under docs/ to SVG (dark canvas → theme 200, light → default)";
              program = pkgs.writeShellApplication {
                name = "render-diagrams";
                runtimeInputs = [ pkgs.d2 ];
                text = ''
                  shopt -s nullglob
                  found=0
                  for file in docs/**/*.d2 docs/*.d2; do
                    [ -f "$file" ] || continue
                    found=1
                    out="''${file%.d2}.svg"
                    if sed -n '1,10p' "$file" | grep -qE 'style:\s*\{[^}]*fill:\s*"#[01][0-9a-fA-F]{5}"'; then
                      echo "[dark]  $file"
                      d2 --layout=elk --theme=200 "$file" "$out"
                    else
                      echo "[light] $file"
                      d2 --layout=elk "$file" "$out"
                    fi
                  done
                  if [ "$found" -eq 0 ]; then
                    echo "No .d2 files found under docs/"
                    exit 1
                  fi
                '';
              };
            };

            errorfamily = {
              type = "app";
              meta.description = "Verify all errors use go-error-family constructors (no stdlib errors.New/fmt.Errorf/errors.Join)";
              program = pkgs.writeShellApplication {
                name = "check-errorfamily";
                runtimeInputs = [ pkgs.go ];
                text = ''
                  # Root + usermgmt + adminui + identity-model + dashboardui + loginpage + datastar:
                  # error-family constructors are mandatory in non-test code.
                  # Auth sub-modules (totp/webauthn/oauth2) are intentionally exempt:
                  # they don't import go-cqrs-lite/event/v4 (keeping deps minimal), and
                  # the Service layer wraps all provider errors with event.Wrapf at the
                  # boundary — so error families are assigned at the correct layer.
                  #
                  # Uses a Go AST-based scanner (go/parser) instead of ripgrep, which
                  # inherently ignores ALL comment types (//, /* */, inline, multi-line).
                  set -euo pipefail
                  export GOWORK=off
                  export GOEXPERIMENT=jsonv2

                  check_module() {
                    local dir="$1"
                    local name="$2"
                    echo "==> $name"
                    go run scripts/errorfamily_scanner.go "$dir"
                    echo "  OK"
                  }

                  check_module "." "Root module"
                  check_module "usermgmt" "usermgmt submodule"
                  check_module "adminui" "adminui submodule"
                  check_module "identity-model" "identity-model submodule"
                  check_module "dashboardui" "dashboardui submodule"
                  check_module "loginpage" "loginpage submodule"
                  check_module "datastar" "datastar submodule"

                  echo "All modules pass errorfamily check."
                '';
              };
            };

            check-modules = {
              type = "app";
              meta.description = "Run all module architecture checks (isolation, dep budgets, version drift, replace directives)";
              program = pkgs.writeShellApplication {
                name = "check-modules";
                runtimeInputs = [ goPkg ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-module-isolation.sh
                  bash scripts/check-dep-budgets.sh
                  bash scripts/check-go-toolchain.sh
                  bash scripts/check-version-drift.sh --strict
                  bash scripts/check-release-train.sh
                  bash scripts/check-replace-directives.sh
                  bash scripts/check-docs-freshness.sh
                  bash scripts/check-docs-links.sh
                  echo ""
                  echo "✓ All module architecture checks passed"
                '';
              };
            };

            check-docs-freshness = {
              type = "app";
              meta.description = "Scan .md files for version strings that don't match go.mod";
              program = pkgs.writeShellApplication {
                name = "check-docs-freshness";
                runtimeInputs = [ goPkg ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-docs-freshness.sh
                '';
              };
            };

            check-docs-links = {
              type = "app";
              meta.description = "Check all markdown file-path links resolve correctly";
              program = pkgs.writeShellApplication {
                name = "check-docs-links";
                runtimeInputs = [
                  pkgs.findutils
                  pkgs.gnugrep
                ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-docs-links.sh
                '';
              };
            };

            check-release-train = {
              type = "app";
              meta.description = "Verify every internal require resolves to a PUBLISHED tag; list train-lag for the next family train";
              program = pkgs.writeShellApplication {
                name = "check-release-train";
                runtimeInputs = [
                  pkgs.git
                  pkgs.coreutils
                  pkgs.gnugrep
                  pkgs.gawk
                  pkgs.findutils
                ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-release-train.sh
                '';
              };
            };

            check-codegen = {
              type = "app";
              meta.description = "Verify adminui + loginpage _templ.go files match .templ sources (no codegen drift)";
              program = pkgs.writeShellApplication {
                name = "check-codegen";
                runtimeInputs = [
                  goPkg
                  pkgs.templ
                ];
                text = ''
                  for mod in adminui loginpage; do
                    echo "==> $mod"
                    (cd "$mod" && templ generate && gofmt -w ./*_templ.go)
                    if ! git diff --exit-code -- "$mod"/*_templ.go; then
                      echo ""
                      echo "FAIL: Generated _templ.go files in $mod differ from committed versions."
                      echo "Run 'templ generate' in $mod/ and commit the result."
                      exit 1
                    fi
                  done
                  echo "Codegen drift check PASSED"
                '';
              };
            };

            check-templates = {
              type = "app";
              meta.description = "Verify //go:build ignore SQL setup template files compile (sqlite/postgres/mysql)";
              program = pkgs.writeShellApplication {
                name = "check-templates";
                runtimeInputs = [ goPkg ];
                text = ''
                  bash scripts/check-templates.sh
                '';
              };
            };

            coverage-gate = goApp {
              name = "coverage-gate";
              description = "Run tests and fail if coverage drops below thresholds";
              runtimeInputs = [ pkgs.bc ];
              text = ''
                fail=0
                check_cov() {
                  local mod="$1" threshold="$2"
                  local cov
                  cov=$(cd "$mod" && go test ./... -count=1 -coverprofile=/tmp/cov >/dev/null 2>&1 && go tool cover -func=/tmp/cov | tail -1 | grep -oP '\d+\.\d+(?=%)')
                  echo "$mod coverage: ''${cov}% (threshold: ''${threshold}%)"
                  if (( $(echo "$cov < $threshold" | bc -l) )); then
                    echo "FAIL: $mod coverage ''${cov}% < ''${threshold}%"
                    fail=1
                  fi
                }
                check_cov . 90
                check_cov identity-model 70
                check_cov usermgmt 74
                check_cov usermgmt/totp 80
                check_cov usermgmt/webauthn 80
                check_cov usermgmt/oauth2 80
                check_cov adminui 66
                check_cov loginpage 79
                check_cov dashboardui 60
                check_cov datastar 90
                check_cov setup 80
                check_cov systemadapter 70
                check_cov health 90
                check_cov auditlog 90
                # Per-package gate: dashboardui/core is the pure data layer.
                core_cov=$(cd dashboardui && go test ./core/... -count=1 -coverprofile=/tmp/corecov >/dev/null 2>&1 && go tool cover -func=/tmp/corecov | tail -1 | grep -oP '\d+\.\d+(?=%)')
                echo "dashboardui/core coverage: ''${core_cov}% (threshold: 80%)"
                if (( $(echo "$core_cov < 80" | bc -l) )); then
                  echo "FAIL: dashboardui/core coverage ''${core_cov}% < 80%"
                  fail=1
                fi
                if [ "$fail" -eq 1 ]; then
                  echo "Coverage gate FAILED"
                  exit 1
                fi
                echo "Coverage gate PASSED"
              '';
            };

            release-checklist = {
              type = "app";
              meta.description = "Pre-release verification: CHANGELOG, versions, builds, git status";
              program = pkgs.writeShellApplication {
                name = "release-checklist";
                runtimeInputs = [ goPkg ];
                text = ''
                  bash scripts/release-checklist.sh
                '';
              };
            };

            e2e = {
              type = "app";
              meta.description = "Run Playwright E2E tests (offline sync) against the local Go test server";
              program = pkgs.writeShellApplication {
                name = "e2e";
                runtimeInputs = [
                  goPkg
                  pkgs.nodejs
                  pkgs.curl
                ]
                ++ pkgs.lib.optional (pkgs ? chromium) pkgs.chromium;
                text = ''
                  export GOEXPERIMENT=jsonv2
                  # On NixOS, Playwright's downloaded Chromium cannot run (no FHS linker).
                  # Use the Nix-packaged Chromium via E2E_BROWSER_PATH.
                  if [ -z "''${E2E_BROWSER_PATH:-}" ] && command -v chromium >/dev/null 2>&1; then
                    E2E_BROWSER_PATH="$(command -v chromium)"
                    export E2E_BROWSER_PATH
                  fi
                  cd "''${BUILD_ROOT:-$(pwd)}"
                  echo "==> Building E2E test server"
                  (cd e2e/server && go build -o /tmp/cqrs-htmx-e2e-server .)

                  echo "==> Starting E2E test server"
                  /tmp/cqrs-htmx-e2e-server &
                  SERVER_PID=$!

                  cleanup() {
                    kill "$SERVER_PID" 2>/dev/null || true
                    wait "$SERVER_PID" 2>/dev/null || true
                  }
                  trap cleanup EXIT

                  sleep 1

                  if ! curl -sf http://localhost:18923/ >/dev/null 2>&1; then
                    echo "FAIL: E2E server did not start on :18923"
                    exit 1
                  fi

                  echo "==> Running Playwright tests"
                  cd e2e

                  if command -v bun >/dev/null 2>&1; then
                    bun install --frozen-lockfile 2>/dev/null || bun install
                    bun run test
                  elif command -v pnpm dlx >/dev/null 2>&1; then
                    pnpm dlx playwright install chromium
                    pnpm dlx playwright test
                  else
                    echo "FAIL: Neither bun nor pnpm dlx found. Install Node.js or Bun to run E2E tests."
                    exit 1
                  fi
                '';
              };
            };

            check-phantom-version = {
              type = "app";
              meta.description = "Detect zero pseudo-versions + verify every internal require resolves to a PUBLISHED tag";
              program = pkgs.writeShellApplication {
                name = "check-phantom-version";
                runtimeInputs = [
                  pkgs.ripgrep
                  pkgs.git
                ];
                text = ''
                  set -euo pipefail
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  echo "=== Phantom Version Check ==="
                  echo "Scanning go.mod files for zero pseudo-versions..."
                  found=0
                  result=$(rg 'v0\.0\.0-00010101000000-000000000000' --glob 'go.mod' . 2>/dev/null || true)
                  if [ -n "$result" ]; then
                    echo "FAIL: zero pseudo-version detected:"
                    echo "$result"
                    found=1
                  fi
                  if [ "$found" -eq 0 ]; then
                    echo "OK: No phantom versions detected."
                  else
                    exit 1
                  fi
                  # The 2026-08-17..29 blind spot: totp/v4 v4.8.0 in
                  # examples/admin-demo was never published, poisoned the
                  # workspace graph, and only this app's zero-pseudo scan
                  # ran per-commit — it cannot see missing tags. Chain the
                  # real tag-existence gate here so the phantom gate covers
                  # both failure classes. CI stays advisory (ls-remote auth
                  # for private repos unverified there).
                  if [ "''${CI:-false}" != "true" ]; then
                    bash scripts/check-version-drift.sh --strict
                  else
                    bash scripts/check-version-drift.sh || true
                  fi
                '';
              };
            };

            check-cqrs-lint = {
              type = "app";
              meta.description = "Run cqrs-lint --strict on all workspace modules";
              program = pkgs.writeShellApplication {
                name = "check-cqrs-lint";
                text = ''
                  set -euo pipefail
                  # GOWORK=off: load each module from its own go.mod (published tags
                  # + module-level relative replaces) instead of the workspace —
                  # workspace-mode loading breaks when go.work's absolute-path
                  # sibling replaces point at in-flight go-cqrs-lite work.
                  export GOWORK=off
                  export GOEXPERIMENT=jsonv2
                  echo "=== cqrs-lint strict check ==="
                  fail=0
                  for mod in . identity-model usermgmt usermgmt/totp usermgmt/webauthn usermgmt/oauth2 adminui loginpage dashboardui datastar systemadapter health auditlog; do
                    echo "==> $mod"
                    if ! (cd "$mod" && cqrs-lint --strict . >/dev/null 2>&1); then
                      echo "FAIL: cqrs-lint findings in $mod (run 'cqrs-lint --strict --verbose .' for details)"
                      fail=1
                    fi
                  done
                  if [ "$fail" -eq 0 ]; then
                    echo "All modules pass cqrs-lint strict."
                  else
                    exit 1
                  fi
                '';
              };
            };
          };
        };
    };
}
