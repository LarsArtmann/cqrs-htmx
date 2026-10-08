{
  description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      # Inline systems: nixpkgs 26.11 dropped x86_64-darwin, which github:nix-systems/default still lists.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          # 2026-09-19: the fleet re-pin condition fired — go-cqrs-lite moved
          # ALL its modules to go 1.27.1 (upstream go-finding v1.11 pushed
          # module floors to >= 1.27.1) and go-etag v0.4.0 declares go 1.27.1
          # too. GOTOOLCHAIN=local (sandbox forbids toolchain downloads)
          # cannot satisfy those directives under go_1_26, so the toolchain
          # moves to nixpkgs' go_1_27 — the same change go-cqrs-lite landed
          # in its flake the same day.
          goPkg = pkgs.go_1_27;

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
              program = lib.getExe (
                pkgs.writeShellApplication {
                  inherit name;
                  runtimeInputs = [
                    goPkg
                    pkgs.jq
                  ]
                  ++ runtimeInputs;
                  text = goEnv + text;
                }
              );
            };
          # benchstat (golang.org/x/perf/cmd/benchstat) is not packaged in
          # nixpkgs; build it from the canonical googlesource repo.
          # To bump: update `rev` + `version`, then fix src.hash (hash.nix)
          # and vendorHash (vendorHash.nix) from the
          # `nix build .#benchstat` error messages.
          benchstat = pkgs.buildGoModule {
            pname = "benchstat";
            version = "0.0.0-20260825160852";

            src = pkgs.fetchgit {
              url = "https://go.googlesource.com/perf";
              rev = "19be9d8e6c701dc8ccabaad34bf705f773fd398b";
              hash = import ./hash.nix;
            };

            subPackages = [ "cmd/benchstat" ];
            vendorHash = import ./vendorHash.nix;
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
              # Aligned with the golangci-lint golines config (.golangci.yml
              # settings.golines.max-len: 120). The 2026-08-29 275-file churn
              # came from treefmt's default max-len of 100 reflowing every
              # 101-120-char line the lint gate had deliberately kept.
              golines.enable = true;
              shfmt.enable = true;
              shellcheck.enable = true;
            };
            settings.formatter.golines.options = [
              "--max-len"
              "120"
            ];
            # treefmt-nix's templ module wraps templ with nixpkgs' DEFAULT go
            # (1.26.x). Since the fleet re-pin moved the module floors to
            # 1.27.1 (2026-09-19), that go hits a GOTOOLCHAIN=auto download of
            # go1.27.1 inside the network-free check sandbox — templ fmt
            # hard-fails and takes the formatting/treefmt checks (and nix
            # buildflow steps) down with it. Pin the formatter's go to goPkg
            # + force offline env: 1.27.1 satisfies the floor locally and
            # GOPROXY=off makes import resolution fail fast so goimports
            # falls back to stdlib-only fixing (same behavior as pre-re-pin;
            # validated offline in-module-context: rc=0, changed=0).
            settings.formatter.templ.command = lib.mkForce (
              lib.getExe (
                pkgs.writeShellApplication {
                  name = "templ-fmt-offline";
                  runtimeInputs = [
                    goPkg
                    pkgs.templ
                  ];
                  text = ''
                    export GOTOOLCHAIN=local
                    export GOPROXY=off
                    exec templ "$@"
                  '';
                }
              )
            );
            # shfmt/shellcheck enabled 2026-08-30 after fixing all findings
            # in scripts/*.sh (dead vars, SC2155, SC1091, shfmt layout).
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
                # license-check: the nixpkgs go-licenses wrapper hard-exports
                # GOROOT of nixpkgs' DEFAULT go (1.26.x). go-webauthn
                # v0.18.2 imports crypto/mldsa (Go 1.27 std), so go/packages
                # dies with "package crypto/mldsa is not in std" and the
                # BuildFlow license-check step fails deterministically.
                # Bypass the wrapper: exec the wrapped binary with goPkg's
                # GOROOT and goPkg first on PATH (the go/packages driver
                # shells out to `go`).
                (pkgs.writeShellApplication {
                  name = "go-licenses";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export GOROOT="${goPkg}/share/go"
                    exec "${pkgs.go-licenses}/bin/.go-licenses-wrapped" "$@"
                  '';
                })
                pkgs.vulnix # NixOS vulnerability scan
                pkgs.deadnix # dead-code linter for .nix files (BuildFlow step)
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
            # Scoped formatting: `nix run .#fmt -- <paths...>` formats ONLY the
            # given paths with the repo's generated treefmt config. treefmt's
            # config is flake-generated into the store, so a bare `treefmt
            # <paths>` on the host fails ("no treefmt.toml"); this app forwards
            # the paths to the wrapped treefmt (config + project-root already
            # baked in). With no paths it behaves like `nix fmt`. Added
            # 2026-10-01 (TODO P3 h).
            fmt = {
              type = "app";
              meta.description = "Format ONLY the given paths with the repo treefmt config (e.g. `nix run .#fmt -- scripts/x.sh docs/y.md`)";
              program = pkgs.lib.getExe config.treefmt.build.wrapper;
            };

            bump-dep = {
              type = "app";
              meta.description = "Sweep a family dependency to one version across the workspace (tidy + go mod verify + build + vet per module); supports --dry-run / --commit / --no-verify";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "bump-dep";
                  runtimeInputs = [
                    goPkg
                    pkgs.git
                    pkgs.findutils
                    pkgs.gnugrep
                    pkgs.coreutils
                    pkgs.gawk
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/tools/bump-dep.sh "$@"
                  '';
                }
              );
            };

            test-bump-dep = {
              type = "app";
              meta.description = "Fixture self-test for bump-dep.sh discovery (block + single-line require forms, $ anchor, testdata skip, no-match)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-bump-dep";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.gnugrep
                    pkgs.coreutils
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-bump-dep.sh
                  '';
                }
              );
            };

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

            # Race tests over EVERY workspace module, including the e2e/examples
            # set that #test / #test-race deliberately exclude (8 modules). The
            # e2e/server module has no Go tests, so it is a clean no-op there.
            # Added 2026-10-01 (TODO P3 j). On-demand only — not wired into the
            # default gates because the embedded example servers bind ports.
            test-all = goApp {
              name = "run-tests-all";
              description = "Run ALL workspace-module tests with the race detector, including e2e/examples (the #test/#test-race exclusion set)";
              text = ''
                forEachGoModule "go test ./... -count=1 -race"
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
              description = "Run the setup spike benchmark (default 5x2s) and fail on a >10% median ns/op regression vs docs/benchmarks/setup-baseline.raw.txt (10% default: run-to-run noise on the pinned machine measures up to ~9% on the ~1us json-roundtrip sub-bench; tighten via BENCH_SPIKE_THRESHOLD). Per-bench overrides: BENCH_THRESHOLD_<NAME> with NAME = bench name uppercased, non-alnum -> _ (suffix match works, e.g. BENCH_THRESHOLD_JSON_ROUNDTRIP=15). Load guard: refuses to measure when 1-min load >= BENCH_MAX_LOAD (default cores/4; 0 disables) because concurrent builds make ns/op garbage. Env: BENCH_COUNT, BENCHTIME, BENCH_SPIKE_THRESHOLD, BENCH_MAX_LOAD, BENCH_THRESHOLD_*. --save-baseline [path] re-pins the baseline (records a load1 context header; re-pin when quiet-state medians drift >10% or after any machine change).";
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
                # shellcheck disable=SC1091
                source scripts/lib/go-cache-env.sh

                # Load guard: concurrent builds poison ns/op (two sessions in
                # a row burned bench runs on ~8.7 load before checking uptime).
                # Fail above BENCH_MAX_LOAD (default: cores/4); 0 disables.
                cores="$(nproc)"
                max_load="''${BENCH_MAX_LOAD:-$((cores / 4))}"
                load1="$(cut -d' ' -f1 /proc/loadavg)"
                if [ "$max_load" != "0" ]; then
                  ok="$(awk -v l="$load1" -v m="$max_load" 'BEGIN{print (l < m) ? 1 : 0}')"
                  if [ "$ok" != "1" ]; then
                    echo "bench-spike: 1-min load $load1 >= BENCH_MAX_LOAD $max_load (cores=$cores) — refusing to measure." >&2
                    echo "  Concurrent load makes ns/op comparisons garbage." >&2
                    echo "  Wait for a quiet machine, or override: BENCH_MAX_LOAD=<n> (0 disables the guard)." >&2
                    exit 2
                  fi
                  echo "bench-spike: load $load1 (max $max_load, cores $cores) — OK"
                fi

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
                  { echo "load1: $load1"; (cd setup && go test "''${bench_args[@]}" .); } | tee "$out"
                  echo "Saved $out — commit it so everyone gates against the same numbers."
                  echo "Re-pin policy: re-pin when quiet-state medians drift >10% from this baseline or after any machine change; the load1 header records the pinning context."
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

                # Per-bench threshold override: BENCH_THRESHOLD_<NAME> where
                # NAME is the benchmark name uppercased with non-alphanumerics
                # mapped to "_" (exact form), or any distinctive suffix of it
                # — BENCH_THRESHOLD_JSON_ROUNDTRIP=15 covers
                # BenchmarkSpikeBaselineVsAppkit/json-roundtrip. Without a
                # match, the global threshold applies. The ~1us sub-bench
                # flaps near any fixed percentage; this lets it have its own
                # budget without loosening the HTTP benches.
                bench_threshold() {
                  local name key var suffix val
                  name="$1"
                  key="$(printf '%s' "$name" | tr -c '[:alnum:]' '_' | tr '[:lower:]' '[:upper:]')"
                  var="BENCH_THRESHOLD_''${key}"
                  if [ -n "''${!var:-}" ]; then
                    val="''${!var}"
                  else
                    val=""
                    while IFS='=' read -r var _; do
                      [ -n "$var" ] || continue
                      suffix="''${var#BENCH_THRESHOLD_}"
                      [ -n "$suffix" ] || continue
                      case "$key" in
                        *"''${suffix}") val="''${!var}"; break ;;
                      esac
                    done < <(env | grep -E '^BENCH_THRESHOLD_[A-Za-z0-9_]+=' | sort)
                  fi
                  if [ -z "$val" ]; then
                    printf '%s' "$threshold"
                    return 0
                  fi
                  if ! printf '%s' "$val" | grep -Eq '^-?[0-9]+([.][0-9]+)?$'; then
                    echo "bench-spike: invalid threshold override $var='$val' (must be a number)" >&2
                    exit 2
                  fi
                  printf '%s' "$val"
                }

                regressed=0
                while read -r name old new; do
                  delta="$(awk -v o="$old" -v n="$new" 'BEGIN { printf "%+.1f", (n - o) / o * 100 }')"
                  t="$(bench_threshold "$name")"
                  printf '  %-50s median %10.1f -> %10.1f ns/op (%s%%, gate %s%%)\n' "$name" "$old" "$new" "$delta" "$t"
                  over="$(awk -v o="$old" -v n="$new" -v t="$t" 'BEGIN { print (((n - o) / o * 100) > t) ? 1 : 0 }')"
                  if [ "$over" = "1" ]; then
                    echo "  REGRESSION: '$name' regressed more than $t% vs baseline" >&2
                    regressed=1
                  fi
                done <<< "$joined"

                if [ "$regressed" -eq 1 ]; then
                  echo "bench-spike: FAIL (median ns/op regression over a per-bench or global threshold)" >&2
                  exit 1
                fi
                echo "bench-spike: OK (no median ns/op regression over each bench's gate)"
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
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
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
                      for pkg in display feedback forms htmx icons layout navigation; do
                        if [ -d "$TC_DIR/$pkg" ]; then
                          cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                        fi
                      done
                      # errorpage is a SEPARATE family module (not a package
                      # inside the root module dir) and defines its runtime
                      # class strings in styles.go (Go source, NOT .templ) —
                      # without both, the error-page utility families are
                      # silently absent from the bundle while the build stays
                      # green (dashboardui M6-class false green).
                      ERRORPAGE_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components/errorpage 2>/dev/null || true)
                      if [ -n "$ERRORPAGE_DIR" ] && [ -d "$ERRORPAGE_DIR" ]; then
                        cp "$ERRORPAGE_DIR/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                        cp "$ERRORPAGE_DIR/styles.go" "$SCAN_DIR/" 2>/dev/null || true
                      fi
                      echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS"
                    fi

                    tailwindcss -i "$TMP_CSS" -o assets/admin-tw.css --minify

                    rm -f "$TMP_CSS"
                    [ -n "''${SCAN_DIR:-}" ] && rm -rf "$SCAN_DIR"
                    echo "Done: adminui/assets/admin-tw.css"
                  '';
                }
              );
            };

            build-dashboardui-css = {
              type = "app";
              meta.description = "Compile dashboardui Tailwind v4 CSS (tailwind.css → assets/dashboard-tw.css)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "build-dashboardui-css";
                  runtimeInputs = [
                    pkgs.tailwindcss_4
                    goPkg
                  ];
                  text = ''
                    cd dashboardui
                    # Resolve the templ-components ROOT module dir at build
                    # time. dashboardui requires only the icons/utils
                    # submodules (the root module is not in the build list),
                    # so derive the root extraction path from the icons
                    # version: the module cache lays out
                    #   $GOMODCACHE/github.com/larsartmann/templ-components@<ver>/
                    #   $GOMODCACHE/github.com/larsartmann/templ-components/icons@<ver>/
                    # The icons dir's PARENT is NOT the root module (it holds
                    # versioned submodule dirs) — the 2026-09-17 false green
                    # came exactly from that fallback copying zero .templ
                    # files and Tailwind emitting zero library utilities.
                    ICONS_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components/icons 2>/dev/null || true)
                    TC_VERSION=$(basename "''${ICONS_DIR:-}" | sed -n 's/^icons@//p')
                    if [ -z "$TC_VERSION" ]; then
                      echo "ERROR: cannot resolve templ-components/icons via 'go list -m' in dashboardui/ (run 'go mod download' there first)" >&2
                      exit 1
                    fi
                    TC_DIR="$(go env GOMODCACHE)/github.com/larsartmann/templ-components@$TC_VERSION"
                    if [ ! -d "$TC_DIR" ]; then
                      GOWORK=off go mod download "github.com/larsartmann/templ-components@$TC_VERSION" >&2 || true
                    fi
                    if [ ! -d "$TC_DIR" ]; then
                      echo "ERROR: templ-components root module not extracted at $TC_DIR (download failed)" >&2
                      exit 1
                    fi

                    TMP_CSS=$(mktemp --suffix=.css)
                    cp tailwind.css "$TMP_CSS"

                    # Since the 2026-09-22 full-templ migration dashboardui has
                    # its own .templ files; the library's components still emit
                    # Tailwind utilities at RUNTIME (class strings in library
                    # sources only), so scan BOTH the module's .templ files and
                    # the library's .templ + *_go.go class-map sources (NOT the
                    # 3x-larger _templ.go mirrors). errorpage is a separate
                    # family module — included below for the amber family.
                    SCAN_DIR=$(mktemp -d)
                    cp ./*.templ "$SCAN_DIR/" 2>/dev/null || true
                    for pkg in display feedback forms htmx icons layout navigation utils recipes; do
                      if [ -d "$TC_DIR/$pkg" ]; then
                        cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                        # Class maps for enums/variants live in the *_go.go
                        # sources (button_go.go holds every Button variant), NOT
                        # in .templ — without them whole variant families are
                        # silently missing from the bundle (found with
                        # ring-blue-300 after the M10 button swap).
                        cp "$TC_DIR/$pkg/"*_go.go "$SCAN_DIR/" 2>/dev/null || true
                      fi
                    done
                    ERRORPAGE_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components/errorpage 2>/dev/null || true)
                    if [ -n "$ERRORPAGE_DIR" ] && [ -d "$ERRORPAGE_DIR" ]; then
                      cp "$ERRORPAGE_DIR/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                      # errorpage defines its runtime class strings in
                      # styles.go (Go source, NOT .templ) — without copying
                      # it the amber utility family is silently absent from
                      # the bundle while the build stays green (M6-class
                      # false green, found 2026-09-17).
                      cp "$ERRORPAGE_DIR/styles.go" "$SCAN_DIR/" 2>/dev/null || true
                    fi
                    if [ -z "$(find "$SCAN_DIR" -name '*.templ' -print -quit)" ]; then
                      echo "ERROR: zero .templ files copied from $TC_DIR — the @source scan would be empty (false-green guard)" >&2
                      rm -f "$TMP_CSS"; rm -rf "$SCAN_DIR"
                      exit 1
                    fi
                    echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS"

                    tailwindcss -i "$TMP_CSS" -o assets/dashboard-tw.css --minify

                    rm -f "$TMP_CSS"
                    rm -rf "$SCAN_DIR"

                    # Canary: the output MUST contain utilities that adopted
                    # components emit at runtime (display.StatusBadge's badge
                    # classes live only in library .templ files) plus dark:
                    # variants. Absence means the @source scan failed — a
                    # utility-free stylesheet passed to users (exit 0 false
                    # green). Fail loudly instead. NOTE: minified CSS escapes
                    # the variant as dark\: — the canary greps the escaped
                    # form.
                    for needle in 'bg-green-100' 'bg-blue-100' 'bg-green-900' 'dark\\:' 'bg-amber-50' 'bg-amber-100' 'border-amber-200' 'bg-amber-900'; do
                      if ! grep -q "$needle" assets/dashboard-tw.css; then
                        echo "ERROR: canary '$needle' missing from assets/dashboard-tw.css — Tailwind scanned no templ-components source" >&2
                        exit 1
                      fi
                    done
                    echo "Done: dashboardui/assets/dashboard-tw.css ($(wc -c < assets/dashboard-tw.css) bytes, canaries OK)"
                  '';
                }
              );
            };

            build-setup-demo-css = {
              type = "app";
              meta.description = "Compile the setup-demo consumer stylesheet (tailwind.css → assets/app.css) that the login page loads at /app.css";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "build-setup-demo-css";
                  runtimeInputs = [
                    pkgs.tailwindcss_4
                    goPkg
                  ];
                  text = ''
                    cd examples/setup-demo
                    # The demo's login page (via setup → loginpage) renders
                    # templ-components Tailwind utilities. This app compiles
                    # the CONSUMER-side stylesheet — the same job every real
                    # setup consumer must do once (README "Styling").
                    #
                    # Scan sources (NOT the 3x-larger _templ.go mirrors):
                    #   - loginpage: page.templ + assets/login.js (the JS
                    #     injects lp-spinner/animate-spin classes at runtime)
                    #   - templ-components: the 5 packages loginpage imports,
                    #     .templ + *_go.go (variant class maps live in the
                    #     *_go.go sources — the M10 lesson)
                    LP_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/cqrs-htmx/loginpage/v4 || true)
                    if [ -z "$LP_DIR" ]; then
                      echo "ERROR: cannot resolve loginpage via 'go list -m' in examples/setup-demo (run 'go mod download' there first)" >&2
                      exit 1
                    fi
                    TC_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components || true)
                    if [ -z "$TC_DIR" ]; then
                      echo "ERROR: cannot resolve templ-components via 'go list -m' in examples/setup-demo" >&2
                      exit 1
                    fi

                    TMP_CSS=$(mktemp --suffix=.css)
                    cp tailwind.css "$TMP_CSS"
                    echo "@source \"$LP_DIR\";" >> "$TMP_CSS"

                    SCAN_DIR=$(mktemp -d)
                    for pkg in layout forms display feedback utils; do
                      if [ -d "$TC_DIR/$pkg" ]; then
                        cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                        cp "$TC_DIR/$pkg/"*_go.go "$SCAN_DIR/" 2>/dev/null || true
                      fi
                    done
                    if [ -z "$(find "$SCAN_DIR" -name '*.templ' -print -quit)" ]; then
                      echo "ERROR: zero .templ files copied from $TC_DIR — the @source scan would be empty (false-green guard)" >&2
                      rm -f "$TMP_CSS"; rm -rf "$SCAN_DIR"
                      exit 1
                    fi
                    echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS"

                    mkdir -p assets
                    tailwindcss -i "$TMP_CSS" -o assets/app.css --minify

                    rm -f "$TMP_CSS"; rm -rf "$SCAN_DIR"

                    # Canaries: utilities the login page emits at runtime —
                    # library button/input classes, dark: variants (escaped
                    # form in minified CSS), and the JS-injected spinner.
                    for needle in 'animate-spin' 'text-blue-600' 'bg-gray-900' 'dark\\:' 'rounded-md'; do
                      if ! grep -q "$needle" assets/app.css; then
                        echo "ERROR: canary '$needle' missing from assets/app.css — Tailwind scanned no loginpage/templ-components source" >&2
                        exit 1
                      fi
                    done
                    echo "Done: examples/setup-demo/assets/app.css ($(wc -c < assets/app.css) bytes, canaries OK)"
                  '';
                }
              );
            };

            gen = {
              type = "app";
              meta.description = "Regenerate adminui + loginpage + dashboardui templ components (module-dir generation is canonical) and normalize formatting";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "templ-generate";
                  runtimeInputs = [
                    goPkg
                    pkgs.templ
                  ];
                  text = ''
                    (cd adminui && templ generate && gofmt -w ./*_templ.go)
                    (cd loginpage && templ generate && gofmt -w ./*_templ.go)
                    (cd dashboardui && templ generate && gofmt -w ./*_templ.go)
                    echo "Done: adminui + loginpage + dashboardui templ components regenerated (module-dir, bare FileName) and formatted"
                  '';
                }
              );
            };

            render-diagrams = {
              type = "app";
              meta.description = "Render all .d2 source files under docs/ to SVG (dark canvas → theme 200, light → default)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
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
                }
              );
            };

            errorfamily = {
              type = "app";
              meta.description = "Verify all errors use go-error-family constructors (no stdlib errors.New/fmt.Errorf/errors.Join)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
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
                      go run scripts/checks/errorfamily_scanner.go "$dir"
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
                }
              );
            };

            check-modules = {
              type = "app";
              meta.description = "Run all module architecture checks (isolation, dep budgets, version drift, release train, replaces, docs). Default: abort at first red. --report: run every stage and print a red/green summary (exit 1 if any failed)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-modules";
                  runtimeInputs = [
                    goPkg
                    pkgs.python3
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    if [ "''${1:-}" = "--report" ]; then
                      stages=(
                        "module-isolation:bash scripts/checks/check-module-isolation.sh"
                        "dep-budgets:bash scripts/checks/check-dep-budgets.sh"
                        "dep-budgets-self-test:bash scripts/selftests/test-check-dep-budgets.sh"
                        "go-toolchain:bash scripts/checks/check-go-toolchain.sh"
                        "workspace-build:bash scripts/checks/check-workspace-build.sh"
                        "workspace-build-self-test:bash scripts/selftests/test-check-workspace-build.sh"
                        "version-drift:bash scripts/checks/check-version-drift.sh --strict"
                        "release-train:bash scripts/checks/check-release-train.sh"
                        "release-train-self-test:bash scripts/selftests/test-check-release-train.sh"
                        "vcs-cache:bash scripts/checks/check-vcs-cache.sh"
                        "vcs-cache-self-test:bash scripts/selftests/test-check-vcs-cache.sh"
                        "css-bundles:bash scripts/checks/check-css-bundles.sh"
                        "css-bundles-self-test:bash scripts/selftests/test-check-css-bundles.sh"
                        "css-bundle-classes:bash scripts/checks/check-css-bundle-classes.sh"
                        "css-bundle-classes-self-test:bash scripts/selftests/test-check-css-bundle-classes.sh"
                        "preflight-self-test:bash scripts/selftests/test-preflight-tree-check.sh"
                        "wait-tree-quiet-self-test:bash scripts/selftests/test-wait-tree-quiet.sh"
                        "train-preflight-self-test:bash scripts/selftests/test-train-preflight.sh"
                        "replace-directives:bash scripts/checks/check-replace-directives.sh"
                        "docs-freshness:bash scripts/checks/check-docs-freshness.sh"
                        "docs-freshness-self-test:bash scripts/selftests/test-check-docs-freshness.sh"
                        "docs-links:bash scripts/checks/check-docs-links.sh"
                        "status-annotations:bash scripts/checks/check-status-annotations.sh"
                        "status-annotations-self-test:bash scripts/selftests/test-check-status-annotations.sh"
                        "status-rows:python3 scripts/checks/check-status-rows.py"
                        "status-rows-self-test:bash scripts/selftests/test-check-status-rows.sh"
                        "status-rows-normalize-self-test:bash scripts/selftests/test-normalize-status-rows.sh"
                        "bump-dep-self-test:bash scripts/selftests/test-bump-dep.sh"
                        "erraudit-inventory-self-test:bash scripts/selftests/test-erraudit-inventory.sh"
                        "cqrs-lint-gate-self-test:bash scripts/selftests/test-check-cqrs-lint.sh"
                        "session-route-wrappers:bash scripts/checks/check-session-route-wrappers.sh"
                        "session-route-wrappers-self-test:bash scripts/selftests/test-check-session-route-wrappers.sh"
                        "streamid-identity:bash scripts/checks/check-streamid-identity.sh"
                        "streamid-identity-self-test:bash scripts/selftests/test-check-streamid-identity.sh"
                        "family-release-consumable:bash scripts/checks/check-family-release-consumable.sh"
                        "family-release-consumable-self-test:bash scripts/selftests/test-check-family-release-consumable.sh"
                        "selftest-env-leaks:bash scripts/checks/check-selftest-env-leaks.sh"
                        "selftest-env-leaks-self-test:bash scripts/selftests/test-check-selftest-env-leaks.sh"
                        "branching-flow:bash scripts/checks/check-branching-flow.sh"
                        "branching-flow-self-test:bash scripts/selftests/test-check-branching-flow.sh"
                      )
                      red=0
                      for stage in "''${stages[@]}"; do
                        name="''${stage%%:*}"
                        cmd="''${stage#*:}"
                        if output=$($cmd 2>&1); then
                          echo "  ✅ $name"
                        else
                          echo "  ❌ $name"
                          printf '%s\n' "$output" | sed 's/^/      /' | tail -15
                          red=$((red + 1))
                        fi
                      done
                      echo ""
                      if [ "$red" -gt 0 ]; then
                        echo "✗ $red of ''${#stages[@]} module architecture checks failed"
                        exit 1
                      fi
                      echo "✓ All ''${#stages[@]} module architecture checks passed"
                      exit 0
                    fi
                    bash scripts/checks/check-module-isolation.sh
                    bash scripts/checks/check-dep-budgets.sh
                    bash scripts/selftests/test-check-dep-budgets.sh
                    bash scripts/checks/check-go-toolchain.sh
                    bash scripts/checks/check-workspace-build.sh
                    bash scripts/selftests/test-check-workspace-build.sh
                    bash scripts/checks/check-version-drift.sh --strict
                    bash scripts/checks/check-release-train.sh
                    bash scripts/checks/check-vcs-cache.sh
                    bash scripts/selftests/test-check-vcs-cache.sh
                    bash scripts/checks/check-css-bundles.sh
                    bash scripts/selftests/test-check-css-bundles.sh
                    bash scripts/checks/check-css-bundle-classes.sh
                    bash scripts/selftests/test-check-css-bundle-classes.sh
                    bash scripts/checks/check-replace-directives.sh
                    bash scripts/checks/check-docs-freshness.sh
                    bash scripts/selftests/test-check-docs-freshness.sh
                    bash scripts/checks/check-docs-links.sh
                    bash scripts/checks/check-status-annotations.sh
                    bash scripts/selftests/test-check-status-annotations.sh
                    python3 scripts/checks/check-status-rows.py
                    bash scripts/selftests/test-check-status-rows.sh
                    bash scripts/selftests/test-normalize-status-rows.sh
                    bash scripts/selftests/test-bump-dep.sh
                    bash scripts/checks/check-session-route-wrappers.sh
                    bash scripts/selftests/test-check-session-route-wrappers.sh
                    bash scripts/checks/check-branching-flow.sh
                    bash scripts/selftests/test-check-branching-flow.sh
                    echo ""
                    echo "✓ All module architecture checks passed"
                  '';
                }
              );
            };

            check-docs-freshness = {
              type = "app";
              meta.description = "Scan .md files for version strings that don't match go.mod";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-docs-freshness";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-docs-freshness.sh
                  '';
                }
              );
            };

            check-docs-links = {
              type = "app";
              meta.description = "Check all markdown file-path links resolve correctly";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-docs-links";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.gnugrep
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-docs-links.sh
                  '';
                }
              );
            };

            check-status-annotations = {
              type = "app";
              meta.description = "Presence gate for archived status-report annotations (inline strikethrough + dated blockquote since 2026-09-09; older reports LEGACY-EXEMPT)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-status-annotations";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.gnugrep
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-status-annotations.sh
                  '';
                }
              );
            };

            check-session-route-wrappers = {
              type = "app";
              meta.description = "ADR-0055 invariant gate: every /auth/* route whose handler reads the session context must be registered behind h.withSession — a bare context-only mount is a dead route, an ungated ceremony is a takeover hole";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-session-route-wrappers";
                  runtimeInputs = [
                    pkgs.gnugrep
                    pkgs.gawk
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-session-route-wrappers.sh
                  '';
                }
              );
            };

            check-streamid-identity = {
              type = "app";
              meta.description = "Gotcha-25/D12 invariant gate: StreamID().String() is display-form (StreamMarker-branded under the in-flight id train), never identity — every occurrence must sit in the pinned display allowlist; identity positions use .Get()";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-streamid-identity";
                  runtimeInputs = [
                    pkgs.gnugrep
                    pkgs.gawk
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-streamid-identity.sh
                  '';
                }
              );
            };

            check-family-release-consumable = {
              type = "app";
              meta.description = "R2 guard: reject upstream tags whose go.mod carries placeholder pseudo-version requires (vX.Y.Z-00010101000000-...) — the templ-components v1.20.0 poison class that blocks every consumer push with a misleading unknown revision";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-family-release-consumable";
                  runtimeInputs = [
                    pkgs.curl
                    pkgs.gnugrep
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-family-release-consumable.sh "$@"
                  '';
                }
              );
            };

            check-selftest-env-leaks = {
              type = "app";
              meta.description = "Gotcha-27c audit mechanized: every self-tested checker that branches on CI must strip CI (env -u CI) in its fixture self-test — GitHub runners export CI=true globally, so unguarded self-tests cover the WRONG branch";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-selftest-env-leaks";
                  runtimeInputs = [
                    pkgs.gnugrep
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-selftest-env-leaks.sh
                  '';
                }
              );
            };

            ci-parity-check = {
              type = "app";
              meta.description = "Gotcha-27c as one command: CI=true fixture self-tests + per-go.mod tidy -diff + lint + test + check-modules — the local battery that surfaces ALL reds in one pass before a push";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "ci-parity-check";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.findutils
                    pkgs.bash
                    pkgs.nix
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/tools/ci-parity-check.sh "$@"
                  '';
                }
              );
            };

            check-status-rows = {
              type = "app";
              meta.description = "Row-integrity gate for archived status reports: PARTIAL rows (cells disagree within one row) fail; deliberately-mixed tables are counted, not failed";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-status-rows";
                  runtimeInputs = [
                    pkgs.python3
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    python3 scripts/checks/check-status-rows.py
                  '';
                }
              );
            };

            test-status-rows = {
              type = "app";
              meta.description = "Fixture self-test for check-status-rows.py (12 cases: partial/mixed/code-span/header-only/dir/missing)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-status-rows";
                  runtimeInputs = [
                    pkgs.python3
                    pkgs.coreutils
                    pkgs.gnugrep
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-status-rows.sh
                  '';
                }
              );
            };

            normalize-status-rows = {
              type = "app";
              meta.description = "Fixer for PARTIAL table rows in archived status reports (whole-row strike). Companion to check-status-rows; pass --dry-run to list without writing";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "normalize-status-rows";
                  runtimeInputs = [
                    pkgs.python3
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    python3 scripts/tools/normalize-status-rows.py "$@"
                  '';
                }
              );
            };

            test-normalize-status-rows = {
              type = "app";
              meta.description = "Fixture self-test for normalize-status-rows.py (5 cases: whole-row strike, clean/STRUCK untouched, code-span literal, idempotent, dry-run)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-normalize-status-rows";
                  runtimeInputs = [
                    pkgs.python3
                    pkgs.coreutils
                    pkgs.gnugrep
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-normalize-status-rows.sh
                  '';
                }
              );
            };

            check-docs-tail-budget = {
              type = "app";
              meta.description = "ADVISORY: warn when the live docs/status/*.md tail exceeds 3 reports (--strict to exit 1). Deliberately not a blocking check-modules stage";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-docs-tail-budget";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-docs-tail-budget.sh "$@"
                  '';
                }
              );
            };

            test-check-docs-tail-budget = {
              type = "app";
              meta.description = "Fixture self-test for check-docs-tail-budget.sh (5 cases: within/over/advisory/strict/README-excluded/budget-override)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-check-docs-tail-budget";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.coreutils
                    pkgs.gnugrep
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-docs-tail-budget.sh
                  '';
                }
              );
            };

            test-status-annotations = {
              type = "app";
              meta.description = "Fixture self-test for check-status-annotations.sh (16 cases: gated/legacy/epoch-boundary/undated/missing-dir)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-status-annotations";
                  runtimeInputs = [
                    pkgs.findutils
                    pkgs.gnugrep
                    pkgs.coreutils
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-status-annotations.sh
                  '';
                }
              );
            };

            test-dep-budgets = {
              type = "app";
              meta.description = "Fixture self-test for check-dep-budgets.sh (pins the comment-line/indirect exclusion in the dep-counting awk; 2026-09-22 regression class)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-dep-budgets";
                  runtimeInputs = [
                    pkgs.gawk
                    pkgs.gnugrep
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-dep-budgets.sh
                  '';
                }
              );
            };

            check-vcs-cache = {
              type = "app";
              meta.description = "GOPRIVATE VCS-cache health: every bare repo under GOMODCACHE/cache/vcs must carry remote.origin.url (the broken-origin class masquerades as code bugs)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-vcs-cache";
                  runtimeInputs = [
                    goPkg
                    pkgs.git
                    pkgs.coreutils
                    pkgs.gnugrep
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-vcs-cache.sh
                  '';
                }
              );
            };

            test-vcs-cache = {
              type = "app";
              meta.description = "Fixture self-test for check-vcs-cache.sh (healthy bare repo / origin-less corruption / missing cache dir)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-vcs-cache";
                  runtimeInputs = [
                    pkgs.git
                    pkgs.coreutils
                    pkgs.gnugrep
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-vcs-cache.sh
                  '';
                }
              );
            };

            check-workspace-build = {
              type = "app";
              meta.description = "Workspace-mode go build ./... in every go.work member — the complement to the hermetic per-module gates; catches go.work use/replace mangling (the 373209a7 ambiguous-import class) that GOWORK=off gates are blind to";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-workspace-build";
                  runtimeInputs = [
                    goPkg
                    pkgs.coreutils
                    pkgs.gawk
                  ];
                  text = ''
                    export GOTOOLCHAIN=local
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-workspace-build.sh
                  '';
                }
              );
            };

            test-workspace-build = {
              type = "app";
              meta.description = "Fixture self-test for check-workspace-build.sh (healthy workspace / dropped-use mangling / missing go.work)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-workspace-build";
                  runtimeInputs = [
                    goPkg
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-workspace-build.sh
                  '';
                }
              );
            };

            erraudit-inventory = {
              type = "app";
              meta.description = "Per-module erraudit critical-findings inventory across the whole go.work workspace (report mode; --gate fails on TOTAL>0); prints its candidate count and fails on a zero-module decomposition";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "erraudit-inventory";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.gawk
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/erraudit-inventory.sh "$@"
                  '';
                }
              );
            };

            test-erraudit-inventory = {
              type = "app";
              meta.description = "Offline fixture self-test for erraudit-inventory.sh (stubbed binary: counting, gate mode, zero-candidate guard, missing-binary guard, erraudit-failure guard)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-erraudit-inventory";
                  runtimeInputs = [
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-erraudit-inventory.sh
                  '';
                }
              );
            };

            check-css-bundles = {
              type = "app";
              meta.description = "Sanity gate for the committed Tailwind bundles: minified 1-line form, banner header, byte floor, canary utilities (catches the hook-rewrite and empty-scan corruption classes)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-css-bundles";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.gnugrep
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-css-bundles.sh
                  '';
                }
              );
            };

            test-css-bundles = {
              type = "app";
              meta.description = "Fixture self-test for check-css-bundles.sh (pristine / missing / pretty rewrite / near-empty truncation / canary-stripped / empty)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-css-bundles";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.gnugrep
                    pkgs.python3
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-css-bundles.sh
                  '';
                }
              );
            };

            check-css-bundle-classes = {
              type = "app";
              meta.description = "Exact-class-set drift gate: sorted class tokens of the committed Tailwind bundles must equal a fresh canonical build (mechanizes the formatting-only proof family bumps need)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-css-bundle-classes";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.gnugrep
                    pkgs.git
                    pkgs.gnused
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-css-bundle-classes.sh
                  '';
                }
              );
            };

            test-css-bundle-classes = {
              type = "app";
              meta.description = "Fixture self-test for check-css-bundle-classes.sh (identical / missing class / extra class / missing fixture / formatting-only)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-css-bundle-classes";
                  runtimeInputs = [
                    pkgs.coreutils
                    pkgs.gnugrep
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-css-bundle-classes.sh
                  '';
                }
              );
            };

            preflight-tree-check = {
              type = "app";
              meta.description = "Abort-on-surprise gate before tree-mutating batch steps in the shared tree (dirty tree / fresh non-daemon commit / commit-velocity burst)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "preflight-tree-check";
                  runtimeInputs = [
                    pkgs.git
                    pkgs.coreutils
                    pkgs.gnused
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/tools/preflight-tree-check.sh
                  '';
                }
              );
            };

            wait-tree-quiet = {
              type = "app";
              meta.description = "Block until the shared tree is quiet: clean tree + stable HEAD for a full window (quiescence gate before push retries / release trains)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "wait-tree-quiet";
                  runtimeInputs = [
                    pkgs.git
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/tools/wait-tree-quiet.sh "$@"
                  '';
                }
              );
            };

            train-preflight = {
              type = "app";
              meta.description = "ONE command before a train: tree quiescence + surprise check + lint + hermetic test battery + strict release-train gate, with candidate-count guards (a zero-iteration stage is a false green)";
              # pkgs.nix: the lint/test stages invoke `nix run .#lint` / `.#test`
              # from inside the app; without nix on PATH they fail with a
              # confusing command-not-found after the slow stages.
              runtimeInputs = [
                pkgs.nix
                pkgs.git
                pkgs.coreutils
              ];
              text = ''
                cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                bash scripts/checks/train-preflight.sh "$@"
              '';
            };

            test-train-preflight = {
              type = "app";
              meta.description = "Fixture self-test for train-preflight.sh (offline stubs: all-green / stage failure / guard misses / keep-going aggregate / outside-repo)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-train-preflight";
                  runtimeInputs = [
                    pkgs.git
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-train-preflight.sh
                  '';
                }
              );
            };

            check-release-train = {
              type = "app";
              meta.description = "Verify every internal require resolves to a PUBLISHED tag; list train-lag for the next family train (forwards flags: --json, --strict-lag N, --no-cache, --refresh-cache)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
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
                    bash scripts/checks/check-release-train.sh "$@"
                  '';
                }
              );
            };

            check-go-toolchain = {
              type = "app";
              meta.description = "Fail when go.work's go directive is newer than the flake's nixpkgs Go toolchain";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-go-toolchain";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-go-toolchain.sh
                  '';
                }
              );
            };

            check-codegen = {
              type = "app";
              meta.description = "Verify adminui + loginpage + dashboardui _templ.go files match .templ sources (no codegen drift; module-dir generation is canonical)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-codegen";
                  runtimeInputs = [
                    goPkg
                    pkgs.templ
                  ];
                  text = ''
                    for mod in adminui loginpage dashboardui; do
                      echo "==> $mod"
                      (cd "$mod" && templ generate && gofmt -w ./*_templ.go)
                      if ! git diff --exit-code -- "$mod"/*_templ.go; then
                        echo ""
                        echo "FAIL: Generated _templ.go files in $mod differ from committed versions."
                        echo "Run 'nix run .#gen' (module-dir generation with bare FileName: is canonical) and commit the result."
                        exit 1
                      fi
                    done
                    echo "Codegen drift check PASSED"
                  '';
                }
              );
            };

            check-templates = {
              type = "app";
              meta.description = "Verify //go:build ignore SQL setup template files compile (sqlite/postgres/mysql)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-templates";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    bash scripts/checks/check-templates.sh
                  '';
                }
              );
            };

            coverage-gate = goApp {
              name = "coverage-gate";
              description = "Run tests and fail if coverage drops below thresholds";
              runtimeInputs = [ pkgs.bc ];
              text = ''
                fail=0
                check_cov() {
                  local mod="$1" threshold="$2"
                  local cov profile
                  profile=$(mktemp /tmp/cov.XXXXXX)
                  cov=$(cd "$mod" && go test ./... -count=1 -coverprofile="$profile" >/dev/null 2>&1 && go tool cover -func="$profile" | tail -1 | grep -oP '\d+\.\d+(?=%)')
                  rm -f "$profile"
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
                core_profile=$(mktemp /tmp/corecov.XXXXXX)
                core_cov=$(cd dashboardui && go test ./core/... -count=1 -coverprofile="$core_profile" >/dev/null 2>&1 && go tool cover -func="$core_profile" | tail -1 | grep -oP '\d+\.\d+(?=%)')
                rm -f "$core_profile"
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
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "release-checklist";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    bash scripts/tools/release-checklist.sh
                  '';
                }
              );
            };

            e2e = {
              type = "app";
              meta.description = "Run Playwright E2E tests (offline sync) against the local Go test server";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "e2e";
                  runtimeInputs = [
                    goPkg
                    pkgs.nodejs
                    pkgs.curl
                  ]
                  ++ pkgs.lib.optional (pkgs ? chromium) pkgs.chromium;
                  text = ''
                    export GOEXPERIMENT=jsonv2
                    # Playwright must download browsers to a WRITABLE path. The
                    # ambient value may point at a dead mount (e.g. the /mnt/buildcache
                    # sda1 on the 2026-08 machine) — fall back to /tmp when the
                    # configured path cannot be created.
                    if ! mkdir -p "''${PLAYWRIGHT_BROWSERS_PATH:-/tmp/pw-browsers}" 2>/dev/null; then
                      export PLAYWRIGHT_BROWSERS_PATH=/tmp/pw-browsers
                      mkdir -p "$PLAYWRIGHT_BROWSERS_PATH"
                    fi
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
                      # Auto-provision the browser when no Nix/system Chromium
                      # was found (no-op fast path when already installed).
                      if [ -z "''${E2E_BROWSER_PATH:-}" ]; then
                        bun x playwright install chromium ffmpeg
                      fi
                      bun run test
                    elif command -v pnpm dlx >/dev/null 2>&1; then
                      pnpm dlx playwright install chromium
                      pnpm dlx playwright test
                    else
                      echo "FAIL: Neither bun nor pnpm dlx found. Install Node.js or Bun to run E2E tests."
                      exit 1
                    fi
                  '';
                }
              );
            };

            check-require-tags = {
              type = "app";
              meta.description = "Detect zero pseudo-versions + verify every internal require resolves to a PUBLISHED tag (strict locally, advisory under CI)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-require-tags";
                  runtimeInputs = [
                    pkgs.ripgrep
                    pkgs.git
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-require-tags.sh
                  '';
                }
              );
            };

            check-phantom-version = {
              type = "app";
              meta.description = "DEPRECATED name for check-require-tags (the gate enforces tag existence now, not just pseudo-versions)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-phantom-version";
                  runtimeInputs = [ pkgs.git ];
                  text = ''
                    echo "check-phantom-version is deprecated; run .#check-require-tags (the gate enforces tag existence now, not just pseudo-versions)" >&2
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-require-tags.sh
                  '';
                }
              );
            };

            check-cqrs-lint = {
              type = "app";
              meta.description = "Run cqrs-lint --strict --fail-on-stale-suppressions on all workspace modules (gate logic in scripts/checks/check-cqrs-lint.sh; local-only — CI has no cqrs-lint until the Go-installable distribution)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-cqrs-lint";
                  # goPkg: cqrs-lint shells out to `go list` for package loading —
                  # without the 1.27.1 toolchain in PATH it resolves the ambient
                  # go 1.26.7 (GOTOOLCHAIN=local), which cannot load modules
                  # requiring go >= 1.27.1, and EVERY module fails with load errors.
                  runtimeInputs = [ goPkg ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/checks/check-cqrs-lint.sh
                  '';
                }
              );
            };

            test-check-cqrs-lint = {
              type = "app";
              meta.description = "Fixture self-test for check-cqrs-lint.sh (offline: clean passes / syntax-broken fails under strict / mixed sweep / candidate count / flag wiring pin)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-check-cqrs-lint";
                  runtimeInputs = [
                    goPkg
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-cqrs-lint.sh
                  '';
                }
              );
            };

            check-branching-flow = {
              type = "app";
              meta.description = "branching-flow (go-design-smells) ratchet gate: fail only on NEW findings vs the committed SARIF baseline (docs/analysis/). Local-only — CI runners have no branching-flow binary (self-skips there; the fixture self-test is the CI coverage)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "check-branching-flow";
                  # goPkg: the experimental panic linter shells out to `go` for
                  # package loading — the ambient go (GOTOOLCHAIN=local by
                  # default here) cannot load the 1.27.1 workspace otherwise
                  # and every run dies rc=69 (same trap class as cqrs-lint).
                  runtimeInputs = [ goPkg ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    export GOTOOLCHAIN=local
                    export GOEXPERIMENT=jsonv2
                    bash scripts/checks/check-branching-flow.sh "$@"
                  '';
                }
              );
            };

            test-check-branching-flow = {
              type = "app";
              meta.description = "Fixture self-test for check-branching-flow.sh (offline: missing-binary guard local-fail/CI-skip, missing + uncommitted + untracked baseline guards, rc pass-through 0/1/69, --baseline/--exit-code flag wiring pin)";
              program = pkgs.lib.getExe (
                pkgs.writeShellApplication {
                  name = "test-check-branching-flow";
                  runtimeInputs = [
                    pkgs.git
                    pkgs.coreutils
                  ];
                  text = ''
                    cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                    bash scripts/selftests/test-check-branching-flow.sh
                  '';
                }
              );
            };
          };
        };
    };
}
